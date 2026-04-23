// Package auth issues and verifies short-lived confirmation tokens for
// destructive verbs. Per docs/threat-model.md, destructive ops cannot rely
// on TTY prompts (agents cannot use them). Instead a human generates a
// token via `coily auth issue --ttl 5m --scope ...` and passes it to the
// mutating verb at invocation time.
//
// A token is an opaque base64url string. Internally it encodes:
//
//	scope | issued-at | ttl-seconds | hmac-sha256(scope|issued|ttl)
//
// scope is a comma-separated list of one or more scope strings shaped as
// `<binary>.<service>:<bucket>` (validated by pkg/policy.ValidateScopeList
// at issuance time). The signing key is loaded from a root-owned path
// configured in config.tokens.issuer_key_path. If the key file is missing,
// Issue creates one on the first call with 0600 perms and a
// cryptographically random 32-byte payload. Verify reads the same path.
//
// An agent running as the coily process user can read the key, which is
// fine: the point of the token is not to prevent a local attacker from
// forging one (that threat is already outside scope of coily). It is to
// force every destructive op through an explicit, scoped, human-initiated
// issuance step that shows up in the audit log.
package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/coilysiren/coily/pkg/policy"
)

// ErrInvalidToken is returned by Verify when the token is malformed, its
// signature does not match, the scope does not match, or it has expired.
var ErrInvalidToken = errors.New("auth: invalid token")

// ErrKeyPathUnset is returned when Issue/Verify is called without a key path.
var ErrKeyPathUnset = errors.New("auth: key path not configured")

// Issuer issues and verifies tokens. Keyed by a secret on disk.
type Issuer struct {
	// KeyPath is where the HMAC key lives. Created on first use if missing.
	KeyPath string
	// Now overrides the clock for tests.
	Now func() time.Time
}

// NewIssuer constructs an Issuer with Now set to time.Now.
func NewIssuer(keyPath string) *Issuer {
	return &Issuer{KeyPath: keyPath, Now: time.Now}
}

func (i *Issuer) now() time.Time {
	if i.Now == nil {
		return time.Now()
	}
	return i.Now()
}

// Issue returns an opaque token for scope that remains valid for ttl. scope
// is a comma-separated list of one or more scope strings. Validation of the
// scope shape is left to the caller (see policy.ValidateScopeList). Issue
// itself only rejects empty scopes and scopes containing the internal
// delimiter `|`.
func (i *Issuer) Issue(scope string, ttl time.Duration) (string, error) {
	if i.KeyPath == "" {
		return "", ErrKeyPathUnset
	}
	if ttl <= 0 {
		return "", fmt.Errorf("auth: ttl must be positive, got %v", ttl)
	}
	if scope == "" {
		return "", errors.New("auth: scope must not be empty")
	}
	if strings.ContainsAny(scope, "|") {
		return "", errors.New("auth: scope must not contain '|'")
	}

	key, err := i.loadOrCreateKey()
	if err != nil {
		return "", err
	}
	issued := i.now().Unix()
	ttlSeconds := int64(ttl.Seconds())
	payload := fmt.Sprintf("%s|%d|%d", scope, issued, ttlSeconds)
	sig := sign(key, payload)
	return base64.RawURLEncoding.EncodeToString([]byte(payload + "|" + sig)), nil
}

// VerifyScopes checks the token's signature and expiration. On success it
// returns the comma-separated scope list bound to the token. The caller
// (typically pkg/policy.Enforce) decides whether those scopes satisfy the
// verb's required scope. Implements pkg/policy.TokenVerifier.
func (i *Issuer) VerifyScopes(token string) (string, error) {
	if i.KeyPath == "" {
		return "", ErrKeyPathUnset
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", fmt.Errorf("%w: not base64", ErrInvalidToken)
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 4 {
		return "", fmt.Errorf("%w: expected 4 parts, got %d", ErrInvalidToken, len(parts))
	}
	gotScope, issuedStr, ttlStr, gotSig := parts[0], parts[1], parts[2], parts[3]

	key, err := i.loadOrCreateKey()
	if err != nil {
		return "", err
	}
	payload := fmt.Sprintf("%s|%s|%s", gotScope, issuedStr, ttlStr)
	wantSig := sign(key, payload)
	if !hmac.Equal([]byte(gotSig), []byte(wantSig)) {
		return "", fmt.Errorf("%w: signature mismatch", ErrInvalidToken)
	}
	issued, err := strconv.ParseInt(issuedStr, 10, 64)
	if err != nil {
		return "", fmt.Errorf("%w: bad issued-at", ErrInvalidToken)
	}
	ttlSeconds, err := strconv.ParseInt(ttlStr, 10, 64)
	if err != nil {
		return "", fmt.Errorf("%w: bad ttl", ErrInvalidToken)
	}
	expires := time.Unix(issued, 0).Add(time.Duration(ttlSeconds) * time.Second)
	if i.now().After(expires) {
		return "", fmt.Errorf("%w: expired at %s", ErrInvalidToken, expires.Format(time.RFC3339))
	}
	return gotScope, nil
}

// Verify is a convenience for `coily auth verify --scope X --token T`. It
// runs VerifyScopes and then checks subsumption against scope using
// pkg/policy.Satisfies.
func (i *Issuer) Verify(scope, token string) error {
	held, err := i.VerifyScopes(token)
	if err != nil {
		return err
	}
	if !policy.Satisfies(held, scope) {
		return fmt.Errorf("%w: token scopes %q do not satisfy required %q",
			ErrInvalidToken, held, scope)
	}
	return nil
}

func (i *Issuer) loadOrCreateKey() ([]byte, error) {
	if b, err := os.ReadFile(i.KeyPath); err == nil {
		if len(b) < 32 {
			return nil, fmt.Errorf("auth: key at %s is too short (got %d bytes)", i.KeyPath, len(b))
		}
		return b, nil
	}
	// Create a new key. Parent dir must exist with tight perms.
	if err := os.MkdirAll(filepath.Dir(i.KeyPath), 0o700); err != nil {
		return nil, fmt.Errorf("auth: mkdir %s: %w", filepath.Dir(i.KeyPath), err)
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("auth: rand: %w", err)
	}
	if err := os.WriteFile(i.KeyPath, b, 0o600); err != nil {
		return nil, fmt.Errorf("auth: write key %s: %w", i.KeyPath, err)
	}
	return b, nil
}

func sign(key []byte, payload string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

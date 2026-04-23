// Package policy enforces the coily verb allowlist and validates that verb
// arguments do not contain shell metacharacters. Per docs/threat-model.md,
// policy is the programmatic expression of the safety boundary.
//
// Every coily verb Action should run through Enforce, which checks:
//
//  1. Every string flag / positional argument passes ValidateArg (no shell
//     metacharacters).
//  2. If the verb is mutating (bucket Write or Delete), a confirmation
//     token whose scope satisfies the verb's required scope must be
//     present.
//
// Enforce is the single choke point. Anything that bypasses it bypasses the
// security boundary, and that is a bug.
//
// Scope strings have the form `<binary>.<service>:<bucket>` where bucket is
// one of read|write|delete. A token's scope is a comma-separated list of
// scope strings. A required scope of `aws.route53:read` is satisfied by a
// token holding `aws.route53:read` or `aws.route53:write` (write subsumes
// read). A required scope of `aws.route53:delete` is satisfied only by a
// token holding `aws.route53:delete`. Delete is not subsumed by write.
package policy

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/coilysiren/coily/pkg/verbclass"
)

// ShellMeta is the set of bytes rejected in any string argument that could
// reach a subprocess. Exported so callers (and tests) can reason about it.
//
// Rationale: coily's subprocess execution path always uses an explicit argv
// slice, never a composed shell string. But some downstream tools (ssh
// <host> <remote-command>, kubectl exec, etc.) hand the last positional
// argument to a remote shell. Rejecting these characters at the coily
// boundary keeps a deny-list surprise at one layer from turning into an
// execution surprise at another.
const ShellMeta = "`$;&|<>(){}\\\n\r\t"

// ErrShellMeta is returned by ValidateArg when value contains a byte in
// ShellMeta.
var ErrShellMeta = errors.New("policy: shell metacharacter rejected")

// ErrTokenRequired is returned by Enforce when a mutating verb is invoked
// without a valid confirmation token.
var ErrTokenRequired = errors.New("policy: mutating verb requires a confirmation token")

// ErrInvalidScope is returned by ParseScope (and surfaces from auth issue)
// when a scope string does not match `<bin>.<svc>:(read|write|delete)`.
var ErrInvalidScope = errors.New("policy: invalid scope format")

// Kind marks whether a verb mutates remote state. ReadOnly verbs need no
// token. Mutating verbs do. The bucket (Read / Write / Delete) is carried
// separately on the Invocation as Bucket.
type Kind int

const (
	// ReadOnly verbs do not mutate remote state. No token required.
	ReadOnly Kind = iota
	// Mutating verbs change remote state. Require a confirmation token.
	Mutating
)

// FromBucket maps a verbclass.Bucket to a Kind. Read maps to ReadOnly,
// everything else to Mutating.
func FromBucket(b verbclass.Bucket) Kind {
	if b == verbclass.Read {
		return ReadOnly
	}
	return Mutating
}

// scopeRE matches `<bin>.<svc>:(read|write|delete)`. bin and svc are
// lowercase letters, digits, dashes, and underscores. svc is the immediate
// first sub-cli (route53, s3, api, get, etc.).
var scopeRE = regexp.MustCompile(`^[a-z0-9_]+(\.[a-z0-9_-]+)+:(read|write|delete)$`)

// ParseScope validates a scope string and returns its bucket. Used by
// `coily auth issue --scope` to reject malformed scopes at issuance time.
func ParseScope(scope string) (verbclass.Bucket, error) {
	if !scopeRE.MatchString(scope) {
		return verbclass.Read, fmt.Errorf("%w: %q (want <bin>.<svc>:(read|write|delete))",
			ErrInvalidScope, scope)
	}
	colon := strings.LastIndex(scope, ":")
	switch scope[colon+1:] {
	case "read":
		return verbclass.Read, nil
	case "write":
		return verbclass.Write, nil
	case "delete":
		return verbclass.Delete, nil
	}
	// Unreachable given the regex, but keeps the compiler happy.
	return verbclass.Read, fmt.Errorf("%w: %q", ErrInvalidScope, scope)
}

// ValidateScopeList validates a comma-separated list of scopes. Returns
// nil if every scope parses. Used by `coily auth issue --scope`.
func ValidateScopeList(list string) error {
	if list == "" {
		return fmt.Errorf("%w: scope must not be empty", ErrInvalidScope)
	}
	for _, s := range strings.Split(list, ",") {
		s = strings.TrimSpace(s)
		if _, err := ParseScope(s); err != nil {
			return err
		}
	}
	return nil
}

// Scope returns the scope string for a (binary, service, bucket) triple.
// Caller passes the binary name ("aws"), the immediate sub-cli ("route53"),
// and the bucket. Used by gen-passthrough at code generation time and by
// hand-written verbs at registration time.
func Scope(binary, service string, bucket verbclass.Bucket) string {
	return fmt.Sprintf("%s.%s:%s", binary, service, bucket)
}

// Satisfies reports whether tokenScopes (a comma-separated list of bound
// scopes from a verified token) satisfies required. Subsumption rules:
//
//   - A required `:read` scope is satisfied by a held `:read` or `:write`
//     scope on the same `<bin>.<svc>`.
//   - A required `:write` scope is satisfied only by a held `:write` scope
//     on the same `<bin>.<svc>`.
//   - A required `:delete` scope is satisfied only by a held `:delete`
//     scope on the same `<bin>.<svc>`. Delete is not subsumed by write.
//
// Ambiguous tokens or required scopes that fail to parse return false.
// (They were issued or computed by something the caller should not trust.)
func Satisfies(tokenScopes, required string) bool {
	requiredBucket, err := ParseScope(required)
	if err != nil {
		return false
	}
	requiredKey := scopeKey(required)
	for _, s := range strings.Split(tokenScopes, ",") {
		s = strings.TrimSpace(s)
		heldBucket, err := ParseScope(s)
		if err != nil {
			continue
		}
		if scopeKey(s) != requiredKey {
			continue
		}
		if heldBucket == requiredBucket {
			return true
		}
		// write subsumes read on the same key.
		if requiredBucket == verbclass.Read && heldBucket == verbclass.Write {
			return true
		}
	}
	return false
}

// scopeKey returns the `<bin>.<svc>` portion of a scope string, ignoring
// the bucket suffix.
func scopeKey(scope string) string {
	colon := strings.LastIndex(scope, ":")
	if colon < 0 {
		return scope
	}
	return scope[:colon]
}

// ValidateArg rejects strings containing shell metacharacters. Empty strings
// are allowed. Callers should check for empty separately if the argument is
// required.
func ValidateArg(name, value string) error {
	if i := strings.IndexAny(value, ShellMeta); i >= 0 {
		return fmt.Errorf("%w: arg %s contains %q at index %d",
			ErrShellMeta, name, value[i], i)
	}
	return nil
}

// ValidateArgs runs ValidateArg over a map, returning the first violation.
// Convenience for Action funcs that have already gathered flag values.
func ValidateArgs(args map[string]string) error {
	for name, value := range args {
		if err := ValidateArg(name, value); err != nil {
			return err
		}
	}
	return nil
}

// ValidateArgSlice runs ValidateArg over a []string (for variadic / positional
// arguments). Uses a synthetic name that includes the index.
func ValidateArgSlice(namePrefix string, values []string) error {
	for i, v := range values {
		if err := ValidateArg(fmt.Sprintf("%s[%d]", namePrefix, i), v); err != nil {
			return err
		}
	}
	return nil
}

// TokenVerifier abstracts the confirmation-token check so pkg/policy does not
// depend on pkg/auth directly. pkg/auth satisfies this interface.
//
// VerifyScopes returns the comma-separated scope list bound to token, or an
// error if the token is invalid (bad signature, malformed, expired). The
// caller (typically Enforce) decides whether the returned scopes satisfy
// the required scope.
type TokenVerifier interface {
	VerifyScopes(token string) (string, error)
}

// Invocation describes a single verb call for Enforce.
type Invocation struct {
	// Verb is the dotted verb path used as the audit-log key, e.g.
	// "aws.route53.change-resource-record-sets" or "lockdown".
	Verb string
	// Kind is ReadOnly or Mutating. Mutating verbs require a token.
	Kind Kind
	// Scope is the required scope string (`<bin>.<svc>:<bucket>`) the
	// token must satisfy. Only consulted when Kind is Mutating. May be
	// empty for ReadOnly verbs.
	Scope string
	// Args are every user-supplied string argument. Each is run through
	// ValidateArg.
	Args map[string]string
	// Positional is every positional argument (same treatment as Args).
	Positional []string
	// Token is the confirmation token presented by the caller (typically via
	// a --token flag or COILY_TOKEN env). Only consulted when Kind is
	// Mutating.
	Token string
}

// Enforce runs all checks for an invocation. Returns nil on success.
func Enforce(inv Invocation, verifier TokenVerifier) error {
	if err := ValidateArgs(inv.Args); err != nil {
		return err
	}
	if err := ValidateArgSlice("positional", inv.Positional); err != nil {
		return err
	}
	if inv.Kind != Mutating {
		return nil
	}
	if inv.Token == "" {
		return fmt.Errorf("%w: verb %q requires a token. Issue one with `coily auth issue --scope %s`",
			ErrTokenRequired, inv.Verb, inv.Scope)
	}
	if verifier == nil {
		return fmt.Errorf("%w: verb %q is mutating but no verifier is configured",
			ErrTokenRequired, inv.Verb)
	}
	heldScopes, err := verifier.VerifyScopes(inv.Token)
	if err != nil {
		return err
	}
	if inv.Scope == "" {
		// Defensive. A mutating invocation without a Scope is a wiring
		// bug somewhere upstream. Refuse rather than wave it through.
		return fmt.Errorf("%w: verb %q is mutating but has no required scope",
			ErrTokenRequired, inv.Verb)
	}
	if !Satisfies(heldScopes, inv.Scope) {
		return fmt.Errorf("%w: token scopes %q do not satisfy required %q",
			ErrTokenRequired, heldScopes, inv.Scope)
	}
	return nil
}

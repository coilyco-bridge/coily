package auth_test

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coilysiren/coily/pkg/auth"
)

func tempIssuer(t *testing.T) *auth.Issuer {
	t.Helper()
	dir := t.TempDir()
	return &auth.Issuer{
		KeyPath: filepath.Join(dir, "token-issuer.key"),
		Now:     func() time.Time { return time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC) },
	}
}

func TestRoundTrip(t *testing.T) {
	i := tempIssuer(t)
	tok, err := i.Issue("aws.route53:write", 5*time.Minute)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if err := i.Verify("aws.route53:write", tok); err != nil {
		t.Errorf("Verify: %v", err)
	}
}

func TestVerify_RejectsWrongScope(t *testing.T) {
	i := tempIssuer(t)
	tok, _ := i.Issue("aws.route53:write", 5*time.Minute)
	err := i.Verify("aws.s3:write", tok)
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Errorf("err = %v, want ErrInvalidToken", err)
	}
}

func TestVerify_WriteScopeGrantsRead(t *testing.T) {
	// Token issued with :write should satisfy a :read requirement on
	// the same service. Subsumption rule.
	i := tempIssuer(t)
	tok, _ := i.Issue("aws.route53:write", 5*time.Minute)
	if err := i.Verify("aws.route53:read", tok); err != nil {
		t.Errorf("write should subsume read: %v", err)
	}
}

func TestVerify_WriteScopeDoesNotGrantDelete(t *testing.T) {
	i := tempIssuer(t)
	tok, _ := i.Issue("aws.route53:write", 5*time.Minute)
	if err := i.Verify("aws.route53:delete", tok); err == nil {
		t.Error("write must not satisfy delete")
	}
}

func TestVerify_DeleteScopeDoesNotGrantWrite(t *testing.T) {
	i := tempIssuer(t)
	tok, _ := i.Issue("aws.route53:delete", 5*time.Minute)
	if err := i.Verify("aws.route53:write", tok); err == nil {
		t.Error("delete must not satisfy write")
	}
}

func TestVerify_CombinedScopesGrantBoth(t *testing.T) {
	i := tempIssuer(t)
	tok, _ := i.Issue("aws.route53:write,aws.route53:delete", 5*time.Minute)
	for _, req := range []string{"aws.route53:read", "aws.route53:write", "aws.route53:delete"} {
		if err := i.Verify(req, tok); err != nil {
			t.Errorf("combined token failed for %q: %v", req, err)
		}
	}
}

func TestVerify_RejectsExpired(t *testing.T) {
	i := tempIssuer(t)
	tok, _ := i.Issue("aws.route53:write", 1*time.Minute)
	// Advance clock past expiration.
	i.Now = func() time.Time { return time.Date(2026, 4, 22, 13, 0, 0, 0, time.UTC) }
	err := i.Verify("aws.route53:write", tok)
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Errorf("err = %v, want ErrInvalidToken", err)
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("err = %v, want to mention expiration", err)
	}
}

func TestVerify_RejectsSignatureTamper(t *testing.T) {
	i := tempIssuer(t)
	tok, _ := i.Issue("aws.route53:write", 5*time.Minute)
	// Flip a bit in the middle of the token.
	bad := []byte(tok)
	bad[len(bad)/2] ^= 0x01
	err := i.Verify("aws.route53:write", string(bad))
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Errorf("err = %v, want ErrInvalidToken", err)
	}
}

func TestVerify_RejectsMalformedToken(t *testing.T) {
	i := tempIssuer(t)
	cases := []string{
		"",
		"not-base64!!!",
		"YWJjZGVm", // base64 of "abcdef" (no pipes)
	}
	for _, c := range cases {
		err := i.Verify("aws.route53:write", c)
		if !errors.Is(err, auth.ErrInvalidToken) {
			t.Errorf("Verify(%q) err = %v, want ErrInvalidToken", c, err)
		}
	}
}

func TestVerify_DifferentKeyIsInvalid(t *testing.T) {
	i1 := tempIssuer(t)
	i2 := tempIssuer(t) // separate tempdir, separate key
	i2.Now = i1.Now
	tok, _ := i1.Issue("aws.route53:write", 5*time.Minute)
	err := i2.Verify("aws.route53:write", tok)
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Errorf("err = %v, want ErrInvalidToken", err)
	}
}

func TestIssue_RejectsEmptyScope(t *testing.T) {
	i := tempIssuer(t)
	_, err := i.Issue("", 1*time.Minute)
	if err == nil {
		t.Error("expected error for empty scope")
	}
}

func TestIssue_RejectsScopeWithPipe(t *testing.T) {
	i := tempIssuer(t)
	_, err := i.Issue("bad|scope", 1*time.Minute)
	if err == nil {
		t.Error("expected error for pipe in scope")
	}
}

func TestIssue_RejectsNonPositiveTTL(t *testing.T) {
	i := tempIssuer(t)
	for _, ttl := range []time.Duration{0, -1 * time.Second} {
		if _, err := i.Issue("aws.route53:write", ttl); err == nil {
			t.Errorf("expected error for ttl %v", ttl)
		}
	}
}

func TestIssue_UnsetKeyPathErrors(t *testing.T) {
	i := &auth.Issuer{}
	if _, err := i.Issue("aws.route53:write", time.Minute); !errors.Is(err, auth.ErrKeyPathUnset) {
		t.Errorf("err = %v, want ErrKeyPathUnset", err)
	}
}

func TestIssue_CreatesKeyWithTightPerms(t *testing.T) {
	i := tempIssuer(t)
	if _, err := i.Issue("aws.route53:write", time.Minute); err != nil {
		t.Fatalf("Issue: %v", err)
	}
	info, err := statKey(i.KeyPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Perm(); perm != 0o600 {
		t.Errorf("key perm = %o, want 0600", perm)
	}
}

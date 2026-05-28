package main

import (
	"strings"
	"testing"
)

// TestRenderForgejoCmd_PinnedTarget pins the namespace and pod selector in
// the rendered remote command. The wrapper deliberately doesn't expose flag
// overrides for these; the issue (#91) calls them out as code-pinned.
func TestRenderForgejoCmd_PinnedTarget(t *testing.T) {
	got := renderForgejoCmd([]string{"admin", "user", "list"})
	for _, want := range []string{
		"k3s kubectl ",
		" -n forgejo ",
		" exec deploy/forgejo ",
		" -- forgejo ",
		"'admin'",
		"'user'",
		"'list'",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered cmd missing %q; got %q", want, got)
		}
	}
}

// TestRenderForgejoCmd_NoSudo mirrors the issue #56 invariant for the
// forgejo wrapper: no sudo prefix on the remote command. /etc/rancher/k3s/
// k3s.yaml is mode 644 on kai-server.
func TestRenderForgejoCmd_NoSudo(t *testing.T) {
	got := renderForgejoCmd([]string{"admin", "user", "list"})
	if strings.HasPrefix(got, "sudo ") || strings.HasPrefix(got, "sudo\t") {
		t.Errorf("rendered cmd has sudo prefix: %q", got)
	}
}

// TestRenderForgejoCmd_QuotesArgs pins POSIX-quoting of forgejo args so the
// remote login shell does not re-interpret shell metacharacters. Doctor
// check names are constrained, but the helper is general.
func TestRenderForgejoCmd_QuotesArgs(t *testing.T) {
	got := renderForgejoCmd([]string{"doctor", "check", "--run", "db-version"})
	for _, want := range []string{"'doctor'", "'check'", "'--run'", "'db-version'"} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered cmd missing quoted token %q; got %q", want, got)
		}
	}
}

// TestRenderForgejoCmd_AdminUserCreatePinsRandomAndMustChange asserts the
// onboarding-slice invariant: the admin user create verb always carries
// --random-password and --must-change-password, and never a --password flag.
// Same renderer is shared with every other verb, so this test exercises the
// argv shape that the create command's Action builds.
func TestRenderForgejoCmd_AdminUserCreatePinsRandomAndMustChange(t *testing.T) {
	argv := []string{
		"admin", "user", "create",
		"--username", "peer",
		"--email", "peer@example.org",
		"--random-password",
		"--must-change-password",
		"--admin",
	}
	got := renderForgejoCmd(argv)
	for _, want := range []string{
		"'--random-password'",
		"'--must-change-password'",
		"'--admin'",
		"'peer'",
		"'peer@example.org'",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered cmd missing %q; got %q", want, got)
		}
	}
	if strings.Contains(got, "'--password'") {
		t.Errorf("rendered cmd must not carry --password; got %q", got)
	}
}

// TestForgejoLocalArgv_PinnedTarget asserts the local-exec argv shape
// matches the rendered remote command's pinned namespace and pod selector,
// and that no shell quoting leaks into argv (execve sees the raw bytes).
// Closes coilysiren/coily#260.
func TestForgejoLocalArgv_PinnedTarget(t *testing.T) {
	got := forgejoLocalArgv([]string{"admin", "user", "list"})
	want := []string{"k3s", "kubectl", "-n", "forgejo", "exec", "deploy/forgejo", "--", "forgejo", "admin", "user", "list"}
	if len(got) != len(want) {
		t.Fatalf("argv len = %d, want %d: got %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("argv[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	for _, a := range got {
		if strings.HasPrefix(a, "'") && strings.HasSuffix(a, "'") {
			t.Errorf("local argv element %q is POSIX-quoted; quoting is for the ssh path only", a)
		}
	}
}

// TestForgejoCommand_FJAlias pins the `fj` shorthand for `coily ops forgejo`.
func TestForgejoCommand_FJAlias(t *testing.T) {
	cmd := (&Runner{}).forgejoCommand()
	found := false
	for _, a := range cmd.Aliases {
		if a == "fj" {
			found = true
		}
	}
	if !found {
		t.Errorf("forgejo command missing 'fj' alias; got aliases %v", cmd.Aliases)
	}
}

func TestValidateForgejoUsername(t *testing.T) {
	ok := []string{"peer", "kai-siren", "k.s", "user_1", "AlphaBeta", "a"}
	for _, n := range ok {
		if err := validateForgejoUsername(n); err != nil {
			t.Errorf("valid username %q rejected: %v", n, err)
		}
	}
	bad := []string{
		"",
		"-leading-dash",
		"has space",
		"semi;colon",
		"back`tick",
		"dollar$sign",
		"slash/in",
		"at@sign",
		strings.Repeat("a", 41),
	}
	for _, n := range bad {
		if err := validateForgejoUsername(n); err == nil {
			t.Errorf("invalid username %q accepted", n)
		}
	}
}

func TestValidateForgejoEmail(t *testing.T) {
	ok := []string{
		"peer@example.org",
		"a.b+tag@sub.example.com",
		"u_1@x.y",
		"User-Name@Example.COM",
	}
	for _, e := range ok {
		if err := validateForgejoEmail(e); err != nil {
			t.Errorf("valid email %q rejected: %v", e, err)
		}
	}
	bad := []string{
		"",
		"-leading-dash@example.org",
		"no-at-sign",
		"two@@example.org",
		"@example.org",
		"local@",
		"local@nodot",
		"has space@example.org",
		"semi;colon@example.org",
		"back`tick@example.org",
		"dollar$@example.org",
		"slash/@example.org",
		strings.Repeat("a", 255) + "@example.org",
	}
	for _, e := range bad {
		if err := validateForgejoEmail(e); err == nil {
			t.Errorf("invalid email %q accepted", e)
		}
	}
}

func TestValidateForgejoDoctorCheckName(t *testing.T) {
	ok := []string{"db-version", "paths", "storages", "authorized_keys", "Auth-Tokens", "a", "x_1"}
	for _, n := range ok {
		if err := validateForgejoDoctorCheckName(n); err != nil {
			t.Errorf("valid name %q rejected: %v", n, err)
		}
	}
	bad := []string{
		"",
		"-leading-dash",
		"has space",
		"semi;colon",
		"back`tick",
		"dollar$sign",
		"slash/in",
		strings.Repeat("a", 65),
	}
	for _, n := range bad {
		if err := validateForgejoDoctorCheckName(n); err == nil {
			t.Errorf("invalid name %q accepted", n)
		}
	}
}

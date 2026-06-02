package main

import (
	"strings"
	"testing"
)

// TestForgejoLocalArgv_PinnedTarget pins the local-exec argv shape: the
// namespace and pod selector are code-pinned (issue #91), no sudo prefix
// (issue #56; /etc/rancher/k3s/k3s.yaml is mode 644 on kai-server), and no
// shell quoting leaks into argv (execve sees the raw bytes). Closes
// coilysiren/coily#260.
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
	if got[0] == "sudo" {
		t.Errorf("local argv must not be sudo-prefixed; got %v", got)
	}
	for _, a := range got {
		if strings.HasPrefix(a, "'") && strings.HasSuffix(a, "'") {
			t.Errorf("local argv element %q is POSIX-quoted; argv goes straight to execve", a)
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

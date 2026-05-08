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

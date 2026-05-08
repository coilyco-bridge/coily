package main

import (
	"strings"
	"testing"
)

// TestRenderSSHKubectlCmd_NoSudo pins issue #56: the remote command
// sent to kai-server for `coily ssh kubectl` does not prepend `sudo`.
// /etc/rancher/k3s/k3s.yaml is mode 644 on kai-server, so k3s kubectl
// reads it without root, and the previous sudo wrap broke every
// non-interactive call by demanding a password the wrapper couldn't
// supply.
func TestRenderSSHKubectlCmd_NoSudo(t *testing.T) {
	cases := [][]string{
		{"get", "nodes"},
		{"get", "pods", "-A"},
		{"-n", "kube-system", "get", "pods", "-l", "app=coredns"},
		{"get", "pods", "-o", "jsonpath={.items[*].metadata.name}"},
	}
	for _, argv := range cases {
		got := renderSSHKubectlCmd(argv)
		// Use word-boundary matching: a literal substring "sudo" inside
		// a quoted arg ("'sudo-test'") is fine; a leading-token "sudo "
		// is the bug shape this test guards against.
		if strings.HasPrefix(got, "sudo ") || strings.HasPrefix(got, "sudo\t") {
			t.Errorf("argv %v rendered with sudo prefix: %q", argv, got)
		}
		if !strings.HasPrefix(got, "k3s kubectl ") {
			t.Errorf("argv %v rendered without k3s kubectl prefix: %q", argv, got)
		}
	}
}

// TestRenderSSHKubectlCmd_QuotesShellMetachars pins the existing argv-
// quoting behavior: kubectl args carrying jsonpath braces, label
// selector commas, or whitespace must reach the remote shell as a
// single token, not be re-interpreted by bash on kai-server.
func TestRenderSSHKubectlCmd_QuotesShellMetachars(t *testing.T) {
	got := renderSSHKubectlCmd([]string{"get", "pods", "-l", "a=b,c=d", "-o", "jsonpath={.items[0].metadata.name}"})
	for _, want := range []string{"'a=b,c=d'", "'jsonpath={.items[0].metadata.name}'"} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered cmd missing quoted token %q; got %q", want, got)
		}
	}
}

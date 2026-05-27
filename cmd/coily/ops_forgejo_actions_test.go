package main

import (
	"strings"
	"testing"
)

// TestRenderForgejoActionsLogsCmd_Shape pins the remote shell command that
// `ops forgejo actions task logs` builds: the kubectl exec target plus the
// zstd decode tail. The glob over hex shards is intentional - one task id
// matches exactly one file, and globbing avoids hardcoding forgejo's
// internal sharding scheme.
func TestRenderForgejoActionsLogsCmd_Shape(t *testing.T) {
	got := renderForgejoActionsLogsCmd("coilysiren", "coily", 97)
	for _, want := range []string{
		"k3s kubectl ",
		" -n forgejo ",
		" exec deploy/forgejo ",
		" -- sh -c ",
		"/var/lib/gitea/actions_log/coilysiren/coily/*/97.log.zst",
		"| zstd -d",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered cmd missing %q; got %q", want, got)
		}
	}
}

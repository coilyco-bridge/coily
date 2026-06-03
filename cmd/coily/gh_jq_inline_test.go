package main

import (
	"os"
	"strings"
	"testing"
)

func TestOpsGHArgStart(t *testing.T) {
	cases := []struct {
		in   []string
		want int
	}{
		{[]string{"coily", "ops", "gh", "api", "x"}, 3},
		{[]string{"coily", "--cwd", "/x", "ops", "gh", "api"}, 5},
		{[]string{"coily", "ops", "aws", "s3", "ls"}, -1},
		{[]string{"coily", "git", "status"}, -1},
		{[]string{"coily", "ops"}, -1},
	}
	for _, c := range cases {
		if got := opsGHArgStart(c.in); got != c.want {
			t.Errorf("opsGHArgStart(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}

// TestNormalizeGHJQInlineRoundTrip is the core guarantee of coily#30: an
// inline --jq with gate-tripping metachars is spilled to a --jq-file path
// (which the gate accepts), and the existing rewriteJQFile pass expands it
// back to the identical `--jq <expr>` before gh runs.
func TestNormalizeGHJQInlineRoundTrip(t *testing.T) {
	const expr = ".workflow_runs[] | select(.status != \"completed\") | .id"
	forms := [][]string{
		{"coily", "ops", "gh", "api", "x", "--jq", expr},
		{"coily", "ops", "gh", "api", "x", "--jq=" + expr},
		{"coily", "ops", "gh", "api", "x", "-q", expr},
		{"coily", "ops", "gh", "api", "x", "-q=" + expr},
	}
	for _, in := range forms {
		out, cleanup := normalizeGHJQInline(in)

		// The validated argv must carry --jq-file + a clean path, not the
		// metachar-bearing expression.
		joined := strings.Join(out, " ")
		if strings.Contains(joined, "|") {
			t.Errorf("%v: normalized argv still contains a pipe: %q", in, joined)
		}
		if !containsString(out, "--jq-file") {
			t.Errorf("%v: normalized argv missing --jq-file: %v", in, out)
		}

		// rewriteJQFile (the post-gate rail) must expand it back to the
		// original expression.
		expanded := rewriteJQFile(out)
		if got := flagValue(expanded, "--jq"); got != expr {
			t.Errorf("%v: round-trip --jq = %q, want %q", in, got, expr)
		}
		cleanup()
	}
}

func TestNormalizeGHJQInlineLeavesOthersAlone(t *testing.T) {
	// Non-ops-gh invocation: untouched.
	in := []string{"coily", "ops", "aws", "route53", "list", "--jq", ".a|.b"}
	out, cleanup := normalizeGHJQInline(in)
	defer cleanup()
	if strings.Join(out, " ") != strings.Join(in, " ") {
		t.Errorf("non-gh argv mutated: %v -> %v", in, out)
	}

	// Dangling --jq with no value: left as-is for gh to report.
	in2 := []string{"coily", "ops", "gh", "api", "x", "--jq"}
	out2, cleanup2 := normalizeGHJQInline(in2)
	defer cleanup2()
	if strings.Join(out2, " ") != strings.Join(in2, " ") {
		t.Errorf("dangling --jq mutated: %v -> %v", in2, out2)
	}

	// An existing --jq-file is not double-handled.
	in3 := []string{"coily", "ops", "gh", "api", "x", "--jq-file", "/tmp/q.jq"}
	out3, cleanup3 := normalizeGHJQInline(in3)
	defer cleanup3()
	if strings.Join(out3, " ") != strings.Join(in3, " ") {
		t.Errorf("--jq-file mutated: %v -> %v", in3, out3)
	}
}

func TestNormalizeGHJQInlineCleanupRemovesTemp(t *testing.T) {
	in := []string{"coily", "ops", "gh", "api", "x", "--jq", ".a|.b"}
	out, cleanup := normalizeGHJQInline(in)
	path := flagValue(out, "--jq-file")
	if path == "" {
		t.Fatalf("expected a --jq-file path, got argv %v", out)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("temp file should exist before cleanup: %v", err)
	}
	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("temp file should be gone after cleanup, stat err = %v", err)
	}
}

// flagValue returns the token following the first occurrence of flag, or "".
func flagValue(argv []string, flag string) string {
	for i := 0; i+1 < len(argv); i++ {
		if argv[i] == flag {
			return argv[i+1]
		}
	}
	return ""
}

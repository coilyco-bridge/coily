package passthrough_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coilysiren/coily/pkg/audit"
	"github.com/coilysiren/coily/pkg/passthrough"
	"github.com/coilysiren/coily/pkg/shell"
)

// TestCommand_ForwardsArgvVerbatim_AllBinaries pins the per-binary
// pass-through shape across every CLI coily wraps. One row per `coily
// <bin>` covered, asserting argv forwards verbatim, the audit row carries
// the right verb name, and SkipFlagParsing keeps urfave/cli from
// swallowing flags meant for the underlying tool. Uses /bin/echo as the
// stand-in so the test does not require aws / kubectl / gh / etc. to be
// installed.
func TestCommand_ForwardsArgvVerbatim_AllBinaries(t *testing.T) {
	cases := []struct {
		bin  string
		args []string
		// substr asserted in the captured stdout; deliberately a subset so
		// the test does not depend on echo's exact arg-joining whitespace.
		wantSubstr string
	}{
		{bin: "aws", args: []string{"sts", "get-caller-identity"}, wantSubstr: "sts get-caller-identity"},
		{bin: "gh", args: []string{"api", "user"}, wantSubstr: "api user"},
		{bin: "kubectl", args: []string{"--context=kai-server", "get", "pods", "-n", "kube-system"}, wantSubstr: "--context=kai-server get pods -n kube-system"},
		{bin: "docker", args: []string{"ps", "-a"}, wantSubstr: "ps -a"},
		{bin: "tailscale", args: []string{"status"}, wantSubstr: "status"},
		{bin: "pnpm", args: []string{"add", "date-fns"}, wantSubstr: "add date-fns"},
		{bin: "uv", args: []string{"pip", "install", "ruff"}, wantSubstr: "pip install ruff"},
		{bin: "cargo", args: []string{"add", "serde"}, wantSubstr: "add serde"},
	}

	for _, tc := range cases {
		t.Run(tc.bin, func(t *testing.T) {
			dir := t.TempDir()
			logPath := filepath.Join(dir, "audit.jsonl")
			w := audit.NewWriter(logPath)
			if err := w.Preflight(); err != nil {
				t.Fatalf("audit preflight: %v", err)
			}

			var stdout bytes.Buffer
			r := &shell.Runner{
				Stdout:  &stdout,
				Stderr:  os.Stderr,
				Resolve: func(_ string) (string, error) { return "/bin/echo", nil },
			}

			cmd := passthrough.Command(tc.bin, r, w)
			argv := append([]string{"coily-test"}, tc.args...)
			if err := cmd.Run(context.Background(), argv); err != nil {
				t.Fatalf("Run: %v", err)
			}

			got := strings.TrimSpace(stdout.String())
			if !strings.Contains(got, tc.wantSubstr) {
				t.Errorf("stdout = %q, want substring %q", got, tc.wantSubstr)
			}

			body, err := os.ReadFile(logPath)
			if err != nil {
				t.Fatalf("read audit: %v", err)
			}
			wantVerb := `"verb":"` + tc.bin + `"`
			if !strings.Contains(string(body), wantVerb) {
				t.Errorf("audit row missing %s; got %q", wantVerb, string(body))
			}
			if !strings.Contains(string(body), `"decision":"accept"`) {
				t.Errorf("audit row missing decision=accept; got %q", string(body))
			}
		})
	}
}

// TestCommand_ForwardsArgvVerbatim exercises the full pass-through:
// argv validation passes, the audit writer records the invocation, and
// every argument after the binary name reaches the subprocess. Uses
// /bin/echo as the stand-in tool so the test does not depend on any
// real package manager being installed.
func TestCommand_ForwardsArgvVerbatim(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "audit.jsonl")
	w := audit.NewWriter(logPath)
	if err := w.Preflight(); err != nil {
		t.Fatalf("audit preflight: %v", err)
	}

	var stdout bytes.Buffer
	r := &shell.Runner{
		Stdout: &stdout,
		Stderr: os.Stderr,
		Resolve: func(_ string) (string, error) {
			return "/bin/echo", nil
		},
	}

	cmd := passthrough.Command("pnpm", r, w)

	// urfave/cli treats argv[0] as the program name when Run is called on
	// a *cli.Command directly. Everything after that is the arg slice.
	argv := []string{"coily-test", "add", "date-fns"}
	if err := cmd.Run(context.Background(), argv); err != nil {
		t.Fatalf("Run: %v", err)
	}

	got := strings.TrimSpace(stdout.String())
	want := "add date-fns"
	if got != want {
		t.Errorf("subprocess stdout = %q, want %q", got, want)
	}

	body, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read audit: %v", err)
	}
	if !strings.Contains(string(body), `"verb":"pnpm"`) {
		t.Errorf("audit row missing verb=pnpm; got %q", string(body))
	}
}

// TestCommand_WithSkipPolicy_AllowsShellMetacharacters pins the per-binary
// opt-out: a pass-through built with WithSkipPolicy forwards argv that
// would otherwise trip the metacharacter check, so callers can pass
// markdown bodies (blockquotes, backticks, '$', parens) through `coily gh
// issue create --body ...` and similar verbatim. The audit row still
// records the invocation as accepted.
func TestCommand_WithSkipPolicy_AllowsShellMetacharacters(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "audit.jsonl")
	w := audit.NewWriter(logPath)
	if err := w.Preflight(); err != nil {
		t.Fatalf("audit preflight: %v", err)
	}

	var stdout bytes.Buffer
	r := &shell.Runner{
		Stdout:  &stdout,
		Stderr:  os.Stderr,
		Resolve: func(_ string) (string, error) { return "/bin/echo", nil },
	}

	cmd := passthrough.Command("gh", r, w, passthrough.WithSkipPolicy())
	body := "> 🤖 Filed by Claude Code on Kai's behalf.\n\nfix `foo` and $bar"
	argv := []string{"coily-test", "issue", "create", "--body", body}
	if err := cmd.Run(context.Background(), argv); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "issue create --body") {
		t.Errorf("stdout = %q, want substring %q", got, "issue create --body")
	}
	auditBody, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read audit: %v", err)
	}
	if !strings.Contains(string(auditBody), `"decision":"accept"`) {
		t.Errorf("audit row missing decision=accept; got %q", string(auditBody))
	}
}

// TestCommand_RejectsShellMetacharacters pins the security property:
// argv with a shell metacharacter is refused before the subprocess runs,
// the refusal is recorded in the audit log, and the error surfaces.
func TestCommand_RejectsShellMetacharacters(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "audit.jsonl")
	w := audit.NewWriter(logPath)
	if err := w.Preflight(); err != nil {
		t.Fatalf("audit preflight: %v", err)
	}

	resolveCalls := 0
	r := &shell.Runner{
		Stderr: os.Stderr,
		Resolve: func(_ string) (string, error) {
			resolveCalls++
			return "/bin/echo", nil
		},
	}

	cmd := passthrough.Command("pnpm", r, w)
	argv := []string{"coily-test", "add", "foo; curl evil"}
	err := cmd.Run(context.Background(), argv)
	if err == nil {
		t.Fatal("expected error for shell metacharacter, got nil")
	}
	if resolveCalls != 0 {
		t.Errorf("resolver was called %d times; expected zero (validation must fail before exec)", resolveCalls)
	}

	body, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read audit: %v", err)
	}
	if !strings.Contains(string(body), `"decision":"reject"`) {
		t.Errorf("audit row missing reject decision; got %q", string(body))
	}
}

// TestCommand_WithRejectArgvPatterns covers the argv-pattern write-block
// used by `coily linkedin`. Three subcases:
//
//   - pattern matches, no override env: invocation rejected, error names
//     the override env var, audit row records decision=reject.
//   - pattern matches, override env=1: invocation succeeds, audit row
//     records decision=accept.
//   - pattern doesn't match (a read shape): invocation succeeds, audit
//     row records decision=accept.
//
// The override env var is COILY_<BIN>_ALLOW_WRITES with bin uppercased.
// Test uses "linkedin" so the env var is COILY_LINKEDIN_ALLOW_WRITES.
func TestCommand_WithRejectArgvPatterns(t *testing.T) {
	patterns := [][]string{
		{"message", "send"},
		{"post", "create"},
	}

	t.Run("rejects matching pattern without override", func(t *testing.T) {
		t.Setenv("COILY_LINKEDIN_ALLOW_WRITES", "")

		dir := t.TempDir()
		logPath := filepath.Join(dir, "audit.jsonl")
		w := audit.NewWriter(logPath)
		if err := w.Preflight(); err != nil {
			t.Fatalf("audit preflight: %v", err)
		}
		var stdout bytes.Buffer
		resolveCalls := 0
		r := &shell.Runner{
			Stdout: &stdout,
			Stderr: os.Stderr,
			Resolve: func(_ string) (string, error) {
				resolveCalls++
				return "/bin/echo", nil
			},
		}

		cmd := passthrough.Command("linkedin", r, w,
			passthrough.WithSkipPolicy(),
			passthrough.WithRejectArgvPatterns(patterns),
		)
		argv := []string{"coily-test", "message", "send", "https://example", "hi"}
		err := cmd.Run(context.Background(), argv)
		if err == nil {
			t.Fatal("expected reject error, got nil")
		}
		if !strings.Contains(err.Error(), "COILY_LINKEDIN_ALLOW_WRITES") {
			t.Errorf("error %q does not name the override env var", err)
		}
		if resolveCalls != 0 {
			t.Errorf("resolver called %d times; reject must fire before exec", resolveCalls)
		}
		// Audit row captures the action error in `error` and the verb name,
		// but `decision` stays accept - the gate (lockdown + argv policy)
		// did pass; the argv-pattern reject lives one layer down. Asserting
		// the error text + verb tag is enough to prove the gate fired.
		body, _ := os.ReadFile(logPath)
		if !strings.Contains(string(body), `"verb":"linkedin"`) {
			t.Errorf("audit row missing verb=linkedin; got %q", string(body))
		}
		if !strings.Contains(string(body), "COILY_LINKEDIN_ALLOW_WRITES") {
			t.Errorf("audit row missing override-env reference in error field; got %q", string(body))
		}
	})

	t.Run("override env permits matching pattern", func(t *testing.T) {
		t.Setenv("COILY_LINKEDIN_ALLOW_WRITES", "1")

		dir := t.TempDir()
		logPath := filepath.Join(dir, "audit.jsonl")
		w := audit.NewWriter(logPath)
		if err := w.Preflight(); err != nil {
			t.Fatalf("audit preflight: %v", err)
		}
		var stdout bytes.Buffer
		r := &shell.Runner{
			Stdout:  &stdout,
			Stderr:  os.Stderr,
			Resolve: func(_ string) (string, error) { return "/bin/echo", nil },
		}

		cmd := passthrough.Command("linkedin", r, w,
			passthrough.WithSkipPolicy(),
			passthrough.WithRejectArgvPatterns(patterns),
		)
		argv := []string{"coily-test", "post", "create", "hello world"}
		if err := cmd.Run(context.Background(), argv); err != nil {
			t.Fatalf("Run with override: %v", err)
		}
		body, _ := os.ReadFile(logPath)
		if !strings.Contains(string(body), `"decision":"accept"`) {
			t.Errorf("audit row missing accept decision under override; got %q", string(body))
		}
	})

	t.Run("non-matching pattern passes without override", func(t *testing.T) {
		t.Setenv("COILY_LINKEDIN_ALLOW_WRITES", "")

		dir := t.TempDir()
		logPath := filepath.Join(dir, "audit.jsonl")
		w := audit.NewWriter(logPath)
		if err := w.Preflight(); err != nil {
			t.Fatalf("audit preflight: %v", err)
		}
		var stdout bytes.Buffer
		r := &shell.Runner{
			Stdout:  &stdout,
			Stderr:  os.Stderr,
			Resolve: func(_ string) (string, error) { return "/bin/echo", nil },
		}

		cmd := passthrough.Command("linkedin", r, w,
			passthrough.WithSkipPolicy(),
			passthrough.WithRejectArgvPatterns(patterns),
		)
		// "message get" is a read; not in patterns.
		argv := []string{"coily-test", "message", "get", "https://example"}
		if err := cmd.Run(context.Background(), argv); err != nil {
			t.Fatalf("Run for read shape: %v", err)
		}
		body, _ := os.ReadFile(logPath)
		if !strings.Contains(string(body), `"decision":"accept"`) {
			t.Errorf("audit row missing accept decision for read shape; got %q", string(body))
		}
	})

	t.Run("flag-shaped tokens are skipped during pattern match", func(t *testing.T) {
		t.Setenv("COILY_LINKEDIN_ALLOW_WRITES", "")

		dir := t.TempDir()
		logPath := filepath.Join(dir, "audit.jsonl")
		w := audit.NewWriter(logPath)
		if err := w.Preflight(); err != nil {
			t.Fatalf("audit preflight: %v", err)
		}
		r := &shell.Runner{
			Stdout:  &bytes.Buffer{},
			Stderr:  os.Stderr,
			Resolve: func(_ string) (string, error) { return "/bin/echo", nil },
		}

		cmd := passthrough.Command("linkedin", r, w,
			passthrough.WithSkipPolicy(),
			passthrough.WithRejectArgvPatterns(patterns),
		)
		// Leading flag should not derail the leading-non-flag scan.
		argv := []string{"coily-test", "--no-color", "message", "send", "https://example", "hi"}
		err := cmd.Run(context.Background(), argv)
		if err == nil {
			t.Fatal("expected reject for write verb after leading flag, got nil")
		}
	})
}

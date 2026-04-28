package pkgmgr_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coilysiren/coily/pkg/audit"
	"github.com/coilysiren/coily/pkg/ops/pkgmgr"
	"github.com/coilysiren/coily/pkg/shell"
)

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

	cmd := pkgmgr.Command("pnpm", r, w)

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

	cmd := pkgmgr.Command("pnpm", r, w)
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

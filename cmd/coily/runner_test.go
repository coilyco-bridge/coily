package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/coilysiren/cli-guard/audit"
	"github.com/coilysiren/cli-guard/config"
	"github.com/coilysiren/cli-guard/shell"
	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// newTestRunner builds a Runner with no real disk-backed dependencies.
// Audit goes to a tempdir so the writer can flush successfully without
// touching ~/.local/state.
func newTestRunner(t *testing.T) *Runner {
	t.Helper()
	dir := t.TempDir()
	aw := audit.NewWriter(filepath.Join(dir, "audit.jsonl"))
	// Close on cleanup so Windows can actually remove the TempDir.
	t.Cleanup(func() { _ = aw.Close() })
	return &Runner{
		Cfg:    &config.Config{},
		Runner: &shell.Runner{Stdout: os.Stdout, Stderr: os.Stderr, Stdin: os.Stdin},
		Audit:  aw,
	}
}

// TestRunner_VersionCommand exercises a no-op verb through a wired Runner
// end-to-end via the cli.Command tree.
func TestRunner_VersionCommand(t *testing.T) {
	r := newTestRunner(t)
	cmd := r.versionCommand()

	if err := cmd.Run(context.Background(), []string{"version"}); err != nil {
		t.Fatalf("version: %v", err)
	}
}

// TestRunner_WrappedActionAuditsAndRuns proves that a wrapped verb runs its
// action on success and records the attempt in the audit log. It also
// exercises the failure path so the audit record captures exit_code=1.
func TestRunner_WrappedActionAuditsAndRuns(t *testing.T) {
	r := newTestRunner(t)

	called := false
	wrapped := r.WrapVerb(
		verb.Spec{
			Name: "test.action",
			ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
				return map[string]string{"--thing": c.String("thing")}, nil
			},
			Action: func(_ context.Context, _ *cli.Command) error {
				called = true
				return nil
			},
		},
		r.Audit,
	)
	cmd := &cli.Command{
		Name: "test",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "thing"},
		},
		Action: wrapped,
	}

	if err := cmd.Run(context.Background(), []string{"test", "--thing", "safe"}); err != nil {
		t.Fatalf("happy: %v", err)
	}
	if !called {
		t.Fatalf("action was not called on happy path")
	}

	// Sad path: argv validation fails before action runs.
	called = false
	err := cmd.Run(context.Background(), []string{"test", "--thing", "foo;bar"})
	if err == nil {
		t.Fatalf("expected shell-meta rejection, got nil")
	}
	if called {
		t.Fatalf("action ran on sad path; argv validation is broken")
	}

	// Audit log should exist and be non-empty after the happy path.
	b, readErr := os.ReadFile(r.Audit.Path)
	if readErr != nil {
		t.Fatalf("read audit log: %v", readErr)
	}
	if !bytes.Contains(b, []byte("test.action")) {
		t.Fatalf("audit log missing test.action, got: %s", b)
	}
}

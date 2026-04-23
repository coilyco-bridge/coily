package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/coilysiren/coily/pkg/audit"
	"github.com/coilysiren/coily/pkg/config"
	"github.com/coilysiren/coily/pkg/policy"
	"github.com/coilysiren/coily/pkg/shell"
	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

// fakeVerifier is a tiny stand-in for *auth.Issuer. Records every call and
// returns whatever Err is set to. Lets the test assert that a Runner's
// Verifier is actually consulted on the path it claims to be.
type fakeVerifier struct {
	mu    sync.Mutex
	calls []verifyCall
	Err   error
}

type verifyCall struct{ Token string }

// VerifyScopes implements policy.TokenVerifier. Records every call and
// returns whatever Err is set to. When Err is nil, returns the constant
// scope "test.svc:write" so subsumption checks against any required scope of
// "test.svc:write" succeed.
func (f *fakeVerifier) VerifyScopes(token string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, verifyCall{token})
	if f.Err != nil {
		return "", f.Err
	}
	return "test.svc:write", nil
}

// newTestRunner builds a Runner with no real disk-backed dependencies.
// Audit goes to a tempdir so the writer can flush successfully without
// touching ~/.local/state. Verifier is the supplied fake.
func newTestRunner(t *testing.T, v *fakeVerifier) *Runner {
	t.Helper()
	dir := t.TempDir()
	aw := audit.NewWriter(filepath.Join(dir, "audit.jsonl"))
	return &Runner{
		Cfg:      &config.Config{},
		Runner:   &shell.Runner{Stdout: os.Stdout, Stderr: os.Stderr, Stdin: os.Stdin},
		Audit:    aw,
		Verifier: v,
	}
}

// TestRunner_VersionCommand exercises a no-op verb through a fake-wired
// Runner end-to-end via the cli.Command tree. Proves the design lets
// cmd/coily/ have actual tests, which was the unresolved bug.
func TestRunner_VersionCommand(t *testing.T) {
	r := newTestRunner(t, &fakeVerifier{})
	cmd := r.versionCommand()

	// version's Action prints to stdout. We only care that it runs without
	// error from the urfave/cli root, not the printed text.
	if err := cmd.Run(context.Background(), []string{"version"}); err != nil {
		t.Fatalf("version: %v", err)
	}
}

// TestRunner_FakeVerifierWired proves the verifier dependency is the one
// consulted on a Mutating verb path through verb.Wrap -> policy.Enforce.
// Uses a fake that records calls and can return errors. Audit-write
// is also exercised so we know the full pipeline runs.
func TestRunner_FakeVerifierWired(t *testing.T) {
	v := &fakeVerifier{}
	r := newTestRunner(t, v)

	// Construct a Mutating verb on the fly using verb.Wrap directly.
	// Required scope "test.svc:write" matches what the fake reports, so
	// subsumption succeeds and the action runs.
	called := false
	wrapped := verb.Wrap(
		verb.Spec{
			Name:  "test.mutator",
			Kind:  policy.Mutating,
			Scope: "test.svc:write",
			ArgsFunc: func(c *cli.Command) (map[string]string, []string, string) {
				return nil, nil, c.String("token")
			},
			Action: func(_ context.Context, _ *cli.Command) error {
				called = true
				return nil
			},
		},
		r.Verifier,
		r.Audit,
	)
	cmd := &cli.Command{
		Name: "test",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "token"},
		},
		Action: wrapped,
	}

	// Happy path.
	if err := cmd.Run(context.Background(), []string{"test", "--token", "tok-abc"}); err != nil {
		t.Fatalf("happy: %v", err)
	}
	if !called {
		t.Fatalf("action was not called on happy path")
	}
	if len(v.calls) != 1 || v.calls[0].Token != "tok-abc" {
		t.Fatalf("verifier calls = %#v, want one {tok-abc}", v.calls)
	}

	// Sad path: verifier returns an error, action must not run.
	v.calls = nil
	v.Err = errors.New("nope")
	called = false
	err := cmd.Run(context.Background(), []string{"test", "--token", "bad"})
	if err == nil {
		t.Fatalf("sad path: expected error, got nil")
	}
	if called {
		t.Fatalf("action ran on sad path; policy gate is broken")
	}

	// Audit log file should exist and be non-empty after both invocations.
	st, statErr := os.Stat(r.Audit.Path)
	if statErr != nil {
		t.Fatalf("stat audit log: %v", statErr)
	}
	if st.Size() == 0 {
		t.Fatalf("audit log is empty; verb.Wrap should have written records")
	}
	b, readErr := os.ReadFile(r.Audit.Path)
	if readErr != nil {
		t.Fatalf("read audit log: %v", readErr)
	}
	if !bytes.Contains(b, []byte("test.mutator")) {
		t.Fatalf("audit log missing test.mutator, got: %s", b)
	}
}

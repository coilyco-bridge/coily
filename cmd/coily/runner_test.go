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
	"github.com/coilysiren/coily/pkg/shell"
)

// fakeVerifier is a tiny stand-in for *auth.Issuer. Records every call and
// returns whatever Err is set to. Lets the test assert that a Runner's
// Verifier is actually consulted on the path it claims to be.
type fakeVerifier struct {
	mu    sync.Mutex
	calls []verifyCall
	Err   error
}

type verifyCall struct{ Scope, Token string }

func (f *fakeVerifier) Verify(scope, token string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, verifyCall{scope, token})
	return f.Err
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

// TestRunner_AuthVerify proves the verifier dependency is the one consulted
// for a scope/token check, that audit logging happens, and that a verifier
// error propagates back to the caller. Three properties from one test.
func TestRunner_AuthVerify(t *testing.T) {
	v := &fakeVerifier{}
	r := newTestRunner(t, v)

	// Happy path: fake returns nil, command should succeed and the verifier
	// should have seen our scope/token.
	cmd := r.authVerifyCommand()
	if err := cmd.Run(context.Background(), []string{"verify", "--scope", "test.scope", "--token", "tok-abc"}); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if len(v.calls) != 1 || v.calls[0].Scope != "test.scope" || v.calls[0].Token != "tok-abc" {
		t.Fatalf("verifier calls = %#v, want one {test.scope tok-abc}", v.calls)
	}

	// Sad path: fake returns an error, command should fail with the same
	// error. Reset call log first so we can re-assert cleanly.
	v.calls = nil
	v.Err = errors.New("nope")
	cmd2 := r.authVerifyCommand()
	err := cmd2.Run(context.Background(), []string{"verify", "--scope", "test.scope", "--token", "bad"})
	if err == nil || err.Error() != "nope" {
		t.Fatalf("verify err = %v, want \"nope\"", err)
	}
	if len(v.calls) != 1 {
		t.Fatalf("verifier calls = %#v, want one", v.calls)
	}

	// Audit log file should exist and be non-empty after both invocations.
	st, statErr := os.Stat(r.Audit.Path)
	if statErr != nil {
		t.Fatalf("stat audit log: %v", statErr)
	}
	if st.Size() == 0 {
		t.Fatalf("audit log is empty; verb.Wrap should have written records")
	}

	// And the records should mention auth.verify.
	b, readErr := os.ReadFile(r.Audit.Path)
	if readErr != nil {
		t.Fatalf("read audit log: %v", readErr)
	}
	if !bytes.Contains(b, []byte("auth.verify")) {
		t.Fatalf("audit log missing auth.verify, got: %s", b)
	}
}

package verb_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/coilysiren/coily/pkg/audit"
	"github.com/coilysiren/coily/pkg/policy"
	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

type fakeVerifier struct{ ok bool }

func (f fakeVerifier) Verify(_, _ string) error {
	if f.ok {
		return nil
	}
	return errors.New("bad token")
}

func newTestWriter(t *testing.T) *audit.Writer {
	t.Helper()
	return &audit.Writer{
		Path: filepath.Join(t.TempDir(), "audit.jsonl"),
		Now:  func() time.Time { return time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC) },
	}
}

func TestWrap_ReadOnlyRunsAction(t *testing.T) {
	called := false
	spec := verb.Spec{
		Name: "test.ro",
		Kind: policy.ReadOnly,
		Action: func(_ context.Context, _ *cli.Command) error {
			called = true
			return nil
		},
	}
	err := runWrapped(t, spec, nil, newTestWriter(t))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !called {
		t.Error("action was not called")
	}
}

func TestWrap_MutatingWithoutTokenFails(t *testing.T) {
	spec := verb.Spec{
		Name:   "test.mut",
		Kind:   policy.Mutating,
		Action: func(_ context.Context, _ *cli.Command) error { return nil },
	}
	err := runWrapped(t, spec, fakeVerifier{ok: true}, newTestWriter(t))
	if !errors.Is(err, policy.ErrTokenRequired) {
		t.Errorf("err = %v, want ErrTokenRequired", err)
	}
}

func TestWrap_RejectsShellMetacharInArg(t *testing.T) {
	spec := verb.Spec{
		Name: "test.ro",
		Kind: policy.ReadOnly,
		ArgsFunc: func(_ *cli.Command) (map[string]string, []string, string) {
			return map[string]string{"--thing": "foo;bar"}, nil, ""
		},
		Action: func(_ context.Context, _ *cli.Command) error {
			t.Error("action should not be called when args fail policy")
			return nil
		},
	}
	err := runWrapped(t, spec, nil, newTestWriter(t))
	if !errors.Is(err, policy.ErrShellMeta) {
		t.Errorf("err = %v, want ErrShellMeta", err)
	}
}

func TestWrap_WritesAuditRecord(t *testing.T) {
	w := newTestWriter(t)
	spec := verb.Spec{
		Name:   "test.ro",
		Kind:   policy.ReadOnly,
		Action: func(_ context.Context, _ *cli.Command) error { return nil },
	}
	if err := runWrapped(t, spec, nil, w); err != nil {
		t.Fatalf("run: %v", err)
	}
	b, err := os.ReadFile(w.Path)
	if err != nil {
		t.Fatalf("read audit: %v", err)
	}
	if len(b) == 0 {
		t.Error("audit file is empty")
	}
	records, err := audit.ReadAll(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("parse audit: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	if records[0].Verb != "test.ro" {
		t.Errorf("verb = %q", records[0].Verb)
	}
}

func TestWrap_NilWriterStillRunsAction(t *testing.T) {
	called := false
	spec := verb.Spec{
		Name: "test.ro",
		Kind: policy.ReadOnly,
		Action: func(_ context.Context, _ *cli.Command) error {
			called = true
			return nil
		},
	}
	err := runWrapped(t, spec, nil, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !called {
		t.Error("action was not called")
	}
}

func TestWrap_RecordsFailureInAudit(t *testing.T) {
	w := newTestWriter(t)
	spec := verb.Spec{
		Name: "test.ro",
		Kind: policy.ReadOnly,
		Action: func(_ context.Context, _ *cli.Command) error {
			return errors.New("boom")
		},
	}
	err := runWrapped(t, spec, nil, w)
	if err == nil {
		t.Fatal("expected error")
	}
	b, _ := os.ReadFile(w.Path)
	records, _ := audit.ReadAll(bytes.NewReader(b))
	if len(records) != 1 || records[0].ExitCode != 1 {
		t.Errorf("records = %+v, want one record with exit_code=1", records)
	}
}

// runWrapped invokes the wrapped action in a way that mimics urfave/cli's
// real invocation shape. We pass an empty *cli.Command because Spec.ArgsFunc
// is the only code path that reads from it, and test specs set ArgsFunc
// when they care.
func runWrapped(t *testing.T, spec verb.Spec, verifier policy.TokenVerifier, w *audit.Writer) error {
	t.Helper()
	action := verb.Wrap(spec, verifier, w)
	return action(context.Background(), &cli.Command{Name: spec.Name})
}

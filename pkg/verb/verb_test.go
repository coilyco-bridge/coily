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

func newTestWriter(t *testing.T) *audit.Writer {
	t.Helper()
	w := &audit.Writer{
		Path: filepath.Join(t.TempDir(), "audit.jsonl"),
		Now:  func() time.Time { return time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC) },
	}
	// Close on cleanup so Windows can actually remove the TempDir. Lumberjack
	// keeps the fd open otherwise and the post-test RemoveAll fails.
	t.Cleanup(func() { _ = w.Close() })
	return w
}

func TestWrap_RunsAction(t *testing.T) {
	called := false
	spec := verb.Spec{
		Name: "test.ro",
		Action: func(_ context.Context, _ *cli.Command) error {
			called = true
			return nil
		},
	}
	if err := runWrapped(t, spec, newTestWriter(t)); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !called {
		t.Error("action was not called")
	}
}

func TestWrap_RejectsShellMetacharInArg(t *testing.T) {
	w := newTestWriter(t)
	spec := verb.Spec{
		Name: "test.ro",
		ArgsFunc: func(_ *cli.Command) (map[string]string, []string) {
			return map[string]string{"--thing": "foo;bar"}, nil
		},
		Action: func(_ context.Context, _ *cli.Command) error {
			t.Error("action should not be called when args fail policy")
			return nil
		},
	}
	err := runWrapped(t, spec, w)
	if !errors.Is(err, policy.ErrShellMeta) {
		t.Errorf("err = %v, want ErrShellMeta", err)
	}
	b, _ := os.ReadFile(w.Path)
	records, _ := audit.ReadAll(bytes.NewReader(b))
	if len(records) != 1 {
		t.Fatalf("got %d audit records on policy reject, want 1", len(records))
	}
	if records[0].Decision != audit.DecisionReject {
		t.Errorf("decision = %q, want %q", records[0].Decision, audit.DecisionReject)
	}
	if records[0].ExitCode != 1 {
		t.Errorf("exit_code = %d, want 1", records[0].ExitCode)
	}
}

func TestWrap_RejectsShellMetacharInPositional(t *testing.T) {
	spec := verb.Spec{
		Name: "test.ro",
		ArgsFunc: func(_ *cli.Command) (map[string]string, []string) {
			return nil, []string{"ok", "bad;bar"}
		},
		Action: func(_ context.Context, _ *cli.Command) error {
			t.Error("action should not be called when positional fails policy")
			return nil
		},
	}
	err := runWrapped(t, spec, newTestWriter(t))
	if !errors.Is(err, policy.ErrShellMeta) {
		t.Errorf("err = %v, want ErrShellMeta", err)
	}
}

func TestWrap_WritesAuditRecord(t *testing.T) {
	w := newTestWriter(t)
	spec := verb.Spec{
		Name:   "test.ro",
		Action: func(_ context.Context, _ *cli.Command) error { return nil },
	}
	if err := runWrapped(t, spec, w); err != nil {
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
	if records[0].Decision != audit.DecisionAccept {
		t.Errorf("decision = %q, want %q", records[0].Decision, audit.DecisionAccept)
	}
}

func TestWrap_NilWriterStillRunsAction(t *testing.T) {
	called := false
	spec := verb.Spec{
		Name: "test.ro",
		Action: func(_ context.Context, _ *cli.Command) error {
			called = true
			return nil
		},
	}
	if err := runWrapped(t, spec, nil); err != nil {
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
		Action: func(_ context.Context, _ *cli.Command) error {
			return errors.New("boom")
		},
	}
	err := runWrapped(t, spec, w)
	if err == nil {
		t.Fatal("expected error")
	}
	b, _ := os.ReadFile(w.Path)
	records, _ := audit.ReadAll(bytes.NewReader(b))
	if len(records) != 1 || records[0].ExitCode != 1 {
		t.Errorf("records = %+v, want one record with exit_code=1", records)
	}
}

func TestWrap_OnCompleteMutatesRecord(t *testing.T) {
	w := newTestWriter(t)
	spec := verb.Spec{
		Name:   "test.ro",
		Action: func(_ context.Context, _ *cli.Command) error { return nil },
		OnComplete: func(r *audit.Record) {
			r.Egress = []audit.EgressRow{
				{Host: "formulae.brew.sh", Decision: audit.EgressAllow, BytesUp: 100, BytesDown: 200, DurationMS: 5},
			}
		},
	}
	if err := runWrapped(t, spec, w); err != nil {
		t.Fatalf("run: %v", err)
	}
	b, _ := os.ReadFile(w.Path)
	records, _ := audit.ReadAll(bytes.NewReader(b))
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	if len(records[0].Egress) != 1 || records[0].Egress[0].Host != "formulae.brew.sh" {
		t.Errorf("egress = %+v, want one row for formulae.brew.sh", records[0].Egress)
	}
}

func TestWrap_CommitScopeOverrideBypassesFlagResolution(t *testing.T) {
	// CommitScopeOverride pins the audit row's commit-scope to a caller-
	// supplied path. Used by `coily exec` discovered-from-child verbs to
	// bind audits to the matched child repo, not cwd's git toplevel
	// (which often is not a repo). Setting a bogus override that wouldn't
	// resolve through gitToplevel proves it bypasses the auto path.
	w := newTestWriter(t)
	override := "/var/empty/not-a-repo"
	spec := verb.Spec{
		Name:                "test.ro",
		CommitScopeOverride: override,
		Action:              func(_ context.Context, _ *cli.Command) error { return nil },
	}
	if err := runWrapped(t, spec, w); err != nil {
		t.Fatalf("run: %v", err)
	}
	b, _ := os.ReadFile(w.Path)
	records, _ := audit.ReadAll(bytes.NewReader(b))
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	if records[0].CommitScope != override {
		t.Errorf("commit_scope = %q, want %q", records[0].CommitScope, override)
	}
}

// TestWrap_CommitScopeArgvHintFiresWhenFlagUnset proves the argv hook
// sets the audit row's commit-scope when neither --commit-scope nor
// $COILY_COMMIT_SCOPE was set explicitly. Used by `coily ops gh` to
// derive the scope from --repo coilysiren/<name> without forcing the
// operator to thread --commit-scope through every invocation.
func TestWrap_CommitScopeArgvHintFiresWhenFlagUnset(t *testing.T) {
	t.Setenv("COILY_COMMIT_SCOPE", "")
	w := newTestWriter(t)
	hinted := "/var/empty/hinted-by-argv"
	spec := verb.Spec{
		Name: "test.ro",
		CommitScopeArgvHint: func(_ []string) string {
			return hinted
		},
		Action: func(_ context.Context, _ *cli.Command) error { return nil },
	}
	if err := runWrapped(t, spec, w); err != nil {
		t.Fatalf("run: %v", err)
	}
	b, _ := os.ReadFile(w.Path)
	records, _ := audit.ReadAll(bytes.NewReader(b))
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	if records[0].CommitScope != hinted {
		t.Errorf("commit_scope = %q, want %q", records[0].CommitScope, hinted)
	}
}

// TestWrap_CommitScopeArgvHintLosesToEnv proves that a hint declines to
// override an explicit $COILY_COMMIT_SCOPE. The hint is a fallback, not a
// preempt, and operator-set values must always win.
func TestWrap_CommitScopeArgvHintLosesToEnv(t *testing.T) {
	envScope := "/var/empty/from-env"
	t.Setenv("COILY_COMMIT_SCOPE", envScope)
	w := newTestWriter(t)
	spec := verb.Spec{
		Name: "test.ro",
		CommitScopeArgvHint: func(_ []string) string {
			t.Error("hint should not fire when COILY_COMMIT_SCOPE is set")
			return "/var/empty/should-not-be-used"
		},
		Action: func(_ context.Context, _ *cli.Command) error { return nil },
	}
	if err := runWrapped(t, spec, w); err != nil {
		t.Fatalf("run: %v", err)
	}
	b, _ := os.ReadFile(w.Path)
	records, _ := audit.ReadAll(bytes.NewReader(b))
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	if records[0].CommitScope != envScope {
		t.Errorf("commit_scope = %q, want %q", records[0].CommitScope, envScope)
	}
}

// runWrapped invokes the wrapped action in a way that mimics urfave/cli's
// real invocation shape. We pass an empty *cli.Command because Spec.ArgsFunc
// is the only code path that reads from it, and test specs set ArgsFunc
// when they care.
func runWrapped(t *testing.T, spec verb.Spec, w *audit.Writer) error {
	t.Helper()
	action := verb.Wrap(spec, w)
	return action(context.Background(), &cli.Command{Name: spec.Name})
}

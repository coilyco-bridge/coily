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

type fakeVerifier struct {
	scopes string
	ok     bool
}

func (f fakeVerifier) VerifyScopes(_ string) (string, error) {
	if f.ok {
		return f.scopes, nil
	}
	return "", errors.New("bad token")
}

func newTestWriter(t *testing.T) *audit.Writer {
	t.Helper()
	return &audit.Writer{
		Path: filepath.Join(t.TempDir(), "audit.jsonl"),
		Now:  func() time.Time { return time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC) },
	}
}

// TestToken_PrefersFlagOverEnv covers the two-source token resolution used
// by every mutating verb's ArgsFunc. The --token flag wins when set;
// otherwise the $COILY_TOKEN env var is consulted. Regression for issue #1,
// where generated verbs dropped the flag entirely and the docstring-promised
// env fallback had no Getenv call behind it.
func TestToken_PrefersFlagOverEnv(t *testing.T) {
	t.Setenv(verb.TokenEnvVar, "env-tok")

	var captured string
	app := &cli.Command{
		Flags: []cli.Flag{&cli.StringFlag{Name: "token"}},
		Action: func(_ context.Context, c *cli.Command) error {
			captured = verb.Token(c)
			return nil
		},
	}
	if err := app.Run(context.Background(), []string{"app", "--token", "flag-tok"}); err != nil {
		t.Fatal(err)
	}
	if captured != "flag-tok" {
		t.Errorf("Token() = %q, want %q (flag should beat env)", captured, "flag-tok")
	}

	captured = ""
	app2 := &cli.Command{
		Flags: []cli.Flag{&cli.StringFlag{Name: "token"}},
		Action: func(_ context.Context, c *cli.Command) error {
			captured = verb.Token(c)
			return nil
		},
	}
	if err := app2.Run(context.Background(), []string{"app"}); err != nil {
		t.Fatal(err)
	}
	if captured != "env-tok" {
		t.Errorf("Token() with unset flag = %q, want %q (env fallback)", captured, "env-tok")
	}
}

// TestTokenFromEnv covers the collision path: generated leaves whose
// underlying binary already owns --token (e.g. kubectl config
// set-credentials) read the coily confirmation token strictly from
// $COILY_TOKEN, never from the cli.Command flag (which would carry the
// native value).
func TestTokenFromEnv(t *testing.T) {
	t.Setenv(verb.TokenEnvVar, "only-env")
	if got := verb.TokenFromEnv(); got != "only-env" {
		t.Errorf("TokenFromEnv() = %q, want %q", got, "only-env")
	}
	t.Setenv(verb.TokenEnvVar, "")
	if got := verb.TokenFromEnv(); got != "" {
		t.Errorf("TokenFromEnv() with unset env = %q, want empty", got)
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
		Scope:  "test.mut:write",
		Action: func(_ context.Context, _ *cli.Command) error { return nil },
	}
	err := runWrapped(t, spec, fakeVerifier{ok: true, scopes: "test.mut:write"}, newTestWriter(t))
	if !errors.Is(err, policy.ErrTokenRequired) {
		t.Errorf("err = %v, want ErrTokenRequired", err)
	}
}

func TestWrap_KindFuncOverridesStaticKind(t *testing.T) {
	// Static Kind says ReadOnly, KindFunc says Mutating. KindFunc wins,
	// so the missing token must trigger ErrTokenRequired.
	spec := verb.Spec{
		Name:     "test.dyn",
		Kind:     policy.ReadOnly,
		KindFunc: func(_ *cli.Command) policy.Kind { return policy.Mutating },
		Action:   func(_ context.Context, _ *cli.Command) error { return nil },
	}
	err := runWrapped(t, spec, fakeVerifier{ok: true}, newTestWriter(t))
	if !errors.Is(err, policy.ErrTokenRequired) {
		t.Errorf("err = %v, want ErrTokenRequired", err)
	}
}

func TestWrap_KindFuncReadOnlyAllowsNoToken(t *testing.T) {
	// Static Kind says Mutating, KindFunc says ReadOnly. KindFunc wins,
	// so no token is needed and the action runs.
	called := false
	spec := verb.Spec{
		Name:     "test.dyn",
		Kind:     policy.Mutating,
		KindFunc: func(_ *cli.Command) policy.Kind { return policy.ReadOnly },
		Action: func(_ context.Context, _ *cli.Command) error {
			called = true
			return nil
		},
	}
	if err := runWrapped(t, spec, nil, newTestWriter(t)); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !called {
		t.Error("action was not called")
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

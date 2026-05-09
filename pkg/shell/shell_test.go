package shell_test

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coilysiren/coily/pkg/shell"
)

func TestExec_RunsBinaryWithArgv(t *testing.T) {
	var out bytes.Buffer
	r := &shell.Runner{Stdout: &out}
	// `echo hello world` — available on every Unix.
	if err := r.Exec(context.Background(), "echo", "hello", "world"); err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != "hello world" {
		t.Errorf("stdout = %q, want %q", got, "hello world")
	}
}

func TestExec_EmptyBinaryErrors(t *testing.T) {
	r := &shell.Runner{}
	err := r.Exec(context.Background(), "")
	if !errors.Is(err, shell.ErrEmptyBinary) {
		t.Errorf("err = %v, want ErrEmptyBinary", err)
	}
}

func TestCapture_ReturnsStdout(t *testing.T) {
	r := &shell.Runner{}
	out, err := r.Capture(context.Background(), "echo", "captured")
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "captured" {
		t.Errorf("got %q, want %q", got, "captured")
	}
}

func TestCapture_EmptyBinaryErrors(t *testing.T) {
	r := &shell.Runner{}
	if _, err := r.Capture(context.Background(), ""); !errors.Is(err, shell.ErrEmptyBinary) {
		t.Errorf("err = %v, want ErrEmptyBinary", err)
	}
}

func TestResolve_PluggedInReplacesPathLookup(t *testing.T) {
	called := 0
	r := &shell.Runner{
		Resolve: func(_ string) (string, error) {
			called++
			return "/bin/echo", nil
		},
	}
	if err := r.Exec(context.Background(), "doesnotexist", "x"); err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if called != 1 {
		t.Errorf("resolver called %d times, want 1", called)
	}
}

func TestExec_ResolverErrorPropagates(t *testing.T) {
	r := &shell.Runner{
		Resolve: func(_ string) (string, error) { return "", errors.New("nope") },
	}
	if err := r.Exec(context.Background(), "x"); err == nil {
		t.Error("expected error")
	}
}

func TestExec_EnvNilLeavesParentEnvironment(t *testing.T) {
	t.Setenv("COILY_TEST_VAR", "from-parent")
	var out bytes.Buffer
	r := &shell.Runner{Stdout: &out}
	if err := r.Exec(context.Background(), "sh", "-c", "echo $COILY_TEST_VAR"); err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != "from-parent" {
		t.Errorf("got %q, want from-parent (parent env should be inherited)", got)
	}
}

func TestExec_EnvAppendsToParent(t *testing.T) {
	t.Setenv("COILY_TEST_PARENT", "parent-value")
	var out bytes.Buffer
	r := &shell.Runner{
		Stdout: &out,
		Env:    []string{"COILY_TEST_INJECTED=injected-value"},
	}
	if err := r.Exec(context.Background(), "sh", "-c", "echo $COILY_TEST_PARENT/$COILY_TEST_INJECTED"); err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != "parent-value/injected-value" {
		t.Errorf("got %q, want parent-value/injected-value", got)
	}
}

func TestExecIn_RunsInDirectory(t *testing.T) {
	// ExecIn forces cmd.Dir so the child process resolves relative paths
	// against the caller-supplied directory rather than cwd. Used by
	// `coily exec` discovered-from-child verbs to run a child repo's
	// declared argv from that repo's root.
	dir := t.TempDir()
	var out bytes.Buffer
	r := &shell.Runner{Stdout: &out}
	if err := r.ExecIn(context.Background(), dir, "pwd"); err != nil {
		t.Fatalf("ExecIn: %v", err)
	}
	got := strings.TrimSpace(out.String())
	gotR, _ := filepath.EvalSymlinks(got)
	wantR, _ := filepath.EvalSymlinks(dir)
	if gotR != wantR {
		t.Errorf("pwd = %q, want %q", got, dir)
	}
}

func TestExecIn_EmptyBinaryErrors(t *testing.T) {
	r := &shell.Runner{}
	if err := r.ExecIn(context.Background(), t.TempDir(), ""); !errors.Is(err, shell.ErrEmptyBinary) {
		t.Errorf("err = %v, want ErrEmptyBinary", err)
	}
}

func TestPathResolver_ReportsMissingBinary(t *testing.T) {
	_, err := shell.PathResolver("coily-certainly-does-not-exist-xyzzy")
	if err == nil {
		t.Error("expected error for missing binary")
	}
}

package scope_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coilysiren/coily/pkg/scope"
)

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"-C", dir, "config", "user.email", "test@example.com"},
		{"-C", dir, "config", "user.name", "test"},
	} {
		// `git init -q` runs inside dir; the rest use -C dir already.
		cmd := exec.Command("git", args...)
		if args[0] == "init" {
			cmd.Dir = dir
		}
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	// Resolve symlinks (macOS /var → /private/var) so equality holds against
	// what `git rev-parse --show-toplevel` returns.
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("resolve symlinks: %v", err)
	}
	return resolved
}

func TestResolve_AutoInsideRepoReturnsToplevel(t *testing.T) {
	dir := initRepo(t)
	got, err := scope.Resolve("auto", "", dir)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != dir {
		t.Errorf("got %q, want %q", got, dir)
	}
}

func TestResolve_AutoInsideSubdirReturnsToplevel(t *testing.T) {
	dir := initRepo(t)
	sub := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := scope.Resolve("auto", "", sub)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != dir {
		t.Errorf("got %q, want %q (toplevel of subdir)", got, dir)
	}
}

func TestResolve_AutoOutsideRepoErrors(t *testing.T) {
	// t.TempDir() is not a git repo. "auto" must fail loudly.
	dir := t.TempDir()
	_, err := scope.Resolve("auto", "", dir)
	if err == nil {
		t.Fatal("Resolve: got nil err, want ErrNotInRepo")
	}
	if !errors.Is(err, scope.ErrNotInRepo) {
		t.Errorf("err = %v, want ErrNotInRepo", err)
	}
}

func TestResolve_DashOptsOut(t *testing.T) {
	got, err := scope.Resolve("-", "", t.TempDir())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want empty (opt-out)", got)
	}
}

func TestResolve_NoneOptsOut(t *testing.T) {
	got, err := scope.Resolve("none", "", t.TempDir())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestResolve_ExplicitPathReturnedAbsolute(t *testing.T) {
	dir := t.TempDir()
	got, err := scope.Resolve("/some/repo", "", dir)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != "/some/repo" {
		t.Errorf("got %q, want /some/repo", got)
	}
}

func TestResolve_RelativePathJoinsCWD(t *testing.T) {
	cwd := t.TempDir()
	got, err := scope.Resolve("subdir", "", cwd)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want := filepath.Join(cwd, "subdir")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolve_EmptyFlagFallsBackToEnv(t *testing.T) {
	dir := t.TempDir()
	got, err := scope.Resolve("", "/from/env", dir)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != "/from/env" {
		t.Errorf("got %q, want /from/env", got)
	}
}

func TestResolve_FlagBeatsEnv(t *testing.T) {
	dir := t.TempDir()
	got, err := scope.Resolve("/from/flag", "/from/env", dir)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != "/from/flag" {
		t.Errorf("got %q, want /from/flag", got)
	}
}

func TestResolve_AutoErrorMessageIsActionable(t *testing.T) {
	_, err := scope.Resolve("auto", "", t.TempDir())
	if err == nil {
		t.Fatal("want error")
	}
	if !strings.Contains(err.Error(), "--commit-scope") {
		t.Errorf("err message %q should mention --commit-scope", err.Error())
	}
}

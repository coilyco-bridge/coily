package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/shell"
)

func TestHoistDashC(t *testing.T) {
	cases := []struct {
		name    string
		argv    []string
		wantDir string
		wantRem []string
	}{
		{"no -C", []string{"-m", "x", "--", "a"}, "", []string{"-m", "x", "--", "a"}},
		{"leading -C", []string{"-C", "/repo", "-m", "x", "--", "a"}, "/repo", []string{"-m", "x", "--", "a"}},
		{"lone -C not hoisted", []string{"-C"}, "", []string{"-C"}},
		{"-C not leading stays", []string{"-m", "x", "-C", "/r"}, "", []string{"-m", "x", "-C", "/r"}},
	}
	for _, c := range cases {
		dir, rem := hoistDashC(c.argv)
		if dir != c.wantDir || !reflect.DeepEqual(rem, c.wantRem) {
			t.Errorf("%s: hoistDashC(%v) = (%q, %v), want (%q, %v)", c.name, c.argv, dir, rem, c.wantDir, c.wantRem)
		}
	}
}

func TestSplitCommitArgs(t *testing.T) {
	cases := []struct {
		name       string
		argv       []string
		wantFlags  []string
		wantPaths  []string
		wantErrSub string
	}{
		{"flags then paths", []string{"-m", "msg", "--", "a", "b"}, []string{"-m", "msg"}, []string{"a", "b"}, ""},
		{"empty paths still splits", []string{"-m", "msg", "--"}, []string{"-m", "msg"}, []string{}, ""},
		{"no separator errors", []string{"-m", "msg", "a"}, nil, nil, "missing '--' separator"},
	}
	for _, c := range cases {
		flags, paths, err := splitCommitArgs(c.argv)
		if c.wantErrSub != "" {
			if err == nil || !strings.Contains(err.Error(), c.wantErrSub) {
				t.Errorf("%s: want error containing %q, got %v", c.name, c.wantErrSub, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: unexpected error %v", c.name, err)
			continue
		}
		if !reflect.DeepEqual(flags, c.wantFlags) || !reflect.DeepEqual(paths, c.wantPaths) {
			t.Errorf("%s: got flags=%v paths=%v, want flags=%v paths=%v", c.name, flags, paths, c.wantFlags, c.wantPaths)
		}
	}
}

func TestHasMessageSource(t *testing.T) {
	yes := [][]string{
		{"-m", "msg"}, {"-mmsg"}, {"--message", "msg"}, {"--message=msg"},
		{"-F", "file"}, {"-Ffile"}, {"--file", "f"}, {"--file=f"},
		{"-s", "-m", "x"},
	}
	no := [][]string{
		{}, {"-s"}, {"--signoff"}, {"--amend"}, {"-mistake-not-a-flag-but-still-counts"},
	}
	for _, f := range yes {
		if !hasMessageSource(f) {
			t.Errorf("hasMessageSource(%v) = false, want true", f)
		}
	}
	// "-mistake..." starts with -m and len>2, so it DOES count as a message
	// source (git would read it as -m"istake..."). Drop it from the no-set.
	for _, f := range no {
		if len(f) == 1 && strings.HasPrefix(f[0], "-m") && len(f[0]) > 2 {
			continue
		}
		if hasMessageSource(f) {
			t.Errorf("hasMessageSource(%v) = true, want false", f)
		}
	}
}

func TestHasEditorFlag(t *testing.T) {
	if !hasEditorFlag([]string{"-m", "x", "-e"}) {
		t.Error("hasEditorFlag should detect -e")
	}
	if !hasEditorFlag([]string{"--edit"}) {
		t.Error("hasEditorFlag should detect --edit")
	}
	if hasEditorFlag([]string{"-m", "x"}) {
		t.Error("hasEditorFlag false positive")
	}
}

func TestGitArgv(t *testing.T) {
	got := gitArgv("/repo", "commit", []string{"-m", "msg"}, []string{"a", "b"})
	want := []string{"-C", "/repo", "commit", "-m", "msg", "--", "a", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("gitArgv with dir = %v, want %v", got, want)
	}
	got = gitArgv("", "reset", []string{"-q", "HEAD"}, []string{"a"})
	want = []string{"reset", "-q", "HEAD", "--", "a"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("gitArgv no dir = %v, want %v", got, want)
	}
}

func TestNewPrivateIndexRemoved(t *testing.T) {
	path, cleanup, err := newPrivateIndex()
	if err != nil {
		t.Fatalf("newPrivateIndex: %v", err)
	}
	defer cleanup()
	if path == "" {
		t.Fatal("empty index path")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("placeholder index %q should be removed before return, stat err = %v", path, err)
	}
}

// gitAvailable reports whether a real git is on PATH for the integration test.
func gitAvailable(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// TestRunGitCommitIsolatesSharedIndex is the regression test for coily#7: a
// commit of one path must not sweep in another path that a concurrent
// session left staged in the shared index, and must carry its own message.
func TestRunGitCommitIsolatesSharedIndex(t *testing.T) {
	gitAvailable(t)
	repo := t.TempDir()
	runGit(t, repo, "init", "-q")
	runGit(t, repo, "config", "user.email", "t@t.t")
	runGit(t, repo, "config", "user.name", "t")
	writeFile(t, filepath.Join(repo, "a.txt"), "base\n")
	writeFile(t, filepath.Join(repo, "b.txt"), "base\n")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-qm", "init")

	writeFile(t, filepath.Join(repo, "a.txt"), "A-change\n")
	writeFile(t, filepath.Join(repo, "b.txt"), "B-change\n")
	// Simulate the other session's staged work in the SHARED index.
	runGit(t, repo, "add", "b.txt")

	r := &Runner{Runner: &shell.Runner{Stdout: os.Stderr, Stderr: os.Stderr}}
	if err := r.runGitCommit(context.Background(), []string{"-C", repo, "-m", "fix: A only", "--", "a.txt"}); err != nil {
		t.Fatalf("runGitCommit: %v", err)
	}

	// HEAD must contain only a.txt, with our message and content.
	if files := gitOut(t, repo, "show", "--name-only", "--format=", "HEAD"); files != "a.txt" {
		t.Errorf("HEAD changed files = %q, want only a.txt", files)
	}
	if msg := gitOut(t, repo, "log", "-1", "--format=%s"); msg != "fix: A only" {
		t.Errorf("HEAD message = %q, want %q", msg, "fix: A only")
	}
	if content := gitOut(t, repo, "show", "HEAD:a.txt"); content != "A-change" {
		t.Errorf("committed a.txt = %q, want A-change", content)
	}
	// b.txt must remain staged with the other session's content - untouched.
	if staged := gitOut(t, repo, "cat-file", "-p", ":b.txt"); staged != "B-change" {
		t.Errorf("b.txt staged content = %q, want B-change (untouched)", staged)
	}
	// a.txt should read clean after the post-commit resync.
	if st := gitOut(t, repo, "status", "--short", "--", "a.txt"); st != "" {
		t.Errorf("a.txt status after resync = %q, want clean", st)
	}
}

func TestRunGitCommitRejects(t *testing.T) {
	r := &Runner{Runner: &shell.Runner{Stdout: os.Stderr, Stderr: os.Stderr}}
	cases := []struct {
		name   string
		argv   []string
		errSub string
	}{
		{"no separator", []string{"-m", "x"}, "missing '--' separator"},
		{"no paths", []string{"-m", "x", "--"}, "name the files"},
		{"no message", []string{"--", "a.txt"}, "pass -m/-F"},
		{"editor flag", []string{"-m", "x", "-e", "--", "a.txt"}, "-e/--edit is refused"},
	}
	for _, c := range cases {
		err := r.runGitCommit(context.Background(), c.argv)
		if err == nil || !strings.Contains(err.Error(), c.errSub) {
			t.Errorf("%s: want error containing %q, got %v", c.name, c.errSub, err)
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

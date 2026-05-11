package workdir_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/coilysiren/coily/pkg/workdir"
)

func TestDetectFrom_EnvOverrideWins(t *testing.T) {
	t.Setenv(workdir.OverrideEnv, "/forced/path")
	got := workdir.DetectFrom(t.TempDir())
	if got.Source != workdir.SourceEnv {
		t.Errorf("source = %q, want env", got.Source)
	}
	if got.Path != "/forced/path" {
		t.Errorf("path = %q, want /forced/path", got.Path)
	}
}

func TestDetectFrom_EnvRelativeJoinsCwd(t *testing.T) {
	cwd := t.TempDir()
	t.Setenv(workdir.OverrideEnv, "sub")
	got := workdir.DetectFrom(cwd)
	want := filepath.Join(cwd, "sub")
	if got.Path != want {
		t.Errorf("path = %q, want %q", got.Path, want)
	}
}

func TestDetectFrom_GitRootDetected(t *testing.T) {
	t.Setenv(workdir.OverrideEnv, "")
	t.Setenv("HOME", t.TempDir()) // dodge coilysiren rule
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got := workdir.DetectFrom(sub)
	if got.Source != workdir.SourceGit {
		t.Errorf("source = %q, want git", got.Source)
	}
	if got.Path != root {
		t.Errorf("path = %q, want %q", got.Path, root)
	}
}

func TestDetectFrom_GitFileCountsAsWorktree(t *testing.T) {
	t.Setenv(workdir.OverrideEnv, "")
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: /elsewhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := workdir.DetectFrom(root)
	if got.Source != workdir.SourceGit {
		t.Errorf("source = %q, want git", got.Source)
	}
}

func TestDetectFrom_CoilysirenParentPicksFirstSegment(t *testing.T) {
	t.Setenv(workdir.OverrideEnv, "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := filepath.Join(home, "projects", "coilysiren", "myrepo")
	deep := filepath.Join(repo, "pkg", "x")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	got := workdir.DetectFrom(deep)
	if got.Source != workdir.SourceCoilysiren {
		t.Errorf("source = %q, want coilysiren", got.Source)
	}
	if got.Path != repo {
		t.Errorf("path = %q, want %q", got.Path, repo)
	}
}

func TestDetectFrom_GitBeatsCoilysiren(t *testing.T) {
	t.Setenv(workdir.OverrideEnv, "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := filepath.Join(home, "projects", "coilysiren", "myrepo")
	sub := filepath.Join(repo, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(sub, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := workdir.DetectFrom(sub)
	if got.Source != workdir.SourceGit {
		t.Errorf("source = %q, want git (more specific than coilysiren)", got.Source)
	}
	if got.Path != sub {
		t.Errorf("path = %q, want %q", got.Path, sub)
	}
}

func TestDetectFrom_FallsBackToCwd(t *testing.T) {
	t.Setenv(workdir.OverrideEnv, "")
	t.Setenv("HOME", t.TempDir())
	cwd := t.TempDir()
	got := workdir.DetectFrom(cwd)
	if got.Source != workdir.SourceCWD {
		t.Errorf("source = %q, want cwd", got.Source)
	}
	if got.Path != filepath.Clean(cwd) {
		t.Errorf("path = %q, want %q", got.Path, cwd)
	}
}

func TestDetectFrom_EmptyCwd(t *testing.T) {
	t.Setenv(workdir.OverrideEnv, "")
	t.Setenv("HOME", t.TempDir())
	got := workdir.DetectFrom("")
	if got.Source != workdir.SourceCWD {
		t.Errorf("source = %q, want cwd", got.Source)
	}
}

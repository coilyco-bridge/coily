package gittree

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initRepo bootstraps a tiny local-only repo at dir, with one committed file
// on a tracking branch. Returns the path to a co-located bare upstream that
// the working repo's `main` branch tracks.
func initRepo(t *testing.T) (workdir, upstream string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	root := t.TempDir()
	workdir = filepath.Join(root, "work")
	upstream = filepath.Join(root, "upstream.git")
	if err := os.Mkdir(workdir, 0o755); err != nil {
		t.Fatal(err)
	}
	mustGit(t, root, "init", "--bare", "--initial-branch=main", upstream)
	mustGit(t, workdir, "init", "--initial-branch=main")
	mustGit(t, workdir, "config", "user.email", "test@example.com")
	mustGit(t, workdir, "config", "user.name", "test")
	mustGit(t, workdir, "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(workdir, "README"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, workdir, "add", "README")
	mustGit(t, workdir, "commit", "-m", "init")
	mustGit(t, workdir, "remote", "add", "origin", upstream)
	mustGit(t, workdir, "push", "-u", "origin", "main")
	return workdir, upstream
}

func mustGit(t *testing.T, cwd string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", cwd}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func TestCheckClean_clean(t *testing.T) {
	workdir, _ := initRepo(t)
	st, err := CheckClean(workdir)
	if err != nil {
		t.Fatalf("CheckClean: %v", err)
	}
	if !st.Clean {
		t.Fatalf("expected Clean=true, got %+v", st)
	}
}

func TestCheckClean_dirtyTracked(t *testing.T) {
	workdir, _ := initRepo(t)
	if err := os.WriteFile(filepath.Join(workdir, "README"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := CheckClean(workdir)
	if err != nil {
		t.Fatalf("CheckClean: %v", err)
	}
	if st.Clean {
		t.Fatal("expected Clean=false for tracked-modified file")
	}
	if !strings.Contains(st.Reason, "dirty") {
		t.Errorf("Reason = %q, want it to mention 'dirty'", st.Reason)
	}
	if !strings.Contains(st.Status, "README") {
		t.Errorf("Status missing README: %q", st.Status)
	}
}

func TestCheckClean_untracked(t *testing.T) {
	workdir, _ := initRepo(t)
	if err := os.WriteFile(filepath.Join(workdir, "new.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := CheckClean(workdir)
	if err != nil {
		t.Fatalf("CheckClean: %v", err)
	}
	if st.Clean {
		t.Fatal("expected Clean=false for untracked file")
	}
	if !strings.Contains(st.Status, "??") {
		t.Errorf("Status missing untracked marker: %q", st.Status)
	}
}

func TestCheckClean_behindUpstream(t *testing.T) {
	workdir, upstream := initRepo(t)
	// Make a peer clone, push a new commit to upstream, leaving workdir behind.
	peer := filepath.Join(t.TempDir(), "peer")
	mustGit(t, ".", "clone", upstream, peer)
	mustGit(t, peer, "config", "user.email", "test@example.com")
	mustGit(t, peer, "config", "user.name", "test")
	mustGit(t, peer, "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(peer, "ahead.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, peer, "add", "ahead.txt")
	mustGit(t, peer, "commit", "-m", "ahead")
	mustGit(t, peer, "push")

	st, err := CheckClean(workdir)
	if err != nil {
		t.Fatalf("CheckClean: %v", err)
	}
	if st.Clean {
		t.Fatalf("expected Clean=false for behind-upstream state, got %+v", st)
	}
	if !strings.Contains(st.Reason, "behind") {
		t.Errorf("Reason = %q, want 'behind'", st.Reason)
	}
}

func TestCheckClean_aheadOnlyIsClean(t *testing.T) {
	workdir, _ := initRepo(t)
	if err := os.WriteFile(filepath.Join(workdir, "x.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, workdir, "add", "x.txt")
	mustGit(t, workdir, "commit", "-m", "ahead-only")
	// Don't push; we are ahead but not behind, with no other dirt.
	st, err := CheckClean(workdir)
	if err != nil {
		t.Fatalf("CheckClean: %v", err)
	}
	if !st.Clean {
		t.Fatalf("ahead-only state should be Clean per issue 86 spec, got %+v", st)
	}
}

func TestCheckClean_noUpstream(t *testing.T) {
	workdir, _ := initRepo(t)
	mustGit(t, workdir, "checkout", "-b", "feature")
	st, err := CheckClean(workdir)
	if err != nil {
		t.Fatalf("CheckClean: %v", err)
	}
	if st.Clean {
		t.Fatal("expected Clean=false for branch with no upstream")
	}
	if !strings.Contains(st.Reason, "upstream") {
		t.Errorf("Reason = %q, want it to mention 'upstream'", st.Reason)
	}
}

func TestCheckClean_notARepo(t *testing.T) {
	dir := t.TempDir()
	if _, err := CheckClean(dir); err == nil {
		t.Fatal("expected error for non-repo path")
	}
}

func TestFormatRefusal_namesVerbAndOverride(t *testing.T) {
	st := &State{
		Reason:   "working tree is dirty",
		Recovery: "  git commit\n",
		Status:   " M README\n",
	}
	msg := st.FormatRefusal("repo.test")
	for _, want := range []string{"repo.test", "working tree", "git commit", "--audit-override-dirty"} {
		if !strings.Contains(msg, want) {
			t.Errorf("FormatRefusal missing %q; got:\n%s", want, msg)
		}
	}
}

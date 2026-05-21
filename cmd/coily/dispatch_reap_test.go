package main

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestParseIssueDirName pins the "issue-N" worktree dir name parse. The
// reaper walks <root>/<repo>/issue-<N> dirs, so anything that is not that
// shape (a stray file, a non-issue dir) must be rejected, not reaped.
func TestParseIssueDirName(t *testing.T) {
	cases := []struct {
		name   string
		wantN  int
		wantOK bool
	}{
		{"issue-683", 683, true},
		{"issue-1", 1, true},
		{"issue-0", 0, false},
		{"issue-", 0, false},
		{"issue-abc", 0, false},
		{"scratch", 0, false},
		{"issue-12x", 0, false},
	}
	for _, tc := range cases {
		gotN, gotOK := parseIssueDirName(tc.name)
		if gotN != tc.wantN || gotOK != tc.wantOK {
			t.Errorf("parseIssueDirName(%q) = (%d,%v), want (%d,%v)",
				tc.name, gotN, gotOK, tc.wantN, tc.wantOK)
		}
	}
}

// TestReapDispatchWorktrees_RemovesMergedOnly is the core contract from
// #300: a merged worktree is removed, an unmerged one is left in place,
// and a worktree whose removal fails is skipped rather than reported as
// removed.
func TestReapDispatchWorktrees_RemovesMergedOnly(t *testing.T) {
	r := newTestRunner(t)
	root := t.TempDir()
	prevRoot := dispatchWorktreeRootOverride
	dispatchWorktreeRootOverride = root
	t.Cleanup(func() { dispatchWorktreeRootOverride = prevRoot })

	// Two repos, each with a merged + an unmerged worktree, plus a stray
	// file the reaper must ignore.
	for _, dir := range []string{
		filepath.Join(root, "coily", "issue-1"),          // merged
		filepath.Join(root, "coily", "issue-2"),          // unmerged
		filepath.Join(root, "agentic-os-kai", "issue-5"), // merged
		filepath.Join(root, "agentic-os-kai", "issue-7"), // remove fails
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "coily", "stray.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write stray file: %v", err)
	}

	prevRepoPath := reapLocalRepoPath
	reapLocalRepoPath = func(repo string) (string, bool) { return "/fake/" + repo, true }
	t.Cleanup(func() { reapLocalRepoPath = prevRepoPath })

	// issue-1 and issue-5 are merged; issue-2 is not; issue-7 is merged
	// but its removal fails.
	merged := map[string]bool{"dispatch/issue-1": true, "dispatch/issue-5": true, "dispatch/issue-7": true}
	prevReapable := worktreeReapable
	worktreeReapable = func(_ context.Context, _ *Runner, _, branch string) bool {
		return merged[branch]
	}
	t.Cleanup(func() { worktreeReapable = prevReapable })

	var removeCalls []string
	prevRemove := reapRemoveWorktree
	reapRemoveWorktree = func(_ context.Context, _ *Runner, _, worktreePath, branch string) error {
		removeCalls = append(removeCalls, worktreePath)
		if branch == "dispatch/issue-7" {
			return os.ErrPermission // simulate a dirty / locked worktree
		}
		return nil
	}
	t.Cleanup(func() { reapRemoveWorktree = prevRemove })

	removed, err := reapDispatchWorktrees(context.Background(), r)
	if err != nil {
		t.Fatalf("reapDispatchWorktrees: %v", err)
	}

	sort.Strings(removed)
	want := []string{
		filepath.Join(root, "agentic-os-kai", "issue-5"),
		filepath.Join(root, "coily", "issue-1"),
	}
	sort.Strings(want)
	if strings.Join(removed, "|") != strings.Join(want, "|") {
		t.Errorf("removed = %v, want %v", removed, want)
	}
	// issue-2 (unmerged) must never reach reapRemoveWorktree.
	for _, c := range removeCalls {
		if strings.HasSuffix(c, filepath.Join("coily", "issue-2")) {
			t.Errorf("unmerged worktree issue-2 was passed to reapRemoveWorktree")
		}
	}
}

// TestReapDispatchWorktrees_NoRoot verifies a missing dispatch-worktree
// root is a clean no-op, not an error: a host that has never dispatched
// must be able to run the reaper.
func TestReapDispatchWorktrees_NoRoot(t *testing.T) {
	r := newTestRunner(t)
	prevRoot := dispatchWorktreeRootOverride
	dispatchWorktreeRootOverride = filepath.Join(t.TempDir(), "does-not-exist")
	t.Cleanup(func() { dispatchWorktreeRootOverride = prevRoot })

	removed, err := reapDispatchWorktrees(context.Background(), r)
	if err != nil {
		t.Fatalf("reapDispatchWorktrees on missing root should be a no-op, got err: %v", err)
	}
	if len(removed) != 0 {
		t.Errorf("removed = %v, want empty", removed)
	}
}

// TestDispatchHasReapSubverb proves the reap maintenance verb hangs off
// the dispatch parent alongside the two modes.
func TestDispatchHasReapSubverb(t *testing.T) {
	r := newTestRunner(t)
	cmd := r.dispatchCommand()
	for _, sub := range cmd.Commands {
		if sub.Name == "reap" {
			return
		}
	}
	t.Error("dispatch parent missing `reap` subverb")
}

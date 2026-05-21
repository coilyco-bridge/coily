package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/urfave/cli/v3"
)

// dispatch_reap.go closes the worktree lifecycle opened by
// dispatch_interactive.go. ensureDispatchWorktree creates one git worktree
// per dispatched issue and nothing ever removed it, so worktrees and
// dispatch/issue-N branches piled up under the dispatch-worktree root
// forever. The reaper removes every worktree whose branch is fully merged
// into main (coilysiren/coily#300).

// reapLocalRepoPath resolves a repo's local checkout and reports whether
// it exists. Seam: tests swap it so reapDispatchWorktrees can run against
// tempdir repos instead of the real ~/projects/coilysiren tree.
var reapLocalRepoPath = func(repo string) (string, bool) {
	p, err := localRepoPath(repo)
	if err != nil {
		return "", false
	}
	st, statErr := os.Stat(p)
	return p, statErr == nil && st.IsDir()
}

// worktreeReapable reports whether a dispatch worktree is safe to remove:
// its branch is either gone (already cleaned up) or fully merged into
// main. An existing, unmerged branch is never reapable - that is
// in-flight work. Seam: tests swap it to avoid real git.
var worktreeReapable = func(ctx context.Context, r *Runner, repoPath, branch string) bool {
	// Branch missing -> nothing unmerged can be lost; reapable.
	if _, err := r.Runner.Capture(ctx, "git", "-C", repoPath,
		"rev-parse", "--verify", "--quiet", "refs/heads/"+branch); err != nil {
		return true
	}
	// Branch present -> reapable only if it is an ancestor of main.
	_, err := r.Runner.Capture(ctx, "git", "-C", repoPath,
		"merge-base", "--is-ancestor", branch, "main")
	return err == nil
}

// reapRemoveWorktree removes one merged worktree, deletes its branch, and
// prunes git's worktree metadata. `git worktree remove` (no --force)
// refuses a dirty worktree, so uncommitted work in a merged worktree is
// preserved rather than silently dropped - the caller skips and warns.
// Seam: tests swap it.
var reapRemoveWorktree = func(ctx context.Context, r *Runner, repoPath, worktreePath, branch string) error {
	if _, err := r.Runner.Capture(ctx, "git", "-C", repoPath,
		"worktree", "remove", worktreePath); err != nil {
		return err
	}
	// Branch delete + prune are best-effort: the worktree is already
	// gone, so a leftover branch ref or stale metadata is cosmetic.
	_, _ = r.Runner.Capture(ctx, "git", "-C", repoPath, "branch", "-D", branch)
	_, _ = r.Runner.Capture(ctx, "git", "-C", repoPath, "worktree", "prune")
	return nil
}

// parseIssueDirName extracts N from an "issue-N" worktree directory name.
func parseIssueDirName(name string) (int, bool) {
	rest, ok := strings.CutPrefix(name, "issue-")
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(rest)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// reapRepoWorktrees removes every merged worktree under one repo's
// dispatch-worktree directory and returns the removed paths. Pulled out
// of reapDispatchWorktrees so the outer walk stays under the cognitive
// complexity threshold.
func reapRepoWorktrees(ctx context.Context, r *Runner, root, repo string) []string {
	repoPath, ok := reapLocalRepoPath(repo)
	if !ok {
		return nil
	}
	issueEntries, err := os.ReadDir(filepath.Join(root, repo))
	if err != nil {
		return nil
	}
	var removed []string
	for _, issueEntry := range issueEntries {
		n, ok := parseIssueDirName(issueEntry.Name())
		if !issueEntry.IsDir() || !ok {
			continue
		}
		worktreePath := filepath.Join(root, repo, issueEntry.Name())
		branch := dispatchWorktreeBranch(n)
		if !worktreeReapable(ctx, r, repoPath, branch) {
			continue
		}
		if err := reapRemoveWorktree(ctx, r, repoPath, worktreePath, branch); err != nil {
			fmt.Fprintf(os.Stderr, "dispatch reap: skip %s: %v\n", worktreePath, err)
			continue
		}
		removed = append(removed, worktreePath)
	}
	return removed
}

// reapDispatchWorktrees walks the dispatch-worktree root and removes every
// worktree whose branch is merged into its repo's main. Best-effort: a
// failure on one worktree (dirty tree, missing repo) is logged and
// skipped, never aborts the sweep. Returns the removed worktree paths.
func reapDispatchWorktrees(ctx context.Context, r *Runner) ([]string, error) {
	root, err := dispatchWorktreeRoot()
	if err != nil {
		return nil, err
	}
	repoEntries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read dispatch-worktree root %s: %w", root, err)
	}
	var removed []string
	for _, repoEntry := range repoEntries {
		if !repoEntry.IsDir() {
			continue
		}
		removed = append(removed, reapRepoWorktrees(ctx, r, root, repoEntry.Name())...)
	}
	return removed, nil
}

// dispatchReapCommand is the explicit all-repos sweep. The same reaper
// runs automatically at the start of every `coily dispatch interactive`,
// so this verb is the on-demand version for cleaning up between
// dispatches.
func (r *Runner) dispatchReapCommand() *cli.Command {
	return &cli.Command{
		Name:  "reap",
		Usage: "Remove dispatch worktrees whose branch is already merged into main.",
		Description: `reap walks ~/projects/coilysiren/.dispatch-worktrees/<repo>/issue-*
and removes every worktree whose dispatch/issue-N branch is fully merged
into that repo's main (or whose branch is already gone). It deletes the
merged branch and runs git worktree prune.

A worktree with an unmerged branch is left alone - that is in-flight
work. A worktree with uncommitted changes is skipped with a warning
rather than force-removed.

The same sweep runs automatically at the start of every
'coily dispatch interactive', so worktree sprawl is self-limiting; this
verb is the explicit on-demand version.`,
		Action: func(ctx context.Context, _ *cli.Command) error {
			removed, err := reapDispatchWorktrees(ctx, r)
			if err != nil {
				return fmt.Errorf("dispatch reap: %w", err)
			}
			if len(removed) == 0 {
				fmt.Println("dispatch reap: no merged worktrees to remove")
				return nil
			}
			fmt.Printf("dispatch reap: removed %d merged worktree(s):\n", len(removed))
			for _, p := range removed {
				fmt.Printf("  %s\n", p)
			}
			return nil
		},
	}
}

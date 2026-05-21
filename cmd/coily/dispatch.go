package main

import (
	"os"
	"path/filepath"

	"github.com/coilysiren/cli-guard/dispatch"
	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// dispatch.go is thin wiring. The dispatch subsystem - fire claude against a
// real open issue, headless or interactive, plus the worktree reaper - lives
// in the reusable cli-guard/dispatch package (coilysiren/cli-guard#86). coily
// only supplies the host-specific seams: which org is allowed, where
// checkouts / worktrees / logs live, and how a verb is wrapped for audit.
//
// See coilysiren/coily#270 for the headless/interactive split design and
// coilysiren/coily#310 for the switch to the package.

// allowedOwner is the org coily will dispatch against. Hard-coded rather
// than configurable: this is the security claim, not a knob.
const allowedOwner = "coilysiren"

// localRepoPath returns the expected local checkout for a coilysiren repo.
// Mirrors the workspace shape from AGENTS.md: ~/projects/coilysiren/<repo>.
func localRepoPath(repo string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "projects", allowedOwner, repo), nil
}

// dispatchWorktreeRoot is the parent directory under which each interactive
// dispatch gets its own git worktree: ~/projects/coilysiren/.dispatch-worktrees.
func dispatchWorktreeRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "projects", allowedOwner, ".dispatch-worktrees"), nil
}

// dispatchLogRoot is the parent directory for headless dispatch log files:
// ~/projects/coilysiren/.dispatch-logs. Lives alongside the worktree root,
// outside any repo, so the detached child's stdio lands somewhere stable.
func dispatchLogRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "projects", allowedOwner, ".dispatch-logs"), nil
}

// dispatchCommand builds the dispatch umbrella verb from the cli-guard
// package, wiring coily's runner, audit pipeline, and workspace layout.
// BinaryName "coily" keeps the help text byte-identical to the prior inline
// implementation. Fails the process loudly at startup if the Config is
// invalid - better than a half-wired privileged verb.
func (r *Runner) dispatchCommand() *cli.Command {
	d, err := dispatch.New(dispatch.Config{
		Runner: r.Runner,
		Wrap: func(s verb.Spec) cli.ActionFunc {
			return r.WrapVerb(s, r.Audit)
		},
		AllowedOwner: allowedOwner,
		BinaryName:   "coily",
		RepoPath:     localRepoPath,
		WorktreeRoot: dispatchWorktreeRoot,
		LogRoot:      dispatchLogRoot,
	})
	if err != nil {
		panic("coily: dispatch wiring invalid: " + err.Error())
	}
	return d.Command()
}

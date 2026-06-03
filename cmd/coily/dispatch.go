package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/dispatch"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/verb"
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

// allowedOwner is the org coily's dispatch trust check accepts, and the
// stable anchor for the dispatch infra dirs (.dispatch-worktrees/-logs) and
// the localRepoPath fallback. The checkout-path scan (coily#173) already spans
// every primary org; widening the trust check itself to the primary-org set
// needs cli-guard's dispatch.Config to accept a set (AllowedOwners), tracked
// in coilyco-bridge/coily#173.
const allowedOwner = "coilysiren"

// localRepoPath resolves a repo name to its on-disk checkout. Post org-split
// the same repo may live under any primary org (~/projects/<org>/<repo>), so
// scan the set and return the first checkout that exists rather than assuming
// a single org (coilyco-bridge/coily#173). Falls back to the historical
// ~/projects/coilysiren/<repo> when none is present, preserving the prior
// "local checkout missing at ..." error shape.
func (r *Runner) localRepoPath(repo string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	for _, org := range r.primaryOrgs() {
		p := filepath.Join(home, "projects", org, repo)
		if st, statErr := os.Stat(p); statErr == nil && st.IsDir() {
			return p, nil
		}
	}
	return filepath.Join(home, "projects", allowedOwner, repo), nil
}

// dispatchWorktreeRoot is the parent dir under which each detached dispatch
// (headless, cascade) gets its worktree (coily#145): .dispatch-worktrees.
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
		AllowedOwner:      allowedOwner,
		BinaryName:        "coily",
		RepoPath:          r.localRepoPath,
		WorktreeRoot:      dispatchWorktreeRoot,
		LogRoot:           dispatchLogRoot,
		ForgejoBaseURL:    r.Cfg.Forgejo.BaseURL,
		FetchForgejoIssue: r.fetchForgejoIssue,
	})
	if err != nil {
		panic("coily: dispatch wiring invalid: " + err.Error())
	}
	return d.Command()
}

// fetchForgejoIssue resolves a forgejo issue via the same forgejoAPIDo
// path the `coily ops forgejo issue view` verb uses. Wraps the response
// into the platform-neutral dispatch.Issue shape. A 404 surfaces as an
// error whose message contains "404" so the dispatch package can fall
// back to GitHub for shortform refs.
func (r *Runner) fetchForgejoIssue(ctx context.Context, owner, repo string, number int) (*dispatch.Issue, error) {
	path := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d", owner, repo, number)
	respBody, err := r.forgejoAPIDo(ctx, "dispatch forgejo", http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	var issue dispatch.Issue
	if err := json.Unmarshal(respBody, &issue); err != nil {
		return nil, fmt.Errorf("dispatch forgejo: parse %s: %w", path, err)
	}
	return &issue, nil
}

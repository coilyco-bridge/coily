package main

import (
	"strings"

	"github.com/urfave/cli/v3"
)

// ghActionsGate is the preflight gate wired onto the `coily ops gh`
// passthrough. It rejects any gh invocation that reads GitHub Actions /
// CI status, so the only path left for Actions status is the Playwright
// MCP driving the Actions web UI.
//
// Why playwright-only (coilysiren/coily#305): the GitHub core and GraphQL
// API kept tripping the rate limit on CI-status queries even though
// playwright was the path used most of the time. The API fallback did
// more harm than good, so it stops existing for Actions. The failure mode
// is a hard, legible error - not a silent fallback.
//
// Scope is Actions-only. Non-Actions gh usage (issues, PRs, releases,
// rate-limit checks) passes through untouched: this gate returns nil for
// everything that is not an Actions-status read.
//
// argv is the post-`coily ops gh` slice (the gh subcommand and its args),
// the same shape `passthrough.Command` hands to its action.
func ghActionsGate(argv []string) error {
	if !isGHActionsRead(argv) {
		return nil
	}
	return cli.Exit(ghActionsBlockMessage, 1)
}

// isGHActionsRead reports whether argv is a gh invocation that queries
// GitHub Actions / CI status. The blocked surface (coily#305, "all
// Actions surface"):
//
//   - `gh run ...`      - workflow runs (list / view / watch / ...)
//   - `gh workflow ...` - workflow definitions and dispatch
//   - `gh cache ...`    - the Actions cache
//   - `gh pr checks`    - per-PR CI check status (gh implements over GraphQL)
//   - `gh api <path>`   - any REST path with an `actions` segment
//     (e.g. /repos/O/R/actions/runs)
func isGHActionsRead(argv []string) bool {
	if len(argv) == 0 {
		return false
	}
	switch argv[0] {
	case "run", "workflow", "cache":
		return true
	case "pr":
		return len(argv) > 1 && argv[1] == "checks"
	case "api":
		for _, tok := range argv[1:] {
			if isActionsAPIPath(tok) {
				return true
			}
		}
	}
	return false
}

// isActionsAPIPath reports whether tok is a `gh api` REST path that hits
// the Actions API. A path token has no `=` (that marks a `-f key=value`
// field, whose value could legitimately mention "actions") and does not
// start with `-` (that marks a flag). The Actions API lives under an
// `actions` path segment, so the check is "does any `/`-delimited segment
// equal actions".
func isActionsAPIPath(tok string) bool {
	if tok == "" || strings.HasPrefix(tok, "-") || strings.Contains(tok, "=") {
		return false
	}
	for _, seg := range strings.Split(tok, "/") {
		if seg == "actions" {
			return true
		}
	}
	return false
}

// ghActionsBlockMessage is the hard error printed when the gate fires. It
// names the issue, says why the API path is gone, and points at the only
// supported path (the Playwright MCP against the Actions web UI).
const ghActionsBlockMessage = `coily: GitHub Actions / CI status is playwright-only (coilysiren/coily#305).

The core and GraphQL API path for Actions was removed - ` + "`gh run`" + `,
` + "`gh workflow`" + `, ` + "`gh cache`" + `, ` + "`gh pr checks`" + `, and ` + "`gh api`" + ` paths under
/actions/ no longer pass through coily. The GitHub API rate limit kept
blocking these even when playwright was the path used most of the time.

Check CI status with the Playwright MCP against the Actions web UI:
  https://github.com/<owner>/<repo>/actions

Non-Actions GitHub data (issues, PRs, releases, rate-limit checks) still
works through ` + "`coily ops gh`" + `.`

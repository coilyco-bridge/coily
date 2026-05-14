package main

import (
	"os"
	"strings"

	"github.com/coilysiren/cli-guard/lockdown"
	"github.com/coilysiren/coily/pkg/config"
	"github.com/coilysiren/coily/pkg/decision"
)

// coilyAllowedPaths is the closed set of filesystem paths a `coily`
// invocation is permitted to resolve to. Anything else (including
// ~/go/bin/coily, /tmp/foo/coily, ./bin/coily) is rejected by the
// generated PreToolUse hook. The list covers homebrew on Apple
// Silicon, homebrew on Intel macOS, and Linuxbrew on Linux
// (kai-server). The hook follows the bin/coily symlink Brew lays
// down rather than the Cellar realpath, because `command -v coily`
// returns the symlink path.
var coilyAllowedPaths = []string{
	"/opt/homebrew/bin/coily",              // Apple Silicon homebrew
	"/usr/local/bin/coily",                 // Intel macOS homebrew
	"/home/linuxbrew/.linuxbrew/bin/coily", // Linuxbrew default prefix
}

// wrapperRecovery maps a denied bare-binary leading-token to the coily
// wrapper that should be used instead. When a deny rule fires for a
// binary with a wrapper, the generated deny message names the wrapper
// as the recovery path per AGENTS.md "opaque errors are design smells -
// recovery messages should name the command or skill Kai can dictate
// next." Issue #61.
//
// Source of truth for cross-repo recovery hints (issue #122). Every
// coily wrapper that shadows a denied bare binary lands here so
// `coily lockdown` renders the hint into each repo's lockdown-deny.sh,
// replacing the prior per-repo hand-sync.
var wrapperRecovery = map[string]string{
	// ops pass-throughs.
	"gh":        "coily ops gh",
	"aws":       "coily ops aws",
	"kubectl":   "coily ops kubectl",
	"docker":    "coily docker",
	"tailscale": "coily tailscale",

	// ssh family. Free-form remote exec was removed; named verbs live
	// under `coily ssh` (copy, systemctl, journalctl, kubectl, git,
	// deploy, fs).
	"ssh": "coily ssh",
	"scp": "coily ssh copy",

	// Package managers. All wrapped under `coily pkg <pkgmgr>`.
	"npm":    "coily pkg npm",
	"pnpm":   "coily pkg pnpm",
	"yarn":   "coily pkg yarn",
	"bun":    "coily pkg bun",
	"uv":     "coily pkg uv",
	"pip":    "coily pkg pip",
	"pipx":   "coily pkg pipx",
	"poetry": "coily pkg poetry",
	"cargo":  "coily pkg cargo",
	"gem":    "coily pkg gem",
	"bundle": "coily pkg bundle",
	"brew":   "coily brew",

	// Build runners. The audited replacement is a named verb in
	// .coily/coily.yaml dispatched via `coily exec <verb>`.
	"make":   "coily exec <verb>",
	"just":   "coily exec <verb>",
	"task":   "coily exec <verb>",
	"invoke": "coily exec <verb>",
}

// coilyLockdownDriver returns the ClaudeCode-shaped lockdown driver
// for the coily binary. Called wherever the lockdown writer needs to
// know which binary it is gating and which wrappers to mention in
// deny recovery hints.
func coilyLockdownDriver() *lockdown.Driver {
	drv := lockdown.ClaudeCode("coily", coilyAllowedPaths, wrapperRecovery)
	// Phase 4 plumbing for #150: attach the resolved per-session
	// Coordinate to the driver so phase 5's BuildSettings consumer can
	// branch on it. Best-effort: a malformed override file leaves the
	// Coordinate unset (nil) so today's settings.json output path is
	// unaffected.
	sid := strings.TrimSpace(os.Getenv(sessionEnvVar))
	if sid != "" {
		if active, err := readSessionProfileName(sid); err == nil {
			if c, err := decision.CoordinatePtr(active); err == nil {
				drv.Coordinate = c
			}
		}
	}
	return drv
}

// readSessionProfileName reads the session sentinel for sid via the
// same code path as `coily session show`, returning the empty string
// if the sentinel is absent. Errors propagate only for malformed
// sentinels.
func readSessionProfileName(sid string) (string, error) {
	path, err := config.SessionProfilePath(sid)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(path) //nolint:gosec // path derived from $CLAUDE_CODE_SESSION_ID via config.SessionProfilePath
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

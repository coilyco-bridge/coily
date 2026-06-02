package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/scope"
	"github.com/urfave/cli/v3"
)

// uvPreflightGate is the preflight gate wired onto the `coily pkg uv`
// passthrough. It rejects `uv run` / `uv tool run` invocations whose
// script target is a free-form path that escapes the repo root or sits
// in a writable scratch tier (/tmp and friends).
//
// Why (coily#10, "coily pkg uv run accepts an arbitrary /tmp script
// path"): `uv run PATH` is a general-purpose Python interpreter. Bare
// `uv` is denied at the lockdown layer so every invocation is forced
// through `coily pkg uv`, but the passthrough did not look at the path,
// so `coily pkg uv run /tmp/x.py` ran any script the caller could write
// to disk - arbitrary code execution laundered through the gate.
//
// This is the same hole as direct `/tmp/foo` execution (coily#9), which
// cli-guard's engine-level scratch-exec deny (cli-guard#87) already
// blocks in the PreToolUse hook. That deny classifies the *leading
// token* of a Bash segment; when the segment leads with `coily`, the
// `/tmp/x.py` argument is invisible to it. This gate is the matching
// check on the coily side so the two stay consistent: scratch-tier and
// repo-escaping script paths are denied no matter which route reaches
// them.
//
// Scope is the script *path*, per the issue. Project-rooted relative
// paths (`uv run script.py`, `uv run pytest tests/foo.py`) and bare tool
// names (`uv run pytest`, `uv run ruff check`) pass untouched, as do the
// non-`run` verbs coily itself relies on (`uv tool install`, `uv pip`,
// `uv sync`). Running an interpreter through uv with no path argument
// (`uv run python -c ...`) is a narrower residual the engine interpreter
// deny does not reach through uv; tracked separately.
//
// argv is the post-`coily pkg uv` slice, the same shape
// passthrough.Command hands to its action.
func uvPreflightGate(argv []string) error {
	if msg, blocked := uvRunPathViolation(argv, scope.CWD(), ""); blocked {
		return cli.Exit(msg, 1)
	}
	return nil
}

// uvRunPathViolation is the testable core. It returns a block message
// and true when argv is a `uv run` / `uv tool run` invocation carrying a
// script-path argument that escapes repoRoot or lands in a scratch tier.
//
// repoRoot may be passed in (tests); when empty it is resolved from cwd
// via scope.RepoRoot. An empty repoRoot after resolution means "not
// inside a git repo", in which case any absolute path argument is
// treated as escaping - a `uv run` against an absolute path with no
// repo to anchor it is exactly the laundering shape the gate exists for.
func uvRunPathViolation(argv []string, cwd, repoRoot string) (string, bool) {
	runArgs, ok := uvRunTargets(argv)
	if !ok {
		return "", false
	}
	if repoRoot == "" {
		repoRoot = scope.RepoRoot(cwd)
	}
	for _, arg := range runArgs {
		if path, bad := uvEscapingPath(arg, cwd, repoRoot); bad {
			return uvRunBlockMessage(path), true
		}
	}
	return "", false
}

// uvRunTargets returns the argument slice to scan for a script path when
// argv is a script-running uv invocation, and false otherwise. It peels
// the `run` / `tool run` prefix; everything after is scanned (uv's own
// flags are skipped by uvEscapingPath, which only flags path-shaped
// tokens).
func uvRunTargets(argv []string) ([]string, bool) {
	switch {
	case len(argv) >= 1 && argv[0] == "run":
		return argv[1:], true
	case len(argv) >= 2 && argv[0] == "tool" && argv[1] == "run":
		return argv[2:], true
	}
	return nil, false
}

// uvEscapingPath reports whether arg denotes a filesystem path that
// escapes repoRoot or lands in a scratch tier. Returns the cleaned
// absolute path and true on a violation. Non-path tokens (flags, bare
// tool/module names) return false.
func uvEscapingPath(arg, cwd, repoRoot string) (string, bool) {
	if arg == "" || strings.HasPrefix(arg, "-") {
		return "", false
	}
	if !looksLikePath(arg) {
		return "", false
	}
	abs := arg
	if !filepath.IsAbs(abs) {
		if cwd == "" {
			// Cannot resolve a relative path with no cwd anchor. The
			// engine interpreter deny is the backstop for bare forms.
			return "", false
		}
		abs = filepath.Join(cwd, abs)
	}
	abs = filepath.Clean(abs)
	if underScratchRoot(abs) {
		return abs, true
	}
	if repoRoot != "" {
		if abs == repoRoot || strings.HasPrefix(abs, repoRoot+string(filepath.Separator)) {
			return "", false
		}
		return abs, true
	}
	// No repo root to anchor against: an absolute path cannot be
	// project-rooted, so treat it as escaping.
	if filepath.IsAbs(arg) {
		return abs, true
	}
	return "", false
}

// looksLikePath reports whether tok is shaped like a filesystem path
// rather than a bare tool or module name. A token counts as a path when
// it is absolute, contains a separator, starts with `.` or `~`, or
// carries a script-file suffix. Bare names (`pytest`, `ruff`, `mod`)
// return false so they pass the gate.
func looksLikePath(tok string) bool {
	if tok == "" {
		return false
	}
	if filepath.IsAbs(tok) || strings.ContainsRune(tok, '/') {
		return true
	}
	if strings.HasPrefix(tok, ".") || strings.HasPrefix(tok, "~") {
		return true
	}
	return hasScriptSuffix(tok)
}

// scriptSuffixes are the file extensions that mark a bare token as a
// script path even without a separator (`uv run foo.py`). Kept to the
// interpreted-language suffixes uv-adjacent workflows actually run.
var scriptSuffixes = []string{
	".py", ".pyz", ".pyc", ".sh", ".bash", ".zsh",
	".pl", ".rb", ".js", ".mjs", ".cjs", ".ts",
}

func hasScriptSuffix(tok string) bool {
	lower := strings.ToLower(tok)
	for _, s := range scriptSuffixes {
		if strings.HasSuffix(lower, s) {
			return true
		}
	}
	return false
}

// uvScratchRoots mirrors cli-guard/hook's scratchExecRoots: the
// world-writable scratch prefixes an agent stages code into. Kept in
// sync by intent; the two guard the same surface from different layers
// (the hook on direct exec, this gate on exec laundered through uv).
var uvScratchRoots = []string{
	"/tmp", "/var/tmp", "/dev/shm",
	"/private/tmp", "/private/var/tmp",
}

func underScratchRoot(abs string) bool {
	for _, root := range uvScratchRoots {
		if abs == root || strings.HasPrefix(abs, root+"/") {
			return true
		}
	}
	return false
}

func uvRunBlockMessage(path string) string {
	return fmt.Sprintf(`coily: `+"`uv run`"+` against the free-form path %q is blocked.

`+"`uv run PATH`"+` is a general-purpose Python interpreter, so running a
script that lives in a scratch tier (/tmp, /var/tmp, /dev/shm) or outside
the repo root is arbitrary code execution laundered through the gate
- the same hole cli-guard's scratch-exec deny closes for direct `+"`/tmp/foo`"+`
execution (coily#9, coily#10).

Allowed: project-rooted script paths (`+"`uv run ./script.py`"+`,
`+"`uv run pytest tests/foo.py`"+`) and bare tool/module names
(`+"`uv run pytest`"+`, `+"`uv run ruff check`"+`). Move the script into the
repo, or run it through its project entry point.`, path)
}

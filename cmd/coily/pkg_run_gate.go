package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/pkg/scope"
	"github.com/urfave/cli/v3"
)

// Preflight gate for the `coily pkg <tool>` passthroughs whose run/exec
// verbs are general-purpose interpreters. Rejects an invocation whose
// script-path argument sits in a writable scratch tier (/tmp and friends)
// or escapes the repo root.
//
// Why (coily#9, coily#10): `uv run PATH` / `bun PATH` / `poetry run python
// PATH` / `bundle exec ruby PATH` and friends will run any script the
// caller can write to disk - arbitrary code execution laundered through
// the gate. cli-guard's engine-level scratch-exec deny (cli-guard#87)
// blocks direct `/tmp/foo` execution, but it classifies the *leading
// token* of a Bash segment; when the segment leads with `coily`, the
// `/tmp/x` argument buried in `coily pkg <tool> run /tmp/x` is invisible
// to it. This gate is the matching check on the coily side so scratch-tier
// and repo-escaping script paths are denied no matter which route reaches
// them. coily#10 named `uv`; the same hole exists for every pkg passthrough
// with a run/exec verb, so the gate is registry-driven.
//
// Scope is the script *path*, per the issue: project-rooted relative paths
// (`uv run script.py`, `pnpm exec eslint src/`) and bare tool/module names
// (`uv run pytest`, `bun run build`) pass untouched, as do the non-run
// verbs coily itself relies on (`uv tool install`, `npm install`,
// `nix build`). `pip` / `gem` (install-time setup-code execution, a
// different threat class) and `cargo` (run targets are repo-defined) are
// intentionally not registered.

// pkgRunRule is one script-running invocation shape: a leading subcommand
// prefix (`run`, `exec`, `tool run`, ...) whose trailing args carry the
// script/command target.
type pkgRunRule struct {
	prefix []string
}

// pkgRunSpec is a package manager's set of script-running shapes. scanAll
// covers tools that exec a bare file with no subcommand at all
// (`bun /tmp/x.ts`), where every argument is a candidate target.
type pkgRunSpec struct {
	rules   []pkgRunRule
	scanAll bool
}

// pkgRunSpecs is the registry of which `coily pkg <bin>` verbs run a
// script/command. Wired onto the matching ptPkg entries via pkgRunGate.
// Keep in sync with passthroughs.go: a bin here without the gate wired is
// dead config, and a gate wired without an entry here is a no-op.
var pkgRunSpecs = map[string]pkgRunSpec{
	"uv":     {rules: []pkgRunRule{{prefix: []string{"run"}}, {prefix: []string{"tool", "run"}}}},
	"bun":    {scanAll: true},
	"pipx":   {rules: []pkgRunRule{{prefix: []string{"run"}}}},
	"poetry": {rules: []pkgRunRule{{prefix: []string{"run"}}}},
	"bundle": {rules: []pkgRunRule{{prefix: []string{"exec"}}}},
	"npm":    {rules: []pkgRunRule{{prefix: []string{"exec"}}, {prefix: []string{"x"}}}},
	"pnpm":   {rules: []pkgRunRule{{prefix: []string{"exec"}}, {prefix: []string{"dlx"}}}},
	"yarn":   {rules: []pkgRunRule{{prefix: []string{"exec"}}, {prefix: []string{"dlx"}}}},
	"nix":    {rules: []pkgRunRule{{prefix: []string{"run"}}, {prefix: []string{"eval"}}, {prefix: []string{"shell"}}, {prefix: []string{"develop"}}}},
}

// pkgRunGate returns the preflight gate for a `coily pkg <bin>` passthrough.
// Returns nil for a bin with no run/exec surface, so callers can wire it
// unconditionally.
func pkgRunGate(bin string) func(argv []string) error {
	if _, ok := pkgRunSpecs[bin]; !ok {
		return nil
	}
	return func(argv []string) error {
		if msg, blocked := pkgRunPathViolation(bin, argv, scope.CWD(), ""); blocked {
			return cli.Exit(msg, 1)
		}
		return nil
	}
}

// pkgRunPathViolation is the testable core. It returns a block message and
// true when argv is a script-running invocation of bin carrying a
// script-path argument that escapes repoRoot or lands in a scratch tier.
//
// repoRoot may be passed in (tests); when empty it is resolved from cwd via
// scope.RepoRoot. An empty repoRoot after resolution means "not inside a
// git repo", in which case any absolute path argument is treated as
// escaping - a run against an absolute path with no repo to anchor it is
// exactly the laundering shape the gate exists for.
func pkgRunPathViolation(bin string, argv []string, cwd, repoRoot string) (string, bool) {
	spec, ok := pkgRunSpecs[bin]
	if !ok {
		return "", false
	}
	scanArgs, label, matched := pkgRunScan(spec, bin, argv)
	if !matched {
		return "", false
	}
	if repoRoot == "" {
		repoRoot = scope.RepoRoot(cwd)
	}
	for _, arg := range scanArgs {
		if path, bad := pkgEscapingPath(arg, cwd, repoRoot); bad {
			return pkgRunBlockMessage(bin, label, path), true
		}
	}
	return "", false
}

// pkgRunScan returns the args to scan for a script path, a human label for
// the matched invocation form, and whether argv is a run/exec invocation.
func pkgRunScan(spec pkgRunSpec, bin string, argv []string) (scan []string, label string, matched bool) {
	for _, rule := range spec.rules {
		if argvHasPrefix(argv, rule.prefix) {
			return argv[len(rule.prefix):], strings.Join(rule.prefix, " "), true
		}
	}
	if spec.scanAll && len(argv) > 0 {
		// Bare-file exec (`bun /tmp/x.ts`): no recognized subcommand, so
		// every argument is a candidate target. Non-path tokens
		// (subcommands, flags, package names) are skipped by pkgEscapingPath.
		return argv, bin, true
	}
	return nil, "", false
}

func argvHasPrefix(argv, prefix []string) bool {
	if len(prefix) == 0 || len(argv) < len(prefix) {
		return false
	}
	for i, p := range prefix {
		if argv[i] != p {
			return false
		}
	}
	return true
}

// pkgEscapingPath reports whether arg denotes a filesystem path that
// escapes repoRoot or lands in a scratch tier. Returns the cleaned
// absolute path and true on a violation. Non-path tokens (flags, bare
// tool/module names) return false.
func pkgEscapingPath(arg, cwd, repoRoot string) (string, bool) {
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

// looksLikePath reports whether tok is shaped like a filesystem path rather
// than a bare tool or module name. A token counts as a path when it is
// absolute, contains a separator, starts with `.` or `~`, or carries a
// script-file suffix. Bare names (`pytest`, `ruff`, `eslint`) return false.
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

// scriptSuffixes are the file extensions that mark a bare token as a script
// path even without a separator (`uv run foo.py`). Kept to the interpreted-
// language suffixes the registered tools actually run.
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

// pkgScratchRoots mirrors cli-guard/hook's scratchExecRoots: the world-
// writable scratch prefixes an agent stages code into. Kept in sync by
// intent; the two guard the same surface from different layers (the hook on
// direct exec, this gate on exec laundered through a pkg passthrough).
var pkgScratchRoots = []string{
	"/tmp", "/var/tmp", "/dev/shm",
	"/private/tmp", "/private/var/tmp",
}

func underScratchRoot(abs string) bool {
	for _, root := range pkgScratchRoots {
		if abs == root || strings.HasPrefix(abs, root+"/") {
			return true
		}
	}
	return false
}

func pkgRunBlockMessage(bin, label, path string) string {
	inv := bin
	if label != "" && label != bin {
		inv = bin + " " + label
	}
	return fmt.Sprintf(`coily: `+"`%s`"+` against the free-form path %q is blocked.

`+"`%s`"+` runs a script as a general-purpose interpreter, so executing one
that lives in a scratch tier (/tmp, /var/tmp, /dev/shm) or outside the repo
root is arbitrary code execution laundered through the gate - the same hole
cli-guard's scratch-exec deny closes for direct `+"`/tmp/foo`"+` execution
(coily#9, coily#10).

Allowed: project-rooted script paths (`+"`%s tests/foo.py`"+`) and bare
tool/module names (`+"`%s pytest`"+`). Move the script into the repo, or run
it through its project entry point.`, inv, path, inv, inv, inv)
}

package main

import (
	"os"
	"strings"
)

// normalizeGHJQInline spills an inline `ops gh ... --jq <expr>` (or -q, --jq=,
// -q=) onto coily's existing gate-safe `--jq-file <path>` rail before the
// shell-metacharacter gate runs. jq programs are full of gate-tripping
// characters (`|`, `[`, `(`, `>`), yet gh execs them directly with no shell,
// so they are inert: the gate's 96.6% rejection of --jq (coilyco-bridge/coily#30)
// is a pure false positive. Rather than exempt the value in-argv (which would
// weaken the gate's whole-argv invariant), route it off-argv into a temp file.
// The gate then sees only a clean filesystem path, and rewriteJQFile expands
// the file back to `--jq <expr>` after the gate, before gh runs. Same
// mechanism as the hand-written --jq-file shorthand (coily#165), just
// automatic, so inline --jq "just works" without sacrificing the boundary.
//
// Scope is deliberately narrow: only the `ops gh` command path, only the jq
// flags. Returns the (possibly rewritten) argv and a cleanup func that removes
// any temp files; cleanup is always safe to call (no-op when nothing spilled).
func normalizeGHJQInline(argv []string) ([]string, func()) {
	start := opsGHArgStart(argv)
	if start < 0 {
		return argv, func() {}
	}
	var tmps []string
	cleanup := func() {
		for _, p := range tmps {
			_ = os.Remove(p)
		}
	}
	out := make([]string, 0, len(argv))
	out = append(out, argv[:start]...)
	for i := start; i < len(argv); i++ {
		tok := argv[i]
		name, inlineVal, hasInline := splitGHJQFlag(tok)
		if name == "" {
			out = append(out, tok)
			continue
		}
		var expr string
		switch {
		case hasInline:
			expr = inlineVal
		case i+1 < len(argv):
			expr = argv[i+1]
			i++
		default:
			// Dangling flag with no value: leave as-is, let gh complain.
			out = append(out, tok)
			continue
		}
		path, err := spillJQExpr(expr)
		if err != nil {
			// Fall back to the original inline form. The gate may still
			// reject it, but coily does not silently swallow the intent.
			out = append(out, name, expr)
			continue
		}
		tmps = append(tmps, path)
		out = append(out, "--jq-file", path)
	}
	return out, cleanup
}

// opsGHArgStart returns the index in argv just past the `ops gh` command path
// (the first gh-subcommand/flag token), or -1 if argv is not an `ops gh`
// invocation. Tolerates global flags before `ops`.
func opsGHArgStart(argv []string) int {
	for i := 0; i+1 < len(argv); i++ {
		if argv[i] == "ops" && argv[i+1] == "gh" {
			return i + 2
		}
	}
	return -1
}

// splitGHJQFlag classifies a token as a gh jq flag. Returns the canonical flag
// name ("--jq" or "-q"), the inline value, and whether one was present (the
// `--jq=expr` / `-q=expr` forms). name is "" when tok is not a jq flag. Note
// `--jq-file` is intentionally not matched: it is already the gate-safe rail.
func splitGHJQFlag(tok string) (name, val string, hasInline bool) {
	switch {
	case tok == "--jq":
		return "--jq", "", false
	case tok == "-q":
		return "-q", "", false
	case strings.HasPrefix(tok, "--jq="):
		return "--jq", strings.TrimPrefix(tok, "--jq="), true
	case strings.HasPrefix(tok, "-q="):
		return "-q", strings.TrimPrefix(tok, "-q="), true
	default:
		return "", "", false
	}
}

// spillJQExpr writes a jq program to a short-lived temp file and returns its
// path. The file is consumed by rewriteJQFile after the metachar gate and
// removed by the cleanup func normalizeGHJQInline returns.
func spillJQExpr(expr string) (string, error) {
	f, err := os.CreateTemp("", "coily-gh-jq-*.jq")
	if err != nil {
		return "", err
	}
	if _, err := f.WriteString(expr); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

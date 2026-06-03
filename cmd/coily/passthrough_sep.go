package main

// stripPassthroughSeparator removes the single `--` token that separates a
// coily passthrough command from the args it forwards, e.g.
// `coily ops aws -- sts get-caller-identity --query X --output text`.
//
// urfave/cli does not preserve argv order across that `--` for a
// SkipFlagParsing passthrough: the trailing flags get reordered ahead of the
// positionals, so the wrapped tool sees mangled argv and rejects it
// ("aws: Unknown options: --query, --output, ..." / "unknown command api for
// gh"). The no-separator form already forwards verbatim and works, so dropping
// the redundant separator makes `coily ops <tool> -- <args>` behave like the
// documented byte-for-byte-forward contract. coilyco-bridge/coily#37.
//
// Scope is deliberately narrow: only a `--` immediately following a registered
// passthrough bin in command position (an ops/pkg group bin, or a top-level
// passthrough bin). A `--` anywhere else (e.g. `ops aws s3 cp -- file`, where
// it is aws's own separator) is left untouched.
func stripPassthroughSeparator(argv []string) []string {
	for i := 1; i+1 < len(argv); i++ {
		if argv[i+1] != "--" {
			continue
		}
		if !isPassthroughCommandPos(argv, i) {
			continue
		}
		out := make([]string, 0, len(argv)-1)
		out = append(out, argv[:i+1]...)
		out = append(out, argv[i+2:]...)
		return out
	}
	return argv
}

// isPassthroughCommandPos reports whether argv[i] is a passthrough binary in
// command position: an ops-group bin preceded by "ops", a pkg-group bin
// preceded by "pkg", or a top-level passthrough bin anywhere.
func isPassthroughCommandPos(argv []string, i int) bool {
	bin := argv[i]
	prev := argv[i-1]
	switch {
	case prev == "ops" && ptBinSet(ptOps)[bin]:
		return true
	case prev == "pkg" && ptBinSet(ptPkg)[bin]:
		return true
	case ptBinSet(ptTopLevel)[bin]:
		return true
	default:
		return false
	}
}

// ptBinSet projects a passthrough registry slice to a bin-name set.
func ptBinSet(entries []ptEntry) map[string]bool {
	m := make(map[string]bool, len(entries))
	for _, e := range entries {
		m[e.Bin] = true
	}
	return m
}

package main

import (
	"strings"
	"time"
)

// ghReadCacheClassifier inspects the post-rewrite argv that the gh
// pass-through is about to hand to gh. Recognizes the shapes the REST
// rewriter emits for reads (`api <path>` with no body and no method
// override) plus bare `gh api <path>` that an operator types directly.
// Returns the gh-api-style path and true on a recognized read; ("", false)
// to decline.
//
// Anything that smells like a write declines: `-X POST/PATCH/DELETE`,
// `-f` / `-F` / `--field` / `--raw-field` body flags, `--input` body
// source, or an unrecognized method via `--method`. The classifier is
// deliberately strict; on doubt it declines and the pass-through runs
// the subprocess as today.
//
// The returned maxAge is the per-invocation cap stashed by
// rewriteGHForRESTAndJQFile via --max-age / COILY_GH_MAX_AGE
// (coilysiren/coily#267). -1 (no cap) when neither was set, so the
// pre-flag behavior is preserved.
//
// Designed to feed cli-guard's passthrough.WithReadCache. See
// coilysiren/coily#266 and cli-guard#76.
func ghReadCacheClassifier(argv []string) (string, time.Duration, bool) {
	if len(argv) < 2 || argv[0] != "api" {
		return "", 0, false
	}
	path := ""
	for i := 1; i < len(argv); i++ {
		tok := argv[i]
		if isBodyFlag(tok) {
			return "", 0, false
		}
		if method, consumed, ok := readMethodFlag(argv, i); ok {
			if strings.ToUpper(method) != "GET" {
				return "", 0, false
			}
			i += consumed
			continue
		}
		if strings.HasPrefix(tok, "-") {
			// Unknown flag - decline rather than guess.
			return "", 0, false
		}
		// First non-flag positional is the path. Subsequent positionals
		// are unexpected for the `api <path>` shape; decline.
		if path != "" {
			return "", 0, false
		}
		path = tok
	}
	if path == "" {
		return "", 0, false
	}
	return path, loadGHMaxAge(), true
}

// isBodyFlag reports whether tok is one of gh api's body-bearing
// flags (any form). Presence forces a write classification.
func isBodyFlag(tok string) bool {
	switch tok {
	case "-f", "-F", "--field", "--raw-field", "--input":
		return true
	}
	for _, p := range []string{"-f=", "-F=", "--field=", "--raw-field=", "--input="} {
		if strings.HasPrefix(tok, p) {
			return true
		}
	}
	return false
}

// readMethodFlag parses the method override at argv[i]. Returns the
// raw method string, the number of additional argv tokens consumed
// (0 for --method=GET, 1 for --method GET), and ok=true if argv[i]
// is a method flag at all.
func readMethodFlag(argv []string, i int) (method string, consumed int, ok bool) {
	tok := argv[i]
	if tok == "-X" || tok == "--method" {
		if i+1 >= len(argv) {
			// Missing value - return ok=true with empty method so the
			// caller declines.
			return "", 0, true
		}
		return argv[i+1], 1, true
	}
	if strings.HasPrefix(tok, "-X=") || strings.HasPrefix(tok, "--method=") {
		eq := strings.IndexByte(tok, '=')
		return tok[eq+1:], 0, true
	}
	return "", 0, false
}

package main

import (
	"os"
	"strings"
)

// rewriteMcporterArgsFile expands the coily-side `--args-file <path>` (or
// `--args-file=<path>`) shorthand into mcporter's native `--args <content>`.
// Pulled out as its own pass so the metachar gate runs on the raw argv
// (where the value is a clean filesystem path) and the JSON payload only
// enters argv after the gate. Same shape as rewriteJQFile for gh.
//
// mcporter's `--args <json>` expects a JSON object payload, which always
// contains `{` and `}`; calling `coily ops mcporter call <tool> --args
// '{"k":"v"}'` therefore trips the shell-metachar policy. Operators write
// the payload to a file, then pass `--args-file /tmp/payload.json` and
// the rewriter substitutes the contents in before the call lands on
// mcporter.
//
// Falls through unchanged on file-read errors so mcporter reports the
// missing-path error itself instead of coily synthesizing one.
// Falls through if the flag has no value after it (malformed argv).
func rewriteMcporterArgsFile(argv []string) []string {
	out := make([]string, 0, len(argv))
	for i := 0; i < len(argv); i++ {
		tok := argv[i]
		switch {
		case tok == "--args-file":
			rest := argv[i+1:]
			if len(rest) == 0 {
				return argv
			}
			// #nosec G304 G602 -- path is operator-supplied via argv;
			// metachar gate already validated it. Bounds checked above.
			content, err := os.ReadFile(rest[0])
			if err != nil {
				return argv
			}
			out = append(out, "--args", strings.TrimRight(string(content), "\n"))
			i++
		case strings.HasPrefix(tok, "--args-file="):
			path := strings.TrimPrefix(tok, "--args-file=")
			// #nosec G304 -- see above.
			content, err := os.ReadFile(path)
			if err != nil {
				return argv
			}
			out = append(out, "--args", strings.TrimRight(string(content), "\n"))
		default:
			out = append(out, tok)
		}
	}
	return out
}

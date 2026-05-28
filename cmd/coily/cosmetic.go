package main

import (
	"fmt"
	"strings"

	"github.com/coilysiren/cli-guard/policy"
)

// cosmeticSanitize rewrites shell metacharacters in a cosmetic string
// argument - an issue/PR/milestone title, a label/release name - into
// harmless visual equivalents so the argv policy gate (which rejects any
// argument containing a byte in policy.ShellMeta) does not bounce a value
// that is shipped as JSON data over the forgejo HTTP API and never
// interpolated into a shell.
//
// Only call this for (verb, param) pairs that are provably data-not-shell;
// the enumerated allowlist lives at the call sites (coilysiren/coily#129).
// Shell-bound argv (tailscale ssh, aws ssm send-command) keeps the hard
// rejection - never route those through here.
//
// Substitution rules:
//
//	;            -> " - "
//	|            -> "/"
//	&            -> " and "
//	everything else in policy.ShellMeta (` $ < > ( ) { } \ \n \r \t) -> " "
//
// Runs of whitespace introduced by the rewrite collapse to a single space and
// the result is trimmed. Returns the sanitized value and whether it differs
// from the input. When it differs, the result is guaranteed free of every
// byte in policy.ShellMeta, so it passes the gate unchanged.
func cosmeticSanitize(value string) (string, bool) {
	if !strings.ContainsAny(value, policy.ShellMeta) {
		return value, false
	}
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		switch r {
		case ';':
			b.WriteString(" - ")
		case '|':
			b.WriteByte('/')
		case '&':
			b.WriteString(" and ")
		case '`', '$', '<', '>', '(', ')', '{', '}', '\\', '\n', '\r', '\t':
			b.WriteByte(' ')
		default:
			b.WriteRune(r)
		}
	}
	// strings.Fields splits on any run of whitespace and drops empties, so a
	// single Join collapses the spaces the rewrite introduced (and any the
	// rewrite left adjacent) back to one.
	out := strings.Join(strings.Fields(b.String()), " ")
	return out, true
}

// cosmeticSanitizeValue returns just the sanitized form of value, dropping
// the changed flag. Convenience for ArgsFunc map literals, which only need
// the gate-safe value; the operator notice fires from the Action via
// cosmeticArg.
func cosmeticSanitizeValue(value string) string {
	out, _ := cosmeticSanitize(value)
	return out
}

// cosmeticArg sanitizes a cosmetic flag value for use inside a verb Action,
// emitting an operator notice on stderr when a substitution occurred. The
// pre-sanitization input is preserved verbatim in the audit row's Argv
// (verb.Wrap captures it from os.Args before any validation), so forensics
// retain the original and the deterministic rules above recover the exact
// substitution. verbName/param are the dotted verb path and flag name, used
// only for the notice. Pairs ArgsFunc (which feeds the gate the same cleaned
// value via cosmeticSanitize) with the Action (which uses the return here).
func (r *Runner) cosmeticArg(verbName, param, raw string) string {
	out, changed := cosmeticSanitize(raw)
	if changed {
		_, _ = fmt.Fprintf(r.Runner.Stderr,
			"coily: %s: auto-sanitized cosmetic %s metacharacter (data-not-shell): %q -> %q\n",
			verbName, param, raw, out)
	}
	return out
}

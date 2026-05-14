package main

// denialReasons maps an error envelope `kind` (the lowercase token from
// exitcode.CodedError.Kind()) to a one-line explanation of the threat or
// invariant the gate is preserving. The envelope renderer pairs this
// with the existing `hint:` (recovery) field so a denial carries both
// "what to do" and "why this gate exists."
//
// Reason text is intentionally short (one line), invariant-shaped (no
// repro details, no audit row ids), and reusable across instances. If a
// reason is missing the envelope just omits the `reason:` line, so
// adding entries here is purely additive.
//
// Source of truth for the why-line per coilysiren/coily#126.
var denialReasons = map[string]string{
	"policy_denied": "policy gate caught a shell metacharacter in argv; the gate exists because some downstream verbs (tailscale ssh, aws ssm send-command) ship argv into a remote shell, and a per-tool toggle can't tell those verbs apart from the safe ones. " +
		"Workarounds for the high-volume cases: " +
		"--body 'markdown' -> --body-file /tmp/body.md (gh native); " +
		"--change-batch '{...}' -> --change-batch file:///tmp/batch.json (aws native); " +
		"--jq '...' -> external pipe (coily ops gh api X | jq '...') or coily-side --jq-file /tmp/q.jq",
	"repo_verb_dirty":      "audit rows must bind to a clean commit so the run can be reconstructed from git history",
	"repo_no_config":       "repo verbs require a .coily/coily.yaml so the verb surface is declarative and reviewable",
	"scope_unresolved":     "every audit row binds to a real commit scope; --commit-scope is the explicit hand-off when cwd cannot resolve one",
	"exec_prompt_no_stdin": "the choice prompt for ambiguous repo names needs an interactive stdin; pipe the 1-indexed pick instead",
	"exec_prompt_read":     "the choice prompt for ambiguous repo names needs an interactive stdin; pipe the 1-indexed pick instead",
	"exec_prompt_empty":    "the choice prompt requires a 1-indexed integer; an empty pick is ambiguous",
	"exec_prompt_invalid":  "the choice prompt requires a 1-indexed integer in range",
}

// reasonFor returns the registered reason for a kind, or "" when none
// exists. Callers should treat the empty string as "omit the field."
func reasonFor(kind string) string {
	return denialReasons[kind]
}

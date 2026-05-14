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
	"policy_denied":        "policy gate caught an argv shape that has no audited path; the gate exists so coily owns the bounds, not the wrapped tool",
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

package main

// denialReasons is the fallback why-line map for error kinds emitted by
// cli-guard sites that have not yet migrated to exitcode.CodedError.
// WithReason() construction. The envelope renderer prefers the
// Reasoner interface (errors.As) and only consults this map when the
// returned reason is empty.
//
// Coily-side construction sites have all migrated to WithReason() at
// the call point. The remaining entries cover cli-guard internals:
// policy_denied (verb.go shell-meta gate) and scope_unresolved
// (verb.go commit-scope binding). Drop entries here once the
// corresponding cli-guard site adds WithReason() upstream
// (tracked at coilysiren/cli-guard for the per-kind migration).
var denialReasons = map[string]string{
	"policy_denied": "policy gate caught a shell metacharacter in argv; the gate exists because some downstream verbs (tailscale ssh, aws ssm send-command) ship argv into a remote shell, and a per-tool toggle can't tell those verbs apart from the safe ones. " +
		"Workarounds for the high-volume cases: " +
		"--body 'markdown' -> --body-file /tmp/body.md (gh native); " +
		"--change-batch '{...}' -> --change-batch file:///tmp/batch.json (aws native); " +
		"--jq '...' -> external pipe (coily ops gh api X | jq '...') or coily-side --jq-file /tmp/q.jq; " +
		"mcporter --args '{...}' -> coily-side --args-file /tmp/args.json; " +
		"for user-defined `coily exec <verb>` invocations, opt the verb into `allow_metacharacters: true` in the declaring .coily/coily.yaml (audit row stamps policy_skipped so forensics still see it)",
	"scope_unresolved": "every audit row binds to a real commit scope; --commit-scope is the explicit hand-off when cwd cannot resolve one",
}

// reasonFor returns the registered fallback reason for a kind, or ""
// when none exists. Callers should treat the empty string as "omit the
// field." Prefer attaching the reason at the construction site via
// exitcode.CodedError.WithReason() over adding entries here.
func reasonFor(kind string) string {
	return denialReasons[kind]
}

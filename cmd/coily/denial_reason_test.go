package main

import "testing"

// TestReasonFor_FallbackKinds pins the why-line fallbacks for kinds
// still emitted by cli-guard sites that have not migrated to
// exitcode.CodedError.WithReason(). Coily-side sites attach reasons at
// construction; the remaining map entries cover cli-guard internals.
// Drop entries here once the corresponding cli-guard site adds
// WithReason() upstream.
func TestReasonFor_FallbackKinds(t *testing.T) {
	required := []string{
		"policy_denied",
	}
	for _, k := range required {
		if got := reasonFor(k); got == "" {
			t.Errorf("reasonFor(%q) = empty; cli-guard-emitted kinds must carry a fallback why-line", k)
		}
	}
}

// TestReasonFor_PolicyDeniedNamesWorkarounds pins coilysiren/coily#164:
// the policy_denied reason must surface the canonical workarounds for
// the high-volume rejection shapes so agents can recover without
// re-tripping the gate on the next variant.
func TestReasonFor_PolicyDeniedNamesWorkarounds(t *testing.T) {
	got := reasonFor("policy_denied")
	for _, want := range []string{
		"--body-file",
		"file://",
		"--jq-file",
		"--args-file",
		"| jq",
	} {
		if !containsSubstr(got, want) {
			t.Errorf("policy_denied reason missing workaround mention %q in:\n%s", want, got)
		}
	}
}

func containsSubstr(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

func TestReasonFor_UnknownKindReturnsEmpty(t *testing.T) {
	if got := reasonFor("not_a_real_kind"); got != "" {
		t.Errorf("reasonFor(unknown) = %q, want empty", got)
	}
}

package main

import "testing"

// TestReasonFor_KnownKinds pins the load-bearing denial-reason mappings
// from coilysiren/coily#126 so a kind rename can't silently drop the
// why-line from the envelope. New kinds added here become required;
// kinds without a registered reason are tolerated (envelope omits the
// field) but the high-frequency denial classes must always carry one.
func TestReasonFor_KnownKinds(t *testing.T) {
	required := []string{
		"policy_denied",
		"repo_verb_dirty",
		"scope_unresolved",
	}
	for _, k := range required {
		if got := reasonFor(k); got == "" {
			t.Errorf("reasonFor(%q) = empty; high-frequency kinds must carry a why-line", k)
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

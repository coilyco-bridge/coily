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

func TestReasonFor_UnknownKindReturnsEmpty(t *testing.T) {
	if got := reasonFor("not_a_real_kind"); got != "" {
		t.Errorf("reasonFor(unknown) = %q, want empty", got)
	}
}

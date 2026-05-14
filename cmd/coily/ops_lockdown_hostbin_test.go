package main

import (
	"testing"

	"github.com/coilysiren/cli-guard/lockdown"
)

// TestApplyHostBinaryAllows_AddsJQAndYQ pins coilysiren/coily#163: jq
// and yq are pure non-shell evaluators (same safety class as grep/rg)
// and must land in the allow list so external pipes off coily wrappers
// don't trip the harness. Idempotent: re-running the helper does not
// duplicate entries.
func TestApplyHostBinaryAllows_AddsJQAndYQ(t *testing.T) {
	d, err := lockdown.LoadDefaults()
	if err != nil {
		t.Fatalf("LoadDefaults: %v", err)
	}
	got := applyHostBinaryAllows(d)
	for _, want := range []string{"Bash(jq:*)", "Bash(yq:*)"} {
		if !containsString(got.Allow, want) {
			t.Errorf("allow list missing %q after applyHostBinaryAllows", want)
		}
	}

	twice := applyHostBinaryAllows(got)
	count := 0
	for _, a := range twice.Allow {
		if a == "Bash(jq:*)" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Bash(jq:*) appears %d times after double-apply, want 1 (idempotent)", count)
	}
}

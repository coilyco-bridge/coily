package main

import (
	"strings"
	"testing"

	"github.com/coilysiren/cli-guard/lockdown"
)

// TestApplyWrapperAllows_AddsExplicitCoilyEntries pins the load-bearing
// behavior from coilysiren/coily#115: every bare-binary deny that maps
// to an audited coily wrapper gains an explicit `Bash(coily ...:*)`
// allow alongside the umbrella `Bash(coily:*)`. Without this the auto-
// mode classifier strips `coily ops gh` to `gh` and re-applies the
// bare deny (#159), making the audited path unreachable.
func TestApplyWrapperAllows_AddsExplicitCoilyEntries(t *testing.T) {
	d, err := lockdown.LoadDefaults()
	if err != nil {
		t.Fatalf("LoadDefaults: %v", err)
	}
	got := applyWrapperAllows(d)

	have := make(map[string]bool, len(got.Allow))
	for _, a := range got.Allow {
		have[a] = true
	}

	for deny, wantAllow := range wrapperAllows {
		if !containsString(got.Deny, deny) {
			// Deny isn't present in the canonical defaults; the mapping
			// still ships the allow, but skip the parity assertion for
			// this entry so a defaults.yaml change doesn't false-flag.
			continue
		}
		if !have[wantAllow] {
			t.Errorf("deny %q has no matching allow %q in augmented defaults", deny, wantAllow)
		}
	}
}

// TestWrapperAllowsParity_AllDenyEntriesCovered enforces the rule from
// the issue: a new wrapped verb cannot ship without its allow rule. If
// defaults.yaml denies a bare binary that has a `coily <wrapper>` form
// on disk, wrapperAllows must carry the mapping. Catches the drift case
// where someone adds a deny but forgets the allow.
//
// The list of wrapped binaries is intentionally enumerated here rather
// than auto-derived from the cli tree, because the cli tree has many
// non-wrapping verbs (audit, lockdown, dispatch). The seam is intent:
// "this deny exists because a coily wrapper takes its place."
func TestWrapperAllowsParity_AllDenyEntriesCovered(t *testing.T) {
	wrappedBinDenies := []string{
		"Bash(tailscale:*)", "Bash(docker:*)", "Bash(aws:*)",
		"Bash(kubectl:*)", "Bash(gh:*)", "Bash(flyctl:*)",
		"Bash(ssh:*)", "Bash(brew:*)",
		"Bash(npm:*)", "Bash(pnpm:*)", "Bash(yarn:*)",
		"Bash(uv:*)", "Bash(pip:*)", "Bash(pipx:*)", "Bash(poetry:*)",
		"Bash(cargo:*)", "Bash(gem:*)", "Bash(bundle:*)",
	}
	for _, deny := range wrappedBinDenies {
		allow, ok := wrapperAllows[deny]
		if !ok {
			t.Errorf("wrapperAllows missing entry for %q", deny)
			continue
		}
		if !strings.HasPrefix(allow, "Bash(coily ") {
			t.Errorf("wrapperAllows[%q] = %q, want a Bash(coily ...) allow", deny, allow)
		}
	}
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

package main

import (
	"strings"
	"testing"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/lockdown"
)

// TestApplyWrapperAllows_AddsExplicitCoilyEntries pins the load-bearing
// behavior from coilysiren/coily#115 and coilyco-bridge/coily#43: every
// audited coily wrapper gains an explicit `Bash(coily ...:*)` allow
// alongside the umbrella `Bash(coily:*)`, UNCONDITIONALLY - regardless of
// whether the matching bare deny is present in the canonical defaults.
// The auto-mode classifier strips `coily ops gh` to `gh` and re-applies
// the user-level bare deny (#43), so every wrapper needs the positive
// allow as its sanction signal even when the per-repo deny is trimmed.
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
		if !have[wantAllow] {
			t.Errorf("wrapper allow %q (for deny %q) missing from augmented defaults", wantAllow, deny)
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
		"Bash(gcloud:*)",
		"Bash(brew:*)",
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

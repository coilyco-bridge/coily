package main

import (
	"fmt"
	"strings"
	"testing"
)

// TestWrapperRecovery_CoversEveryPassthroughInTree is the anti-drift guard
// that motivated coilyco-bridge/coily#197: every passthrough leaf the
// command tree exposes (the bins `coily --tree --json` enumerates) must
// have a recovery mapping AND an explicit allow, and the recovery wrapper
// must route to that bin. Add a passthrough to ptOps/ptTopLevel/ptPkg and
// forget a surface, and this fails - the maps can no longer silently lag
// the registry the way the old hand-maintained literals did.
func TestWrapperRecovery_CoversEveryPassthroughInTree(t *testing.T) {
	r := newTestRunner(t)
	tokens := sortedCapabilityTokens(buildTree(r.builtInCommands(), nil))
	if len(tokens) == 0 {
		t.Fatal("no passthrough bins discovered in the tree")
	}
	for _, tok := range tokens {
		wrap, ok := wrapperRecovery[tok]
		if !ok {
			t.Errorf("passthrough %q in tree has no wrapperRecovery entry", tok)
			continue
		}
		if !strings.Contains(wrap, tok) {
			t.Errorf("wrapperRecovery[%q] = %q does not route to the bin", tok, wrap)
		}
		bareDeny := fmt.Sprintf("Bash(%s:*)", tok)
		allow, ok := wrapperAllows[bareDeny]
		if !ok {
			t.Errorf("passthrough %q in tree has no wrapperAllows entry (re-opens the #43 asymmetry)", tok)
			continue
		}
		if !strings.HasPrefix(allow, "Bash(coily ") || !strings.Contains(allow, tok) {
			t.Errorf("wrapperAllows[%q] = %q, want a Bash(coily ...%s...) allow", bareDeny, allow, tok)
		}
	}
}

// TestRecoveryWrappers_IncludeNonPassthroughIntentEntries proves the
// hand-listed intent entries (scoped pkg wrappers, build runners) survive
// the move to generation - they are not passthrough leaves, so only the
// explicit lists carry them.
func TestRecoveryWrappers_IncludeNonPassthroughIntentEntries(t *testing.T) {
	for _, bin := range []string{"brew", "scoop", "make", "just", "task", "invoke"} {
		if _, ok := wrapperRecovery[bin]; !ok {
			t.Errorf("wrapperRecovery lost intent entry %q", bin)
		}
	}
	// Build runners are recovery-only: no explicit allow.
	for _, runner := range []string{"make", "just", "task", "invoke"} {
		if _, ok := wrapperAllows[fmt.Sprintf("Bash(%s:*)", runner)]; ok {
			t.Errorf("build runner %q should not have an explicit allow", runner)
		}
	}
	// Scoped pkg wrappers DO earn an allow.
	for _, scoped := range []string{"brew", "scoop"} {
		if _, ok := wrapperAllows[fmt.Sprintf("Bash(%s:*)", scoped)]; !ok {
			t.Errorf("scoped wrapper %q should have an explicit allow", scoped)
		}
	}
}

package main

import (
	"strings"
	"testing"
)

// TestUpgradeFormula_LockedToCoilysirenPerRepo pins coilysiren/coily#19
// + coilyco-bridge/coily#22: the upgrade verb is bound to coily-the-
// formula specifically, and the formula is the per-repo tap shape
// coilysiren/coily/coily. Widening it to a non-coilysiren tap would
// defeat the "audited self-update" claim by letting an agent upgrade
// arbitrary formulae through the audited path; that's what `coily brew
// upgrade` is for. The umbrella shape (coilysiren/tap/coily) is no
// longer accepted here because that tap was decommissioned.
func TestUpgradeFormula_LockedToCoilysirenPerRepo(t *testing.T) {
	if upgradeFormula != "coilysiren/coily/coily" {
		t.Errorf("upgradeFormula = %q, want %q", upgradeFormula, "coilysiren/coily/coily")
	}
	if !strings.HasPrefix(upgradeFormula, "coilysiren/") {
		t.Errorf("upgradeFormula = %q must live under coilysiren/", upgradeFormula)
	}
	if !strings.HasSuffix(upgradeFormula, "/coily") {
		t.Errorf("upgradeFormula = %q must name the coily formula", upgradeFormula)
	}
}

// TestUpgradeCommand_HasDryFlag pins the --dry escape hatch from #19's
// shape sketch.
func TestUpgradeCommand_HasDryFlag(t *testing.T) {
	r := NewRunner()
	cmd := r.upgradeCommand()
	for _, f := range cmd.Flags {
		if f.Names()[0] == "dry" {
			return
		}
	}
	t.Error("upgrade command missing --dry flag")
}

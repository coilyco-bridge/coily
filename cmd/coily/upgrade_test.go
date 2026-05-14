package main

import (
	"strings"
	"testing"
)

// TestUpgradeFormula_LockedToCoilyTap pins coilysiren/coily#19: the
// upgrade verb's whole point is binding to coilysiren/tap/coily
// specifically. Widening the formula to a knob would defeat the
// "audited self-update" claim by letting an agent upgrade arbitrary
// formulae through the audited path; that's what `coily brew upgrade`
// is for.
func TestUpgradeFormula_LockedToCoilyTap(t *testing.T) {
	if upgradeFormula != "coilysiren/tap/coily" {
		t.Errorf("upgradeFormula = %q, want coilysiren/tap/coily; widening this needs a deliberate review", upgradeFormula)
	}
	if !strings.HasPrefix(upgradeFormula, "coilysiren/tap/") {
		t.Error("upgradeFormula must live under coilysiren/tap/")
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

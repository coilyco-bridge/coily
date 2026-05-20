package main

import (
	"strings"
	"testing"
)

// TestUpgradeFormulaCandidates_LockedToCoilysiren pins coilysiren/coily#19
// + coilysiren/coily#271: the upgrade verb is bound to coily-the-formula
// specifically, and the only knob is which coilysiren tap is providing
// it. Widening either candidate to a non-coilysiren tap would defeat
// the "audited self-update" claim by letting an agent upgrade arbitrary
// formulae through the audited path; that's what `coily brew upgrade`
// is for.
func TestUpgradeFormulaCandidates_LockedToCoilysiren(t *testing.T) {
	candidates := []struct {
		name    string
		formula string
	}{
		{"per-repo tap", upgradeFormulaPerRepo},
		{"umbrella tap", upgradeFormulaUmbrella},
	}
	for _, c := range candidates {
		t.Run(c.name, func(t *testing.T) {
			if !strings.HasPrefix(c.formula, "coilysiren/") {
				t.Errorf("%s = %q must live under coilysiren/", c.name, c.formula)
			}
			if !strings.HasSuffix(c.formula, "/coily") {
				t.Errorf("%s = %q must name the coily formula", c.name, c.formula)
			}
		})
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

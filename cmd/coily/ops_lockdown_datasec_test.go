package main

import (
	"os"
	"strings"
	"testing"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/lockdown"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/profile"
)

func TestApplyDataSecurityDenies_LowNoChange(t *testing.T) {
	d := &lockdown.Defaults{Allow: []string{"a"}, Deny: []string{"d"}}
	drv := &lockdown.Driver{Coordinate: &profile.Coordinate{DataSecurity: profile.DataSecurityLow}}
	got := applyDataSecurityDenies(d, drv)
	if len(got.Deny) != 1 {
		t.Errorf("low should not extend deny list, got %v", got.Deny)
	}
}

func TestApplyDataSecurityDenies_HighAddsVaultDenies(t *testing.T) {
	d := &lockdown.Defaults{Deny: []string{"base"}}
	drv := &lockdown.Driver{Coordinate: &profile.Coordinate{DataSecurity: profile.DataSecurityHigh}}
	got := applyDataSecurityDenies(d, drv)
	found := false
	for _, e := range got.Deny {
		if strings.Contains(e, "coilyco-vault") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("high tier should add vault deny, got %v", got.Deny)
	}
}

func TestApplyDataSecurityDenies_MaxIncludesVaultDeny(t *testing.T) {
	d := &lockdown.Defaults{Deny: []string{"base"}}
	drv := &lockdown.Driver{Coordinate: &profile.Coordinate{DataSecurity: profile.DataSecurityMax}}
	got := applyDataSecurityDenies(d, drv)
	foundVault := false
	for _, e := range got.Deny {
		if strings.Contains(e, "coilyco-vault") {
			foundVault = true
		}
	}
	if !foundVault {
		t.Errorf("max should include vault deny, got %v", got.Deny)
	}
}

// TestApplyDataSecurityDenies_VaultPathIsPortable pins the
// post-cli-guard#14 invariant: no host-specific path is hardcoded.
// The absolute form is derived from os.UserHomeDir() at render time,
// not baked into source.
func TestApplyDataSecurityDenies_VaultPathIsPortable(t *testing.T) {
	d := &lockdown.Defaults{Deny: []string{"base"}}
	drv := &lockdown.Driver{Coordinate: &profile.Coordinate{DataSecurity: profile.DataSecurityHigh}}
	got := applyDataSecurityDenies(d, drv)
	home, _ := os.UserHomeDir()
	for _, e := range got.Deny {
		if !strings.Contains(e, "coilyco-vault") {
			continue
		}
		if strings.HasPrefix(e, "Read(~") {
			continue
		}
		if home != "" && strings.Contains(e, home) {
			continue
		}
		t.Errorf("vault deny is not portable (not ~/, not runtime home %q): %s", home, e)
	}
}

// TestApplyDataSecurityDenies_EmitsBothPathForms pins coily#111: the
// vault deny ships both the tilde form and the runtime-resolved
// absolute form so Claude Code's literal string compare cannot be
// bypassed by switching path representations.
func TestApplyDataSecurityDenies_EmitsBothPathForms(t *testing.T) {
	d := &lockdown.Defaults{Deny: []string{"base"}}
	drv := &lockdown.Driver{Coordinate: &profile.Coordinate{DataSecurity: profile.DataSecurityHigh}}
	got := applyDataSecurityDenies(d, drv)
	wantTilde := "Read(~/projects/coilysiren/coilyco-vault/**)"
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("os.UserHomeDir() unavailable: %v", err)
	}
	wantAbs := "Read(" + home + "/projects/coilysiren/coilyco-vault/**)"
	foundTilde, foundAbs := false, false
	for _, e := range got.Deny {
		if e == wantTilde {
			foundTilde = true
		}
		if e == wantAbs {
			foundAbs = true
		}
	}
	if !foundTilde {
		t.Errorf("missing tilde form %q in %v", wantTilde, got.Deny)
	}
	if !foundAbs {
		t.Errorf("missing absolute form %q in %v", wantAbs, got.Deny)
	}
}

func TestApplyDataSecurityDenies_DoesNotMutateInput(t *testing.T) {
	d := &lockdown.Defaults{Deny: []string{"only"}}
	drv := &lockdown.Driver{Coordinate: &profile.Coordinate{DataSecurity: profile.DataSecurityMax}}
	_ = applyDataSecurityDenies(d, drv)
	if len(d.Deny) != 1 {
		t.Errorf("input mutated: %v", d.Deny)
	}
}

func TestApplyDataSecurityDenies_NilDriverIsNoop(t *testing.T) {
	d := &lockdown.Defaults{Deny: []string{"x"}}
	got := applyDataSecurityDenies(d, nil)
	if got != d {
		t.Errorf("nil driver should be passthrough")
	}
}

func TestApplyDataSecurityDenies_NilCoordinateIsNoop(t *testing.T) {
	d := &lockdown.Defaults{Deny: []string{"x"}}
	drv := &lockdown.Driver{}
	got := applyDataSecurityDenies(d, drv)
	if got != d {
		t.Errorf("nil coordinate should be passthrough")
	}
}

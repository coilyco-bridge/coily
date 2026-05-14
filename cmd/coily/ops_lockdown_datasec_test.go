package main

import (
	"strings"
	"testing"

	"github.com/coilysiren/cli-guard/lockdown"
	"github.com/coilysiren/cli-guard/profile"
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

func TestApplyDataSecurityDenies_MaxAddsBashEchoDenies(t *testing.T) {
	d := &lockdown.Defaults{Deny: []string{"base"}}
	drv := &lockdown.Driver{Coordinate: &profile.Coordinate{DataSecurity: profile.DataSecurityMax}}
	got := applyDataSecurityDenies(d, drv)
	foundEcho := false
	foundVault := false
	for _, e := range got.Deny {
		if strings.HasPrefix(e, "Bash(echo") {
			foundEcho = true
		}
		if strings.Contains(e, "coilyco-vault") {
			foundVault = true
		}
	}
	if !foundEcho {
		t.Errorf("max should add Bash echo denies, got %v", got.Deny)
	}
	if !foundVault {
		t.Errorf("max should still include vault deny, got %v", got.Deny)
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

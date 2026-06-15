package main

import (
	"testing"
)

func TestPrimaryOrgsDefaultAndOverride(t *testing.T) {
	r := &Runner{Cfg: &Config{}}
	got := r.primaryOrgs()
	if len(got) != 3 || got[0] != "coilysiren" {
		t.Errorf("empty config should yield default primary orgs, got %v", got)
	}

	r2 := &Runner{Cfg: &Config{PrimaryOrgs: []string{"only-org"}}}
	if g := r2.primaryOrgs(); len(g) != 1 || g[0] != "only-org" {
		t.Errorf("configured primary orgs should win, got %v", g)
	}
}

func TestIsPrimaryOrg(t *testing.T) {
	orgs := defaultPrimaryOrgs()
	for _, o := range []string{"coilysiren", "coilyco-bridge", "coilyco-flight-deck"} {
		if !isPrimaryOrg(orgs, o) {
			t.Errorf("%q should be a primary org", o)
		}
	}
	for _, o := range []string{"someuser", "", "coilysiren-x"} {
		if isPrimaryOrg(orgs, o) {
			t.Errorf("%q should not be a primary org", o)
		}
	}
}

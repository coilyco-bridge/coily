package main

import (
	"os"
	"path/filepath"
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

// TestLocalRepoPathScan pins #173 part 2: a repo whose checkout lives under a
// non-default primary org is found by scanning, and a missing repo falls back
// to the historical coilysiren path (preserving the prior error shape).
func TestLocalRepoPathScan(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// coily lives under coilyco-bridge post-rename, not coilysiren.
	want := filepath.Join(home, "projects", "coilyco-bridge", "coily")
	if err := os.MkdirAll(want, 0o755); err != nil {
		t.Fatal(err)
	}

	r := &Runner{Cfg: &Config{PrimaryOrgs: defaultPrimaryOrgs()}}

	got, err := r.localRepoPath("coily")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("localRepoPath(coily) = %q, want %q (on-disk scan)", got, want)
	}

	// Missing repo: fall back to the historical coilysiren path.
	missing, err := r.localRepoPath("nope")
	if err != nil {
		t.Fatal(err)
	}
	wantFallback := filepath.Join(home, "projects", "coilysiren", "nope")
	if missing != wantFallback {
		t.Errorf("missing repo fallback = %q, want %q", missing, wantFallback)
	}
}

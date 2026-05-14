package profiles

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coilysiren/cli-guard/profile"
)

// TestDefaultYAML_ParsesAndValidates confirms the embedded template
// the user copies via `coily lockdown init-config` is itself valid.
// Without this test a typo in default.yaml would silently land in a
// release and fail at first user-copy.
func TestDefaultYAML_ParsesAndValidates(t *testing.T) {
	got, err := ParseAndValidate(DefaultYAML, "embedded default.yaml")
	if err != nil {
		t.Fatalf("embedded default.yaml does not validate: %v", err)
	}
	wantNames := []string{"mobile", "mac-tower", "windows-laptop", "web", "headless"}
	for _, n := range wantNames {
		if _, ok := got[n]; !ok {
			t.Errorf("embedded default.yaml missing profile %q", n)
		}
	}
	if got["headless"].NetworkEgress != profile.NetworkEgressAirGapped {
		t.Errorf("headless network_egress = %q, want air-gapped",
			got["headless"].NetworkEgress)
	}
	if got["mac-tower"].FilesystemReach != profile.FilesystemReachUnrestricted {
		t.Errorf("mac-tower filesystem_reach = %q, want unrestricted",
			got["mac-tower"].FilesystemReach)
	}
}

func TestParseAndValidate_MissingAxisFails(t *testing.T) {
	body := []byte(`profiles:
  partial:
    data_security: high
    blast_radius: medium
    network_egress: allowlisted
`)
	_, err := ParseAndValidate(body, "test")
	if err == nil {
		t.Fatal("expected error on partial profile, got nil")
	}
	if !strings.Contains(err.Error(), "filesystem_reach") {
		t.Errorf("error should name the missing axis, got: %v", err)
	}
}

func TestParseAndValidate_UnknownTierFails(t *testing.T) {
	body := []byte(`profiles:
  weird:
    data_security: ultra
    blast_radius: medium
    network_egress: allowlisted
    filesystem_reach: repo-only
`)
	_, err := ParseAndValidate(body, "test")
	if err == nil {
		t.Fatal("expected error on unknown tier, got nil")
	}
	if !strings.Contains(err.Error(), "ultra") {
		t.Errorf("error should name the bad tier, got: %v", err)
	}
}

func TestParseAndValidate_EmptyFails(t *testing.T) {
	_, err := ParseAndValidate([]byte("profiles: {}\n"), "test")
	if err == nil {
		t.Fatal("expected error on empty profiles map")
	}
}

func TestParseAndValidate_BadProfileNameFails(t *testing.T) {
	body := []byte(`profiles:
  Bad-Name:
    data_security: high
    blast_radius: medium
    network_egress: allowlisted
    filesystem_reach: repo-only
`)
	_, err := ParseAndValidate(body, "test")
	if err == nil {
		t.Fatal("expected error on uppercase profile name")
	}
}

// withHome stages a tempdir as $HOME, returning the override path.
func withHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	return filepath.Join(dir, ".coily", "coily.yaml")
}

func TestResolve_MissingFileFallsBackToStrictest(t *testing.T) {
	withHome(t)
	got, err := Resolve("mobile")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got.Source != SourceMissingFile {
		t.Errorf("source = %q, want missing_file", got.Source)
	}
	if got.Coord != profile.Strictest() {
		t.Errorf("coord = %+v, want Strictest", got.Coord)
	}
}

func TestResolve_UnsetSentinelFallsBackToStrictest(t *testing.T) {
	withHome(t)
	got, err := Resolve("")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got.Source != SourceUnset {
		t.Errorf("source = %q, want unset", got.Source)
	}
	if got.Coord != profile.Strictest() {
		t.Errorf("coord = %+v, want Strictest", got.Coord)
	}
}

func TestResolve_OverrideHit(t *testing.T) {
	path := withHome(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, DefaultYAML, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := Resolve("mobile")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got.Source != SourceOverride {
		t.Errorf("source = %q, want override", got.Source)
	}
	if got.Coord.DataSecurity != profile.DataSecurityHigh {
		t.Errorf("mobile data_security = %q, want high", got.Coord.DataSecurity)
	}
	if got.Coord.NetworkEgress != profile.NetworkEgressAllowlisted {
		t.Errorf("mobile network_egress = %q, want allowlisted", got.Coord.NetworkEgress)
	}
}

func TestResolve_UnknownProfileFallsBackToStrictest(t *testing.T) {
	path := withHome(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, DefaultYAML, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := Resolve("does-not-exist")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got.Source != SourceUnknownProfile {
		t.Errorf("source = %q, want unknown_profile", got.Source)
	}
	if got.Coord != profile.Strictest() {
		t.Errorf("coord = %+v, want Strictest", got.Coord)
	}
}

func TestResolve_MalformedFileErrorsLoud(t *testing.T) {
	path := withHome(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("profiles:\n  mobile:\n    data_security: ultra\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := Resolve("mobile")
	if err == nil {
		t.Fatal("expected loud error on malformed override, got nil")
	}
	if errors.Is(err, os.ErrNotExist) {
		t.Errorf("malformed file should not surface as ErrNotExist, got: %v", err)
	}
}

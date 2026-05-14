package decision

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/coilysiren/cli-guard/profile"
	"github.com/coilysiren/coily/pkg/profiles"
)

func withHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	return filepath.Join(dir, ".coily", "coily.yaml")
}

func TestEvaluate_MissingFileReturnsAllowWithStrictest(t *testing.T) {
	withHome(t)
	pd, err := Evaluate("mobile")
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if pd == nil || !pd.Allowed {
		t.Fatalf("expected allow, got %+v", pd)
	}
	if pd.Source != string(profiles.SourceMissingFile) {
		t.Errorf("source = %q, want missing_file", pd.Source)
	}
	if pd.Coordinate.DataSecurity != "max" {
		t.Errorf("data_security = %q, want max (Strictest)", pd.Coordinate.DataSecurity)
	}
}

func TestEvaluate_UnsetReturnsAllowWithStrictest(t *testing.T) {
	withHome(t)
	pd, err := Evaluate("")
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if pd.Source != string(profiles.SourceUnset) {
		t.Errorf("source = %q, want unset", pd.Source)
	}
}

func TestEvaluate_OverrideHitReturnsResolvedCoordinate(t *testing.T) {
	path := withHome(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, profiles.DefaultYAML, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	pd, err := Evaluate("mac-tower")
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if pd.Source != string(profiles.SourceOverride) {
		t.Errorf("source = %q, want override", pd.Source)
	}
	if pd.Coordinate.DataSecurity != string(profile.DataSecurityMedium) {
		t.Errorf("data_security = %q, want medium", pd.Coordinate.DataSecurity)
	}
	if pd.Coordinate.FilesystemReach != string(profile.FilesystemReachUnrestricted) {
		t.Errorf("filesystem_reach = %q, want unrestricted", pd.Coordinate.FilesystemReach)
	}
}

func TestCoordinatePtr_NotNilOnMissingFile(t *testing.T) {
	withHome(t)
	c, err := CoordinatePtr("anything")
	if err != nil {
		t.Fatalf("coord ptr: %v", err)
	}
	if c == nil {
		t.Fatal("CoordinatePtr returned nil on missing file; expected Strictest fallback")
	}
	if c.DataSecurity != profile.DataSecurityMax {
		t.Errorf("data_security = %q, want max", c.DataSecurity)
	}
}

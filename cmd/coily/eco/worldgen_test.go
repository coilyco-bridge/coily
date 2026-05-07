package eco_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coilysiren/coily/cmd/coily/eco"
)

// sampleWorldGen returns a minimal WorldGenerator.eco shape that includes
// the HeightmapModule.Source.Config.Seed field plus enough surrounding
// noise to verify SetSeed leaves the rest of the document alone.
func sampleWorldGen(seed int64) string {
	doc := map[string]any{
		"HeightmapModule": map[string]any{
			"Source": map[string]any{
				"Config": map[string]any{
					"Seed":      seed,
					"OtherKey":  "preserve me",
					"Threshold": 0.5,
				},
			},
		},
		"Unrelated": "preserve me too",
	}
	b, _ := json.MarshalIndent(doc, "", "  ")
	return string(b) + "\n"
}

func writeSample(t *testing.T, dir string, seed int64) {
	t.Helper()
	path := eco.WorldGenPath(dir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(sampleWorldGen(seed)), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGetSeed_RoundTripsWrittenValue(t *testing.T) {
	dir := t.TempDir()
	writeSample(t, dir, 12345)
	got, err := eco.GetSeed(dir)
	if err != nil {
		t.Fatalf("GetSeed: %v", err)
	}
	if got != 12345 {
		t.Errorf("GetSeed = %d, want 12345", got)
	}
}

func TestGetSeed_FileMissing(t *testing.T) {
	_, err := eco.GetSeed(t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestGetSeed_FieldMissing(t *testing.T) {
	dir := t.TempDir()
	path := eco.WorldGenPath(dir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"OtherStuff":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := eco.GetSeed(dir)
	if !errors.Is(err, eco.ErrSeedFieldMissing) {
		t.Errorf("err = %v, want ErrSeedFieldMissing", err)
	}
}

func TestSetSeed_PreservesSiblingKeys(t *testing.T) {
	dir := t.TempDir()
	writeSample(t, dir, 1)
	if err := eco.SetSeed(dir, 9999); err != nil {
		t.Fatalf("SetSeed: %v", err)
	}
	got, err := eco.GetSeed(dir)
	if err != nil {
		t.Fatalf("GetSeed: %v", err)
	}
	if got != 9999 {
		t.Errorf("seed = %d, want 9999", got)
	}
	// Verify sibling keys survived.
	raw, err := os.ReadFile(eco.WorldGenPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	cfg := doc["HeightmapModule"].(map[string]any)["Source"].(map[string]any)["Config"].(map[string]any)
	if cfg["OtherKey"] != "preserve me" {
		t.Errorf("OtherKey lost, got %v", cfg["OtherKey"])
	}
	if doc["Unrelated"] != "preserve me too" {
		t.Errorf("Unrelated lost, got %v", doc["Unrelated"])
	}
	// Trailing newline check, matches worldgen.py.
	if !strings.HasSuffix(string(raw), "\n") {
		t.Error("output missing trailing newline")
	}
}

func TestSetSeed_RejectsOutOfRange(t *testing.T) {
	dir := t.TempDir()
	writeSample(t, dir, 1)
	for _, bad := range []int64{0, -1, eco.SeedMax + 1} {
		if err := eco.SetSeed(dir, bad); !errors.Is(err, eco.ErrSeedOutOfRange) {
			t.Errorf("seed=%d: err = %v, want ErrSeedOutOfRange", bad, err)
		}
	}
}

func TestRandomSeed_InRange(t *testing.T) {
	for i := 0; i < 100; i++ {
		s, err := eco.RandomSeed()
		if err != nil {
			t.Fatalf("RandomSeed: %v", err)
		}
		if s < 1 || s > eco.SeedMax {
			t.Errorf("seed %d out of [1,%d]", s, eco.SeedMax)
		}
	}
}

func TestSnapshot_CopiesFile(t *testing.T) {
	dir := t.TempDir()
	writeSample(t, dir, 42)
	target := filepath.Join(t.TempDir(), "nested", "snap.eco")
	if err := eco.Snapshot(dir, target); err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	src, err := os.ReadFile(eco.WorldGenPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	dst, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(src) != string(dst) {
		t.Errorf("snapshot mismatch\nsrc: %s\ndst: %s", src, dst)
	}
}

// Package eco mirrors the local-side world-generation helpers that Kai's
// eco-cycle-prep repo provides. The reference implementation lives at
//
//	https://github.com/coilysiren/eco-cycle-prep/blob/main/eco_cycle_prep/worldgen.py
//
// Scope. This package only covers the local file ops on the eco-configs
// repo (read/write the Seed in WorldGenerator.eco, copy the file to a
// snapshot path). The remote restart-the-eco-server flow already lives in
// the existing `coily eco {stop,start,restart,status,tail}` verbs and is
// not duplicated here.
package eco

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
)

// SeedMax matches eco-cycle-prep/worldgen.py: Eco's Seed field is an int,
// kept well under int32 max so it round-trips through every game tool.
const SeedMax = 2_000_000_000

// WorldGenRelPath is the location of WorldGenerator.eco inside an
// eco-configs checkout. Matches worldgen.py.
const WorldGenRelPath = "Configs/WorldGenerator.eco"

// ErrSeedOutOfRange is returned by SetSeed when seed is <= 0 or > SeedMax.
var ErrSeedOutOfRange = errors.New("eco: seed out of range")

// ErrSeedFieldMissing is returned by GetSeed/SetSeed when the WorldGenerator
// JSON does not contain HeightmapModule.Source.Config.Seed. Indicates the
// file shape changed and this package needs updating.
var ErrSeedFieldMissing = errors.New("eco: WorldGenerator HeightmapModule.Source.Config.Seed missing")

// WorldGenPath returns the absolute path to WorldGenerator.eco inside the
// given eco-configs directory.
func WorldGenPath(configsDir string) string {
	return filepath.Join(configsDir, WorldGenRelPath)
}

// RandomSeed returns a cryptographically random seed in [1, SeedMax].
// crypto/rand is overkill for game seeds but it removes the need for any
// global rand seeding dance and keeps determinism explicit (callers that
// want a specific seed pass --seed instead).
func RandomSeed() (int64, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(SeedMax))
	if err != nil {
		return 0, fmt.Errorf("eco: rand: %w", err)
	}
	return n.Int64() + 1, nil
}

// GetSeed reads WorldGenerator.eco at configsDir and returns the current
// HeightmapModule.Source.Config.Seed value.
func GetSeed(configsDir string) (int64, error) {
	data, err := readWorldGen(configsDir)
	if err != nil {
		return 0, err
	}
	return readSeed(data)
}

// SetSeed writes seed into WorldGenerator.eco at configsDir, preserving the
// rest of the JSON. Mirrors worldgen.py: serialize with indent=2 and
// sort_keys=True, append a trailing newline.
func SetSeed(configsDir string, seed int64) error {
	if seed <= 0 || seed > SeedMax {
		return fmt.Errorf("%w: %d (must be 1..%d)", ErrSeedOutOfRange, seed, SeedMax)
	}
	data, err := readWorldGen(configsDir)
	if err != nil {
		return err
	}
	if err := writeSeed(data, seed); err != nil {
		return err
	}
	out, err := marshalSorted(data)
	if err != nil {
		return fmt.Errorf("eco: marshal: %w", err)
	}
	out = append(out, '\n')
	if err := os.WriteFile(WorldGenPath(configsDir), out, 0o644); err != nil {
		return fmt.Errorf("eco: write %s: %w", WorldGenPath(configsDir), err)
	}
	return nil
}

// Snapshot copies WorldGenerator.eco at configsDir to target, creating the
// parent directory if needed. Mirrors worldgen.py snapshot().
func Snapshot(configsDir, target string) error {
	src := WorldGenPath(configsDir)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("eco: mkdir %s: %w", filepath.Dir(target), err)
	}
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("eco: open %s: %w", src, err)
	}
	defer in.Close()
	out, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("eco: create %s: %w", target, err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("eco: copy %s -> %s: %w", src, target, err)
	}
	return nil
}

func readWorldGen(configsDir string) (map[string]any, error) {
	path := WorldGenPath(configsDir)
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("eco: read %s: %w", path, err)
	}
	var data map[string]any
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("eco: parse %s: %w", path, err)
	}
	return data, nil
}

// readSeed walks HeightmapModule.Source.Config.Seed.
func readSeed(data map[string]any) (int64, error) {
	hm, ok := asMap(data["HeightmapModule"])
	if !ok {
		return 0, ErrSeedFieldMissing
	}
	src, ok := asMap(hm["Source"])
	if !ok {
		return 0, ErrSeedFieldMissing
	}
	cfg, ok := asMap(src["Config"])
	if !ok {
		return 0, ErrSeedFieldMissing
	}
	v, ok := cfg["Seed"]
	if !ok {
		return 0, ErrSeedFieldMissing
	}
	switch n := v.(type) {
	case float64:
		return int64(n), nil
	case int64:
		return n, nil
	case int:
		return int64(n), nil
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return 0, fmt.Errorf("eco: seed not an int: %w", err)
		}
		return i, nil
	default:
		return 0, fmt.Errorf("eco: seed has unexpected type %T", v)
	}
}

// writeSeed sets HeightmapModule.Source.Config.Seed in-place. Returns
// ErrSeedFieldMissing if the parent path does not exist; we do not invent
// missing parents because doing so would silently corrupt the file shape.
func writeSeed(data map[string]any, seed int64) error {
	hm, ok := asMap(data["HeightmapModule"])
	if !ok {
		return ErrSeedFieldMissing
	}
	src, ok := asMap(hm["Source"])
	if !ok {
		return ErrSeedFieldMissing
	}
	cfg, ok := asMap(src["Config"])
	if !ok {
		return ErrSeedFieldMissing
	}
	cfg["Seed"] = seed
	return nil
}

func asMap(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

// marshalSorted matches the Python reference's json.dumps(indent=2, sort_keys=True)
// output. encoding/json already emits map keys in sorted order, so a plain
// Indent over Marshal gets us the same byte stream (modulo the trailing
// newline which the caller appends).
func marshalSorted(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

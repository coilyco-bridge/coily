package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestEcoWorld_HelpListsAllSubVerbs runs `go run ./cmd/coily eco world --help`
// and verifies every world sub-verb appears. The point is to catch a
// commit that drops a sub-verb from ecoWorldCmd.Commands. Cheaper than a
// real action test because it doesn't need a kai-server, an eco-configs
// checkout, or a token.
func TestEcoWorld_HelpListsAllSubVerbs(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/coily", "eco", "world", "--help")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("eco world --help failed: %v\n%s", err, out)
	}
	got := string(out)
	for _, want := range []string{"get-seed", "set-seed", "randomize", "snapshot"} {
		if !strings.Contains(got, want) {
			t.Errorf("help output missing sub-verb %q\n%s", want, got)
		}
	}
}

// TestEcoWorld_SetSeedRejectsMissingToken proves the Mutating policy is
// wired up. Invoking set-seed without a token should fail with the
// token-required error before any file IO happens.
func TestEcoWorld_SetSeedRejectsMissingToken(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/coily",
		"eco", "world", "set-seed",
		"--configs-dir", t.TempDir(),
		"--seed", "12345",
	)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit, got success\n%s", out)
	}
	if !strings.Contains(string(out), "token") {
		t.Errorf("expected token-required error, got:\n%s", out)
	}
}

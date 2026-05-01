package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestEcoWorld_HelpListsAllSubVerbs runs
// `go run ./cmd/coily gaming eco world --help` and verifies every world
// sub-verb appears. The point is to catch a commit that drops a sub-verb
// from ecoWorldCmd.Commands.
func TestEcoWorld_HelpListsAllSubVerbs(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/coily", "gaming", "eco", "world", "--help")
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

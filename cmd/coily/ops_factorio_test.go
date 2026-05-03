package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestFactorioMods_HelpListsListAndSync runs
// `go run ./cmd/coily gaming factorio mods --help` and verifies both the
// existing list verb and the newly added sync verb appear. Catches a
// commit that drops sync from factorioModsCommand.Commands.
func TestFactorioMods_HelpListsListAndSync(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/coily", "gaming", "factorio", "mods", "--help")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("factorio mods --help failed: %v\n%s", err, out)
	}
	got := string(out)
	for _, want := range []string{"list", "sync"} {
		if !strings.Contains(got, want) {
			t.Errorf("help output missing sub-verb %q\n%s", want, got)
		}
	}
}

// TestFactorioMods_SyncHelpListsFlags verifies the sync verb advertises
// its --dry-run and --mod flags so callers discover them via --help.
func TestFactorioMods_SyncHelpListsFlags(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/coily", "gaming", "factorio", "mods", "sync", "--help")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("factorio mods sync --help failed: %v\n%s", err, out)
	}
	got := string(out)
	for _, want := range []string{"--dry-run", "--mod"} {
		if !strings.Contains(got, want) {
			t.Errorf("help output missing flag %q\n%s", want, got)
		}
	}
}

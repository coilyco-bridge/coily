package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
)

// runLockdown invokes lockdownAction via a stand-in cli.Command that mirrors
// the real lockdownCmd's flag schema. We don't call into the live verb.Wrap
// pipeline here because that wires up runtime audit state we do not need
// for these action-level tests.
func runLockdown(t *testing.T, dir string, apply, replace bool) error {
	t.Helper()
	return runLockdownFlags(t, dir, apply, replace, false)
}

func runLockdownFlags(t *testing.T, dir string, apply, replace, recursive bool) error {
	t.Helper()
	root := &cli.Command{
		Name: "test-root",
		Commands: []*cli.Command{
			{
				Name: "lockdown",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "path", Value: dir},
					&cli.BoolFlag{Name: "local"},
					&cli.BoolFlag{Name: "apply"},
					&cli.BoolFlag{Name: "replace"},
					&cli.BoolFlag{Name: "recursive"},
					&cli.StringFlag{Name: "token"},
				},
				Action: lockdownAction,
			},
		},
	}
	args := []string{"test-root", "lockdown", "--path", dir}
	if apply {
		args = append(args, "--apply")
	}
	if replace {
		args = append(args, "--replace")
	}
	if recursive {
		args = append(args, "--recursive")
	}
	return root.Run(context.Background(), args)
}

func TestLockdown_NoFlagsIsDryRun(t *testing.T) {
	dir := t.TempDir()
	if err := runLockdown(t, dir, false, false); err != nil {
		t.Fatalf("dry-run errored: %v", err)
	}
	target := filepath.Join(dir, ".claude", "settings.json")
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Errorf("dry-run wrote a file at %s (err=%v)", target, err)
	}
}

func TestLockdown_ApplyOnFreshRepoWritesFile(t *testing.T) {
	dir := t.TempDir()
	if err := runLockdown(t, dir, true, false); err != nil {
		t.Fatalf("apply on fresh repo errored: %v", err)
	}
	target := filepath.Join(dir, ".claude", "settings.json")
	if _, err := os.Stat(target); err != nil {
		t.Errorf("expected %s to exist: %v", target, err)
	}
}

func TestLockdown_ApplyRefusesExistingFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(`{"permissions":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	err := runLockdown(t, dir, true, false)
	if err == nil {
		t.Fatal("expected error when --apply hits an existing file")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error message did not mention 'already exists': %v", err)
	}
	// File should be unchanged.
	got, _ := os.ReadFile(target)
	if string(got) != `{"permissions":{}}` {
		t.Errorf("file was modified despite refusal: %q", string(got))
	}
}

func TestLockdown_ReplaceWithoutApplyErrors(t *testing.T) {
	dir := t.TempDir()
	err := runLockdown(t, dir, false, true)
	if err == nil {
		t.Fatal("expected error for --replace without --apply")
	}
	if !strings.Contains(err.Error(), "--replace requires --apply") {
		t.Errorf("error message did not mention required flag combo: %v", err)
	}
}

func TestLockdown_RecursiveAppliesToEachGitRepo(t *testing.T) {
	root := t.TempDir()
	repos := []string{
		filepath.Join(root, "a"),
		filepath.Join(root, "nested", "b"),
		filepath.Join(root, "x", "y", "z", "deep"), // depth 4, allowed
	}
	for _, r := range repos {
		if err := os.MkdirAll(filepath.Join(r, ".git"), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	// Out-of-range repo at depth 5 should be ignored.
	tooDeep := filepath.Join(root, "x", "y", "z", "deep2", "skip")
	if err := os.MkdirAll(filepath.Join(tooDeep, ".git"), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := runLockdownFlags(t, root, true, false, true); err != nil {
		t.Fatalf("recursive apply errored: %v", err)
	}
	for _, r := range repos {
		target := filepath.Join(r, ".claude", "settings.json")
		if _, err := os.Stat(target); err != nil {
			t.Errorf("expected %s to exist: %v", target, err)
		}
	}
	skipped := filepath.Join(tooDeep, ".claude", "settings.json")
	if _, err := os.Stat(skipped); !os.IsNotExist(err) {
		t.Errorf("repo beyond max depth was locked down: %s (err=%v)", skipped, err)
	}
}

func TestLockdown_RecursiveNoReposErrors(t *testing.T) {
	root := t.TempDir()
	err := runLockdownFlags(t, root, false, false, true)
	if err == nil || !strings.Contains(err.Error(), "no git repos") {
		t.Fatalf("expected 'no git repos' error, got %v", err)
	}
}

func TestLockdown_ApplyReplaceOverwrites(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		t.Fatal(err)
	}
	original := `{"permissions":{"allow":["Bash(custom-tool:*)"]}}`
	if err := os.WriteFile(target, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runLockdown(t, dir, true, true); err != nil {
		t.Fatalf("apply --replace errored: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if strings.Contains(string(got), "Bash(custom-tool:*)") {
		t.Error("--replace did not clobber the custom allow entry")
	}
	if !strings.Contains(string(got), "Bash(coily:*)") {
		t.Error("--replace did not write the canonical defaults")
	}
}

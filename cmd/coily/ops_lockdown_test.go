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
					&cli.BoolFlag{Name: "user"},
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

func runLockdownUser(t *testing.T, apply bool) error {
	t.Helper()
	root := &cli.Command{
		Name: "test-root",
		Commands: []*cli.Command{
			{
				Name: "lockdown",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "path", Value: "."},
					&cli.BoolFlag{Name: "local"},
					&cli.BoolFlag{Name: "apply"},
					&cli.BoolFlag{Name: "replace"},
					&cli.BoolFlag{Name: "recursive"},
					&cli.BoolFlag{Name: "user"},
					&cli.StringFlag{Name: "token"},
				},
				Action: lockdownAction,
			},
		},
	}
	args := []string{"test-root", "lockdown", "--user"}
	if apply {
		args = append(args, "--apply")
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

// TestLockdown_RecursiveSkipsExistingAndContinues pins coily#124: in
// recursive mode, --apply (without --replace) must skip a repo that already
// has a settings.json and keep stamping the rest, rather than erroring on the
// first existing file and aborting the whole recursion.
func TestLockdown_RecursiveSkipsExistingAndContinues(t *testing.T) {
	root := t.TempDir()
	stamped := filepath.Join(root, "already")
	fresh := filepath.Join(root, "fresh")
	for _, r := range []string{stamped, fresh} {
		if err := os.MkdirAll(filepath.Join(r, ".git"), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	// Pre-stamp one repo with a sentinel settings.json.
	existing := filepath.Join(stamped, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(existing), 0o750); err != nil {
		t.Fatal(err)
	}
	sentinel := `{"permissions":{}}`
	if err := os.WriteFile(existing, []byte(sentinel), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := runLockdownFlags(t, root, true, false, true); err != nil {
		t.Fatalf("recursive apply over a pre-stamped repo errored instead of skipping: %v", err)
	}

	// The pre-stamped repo is left untouched (skip, not clobber).
	got, _ := os.ReadFile(existing)
	if string(got) != sentinel {
		t.Errorf("pre-stamped repo was modified despite no --replace: %q", string(got))
	}
	// The fresh repo still got stamped: recursion continued past the skip.
	freshTarget := filepath.Join(fresh, ".claude", "settings.json")
	if _, err := os.Stat(freshTarget); err != nil {
		t.Errorf("recursion did not continue to fresh repo %s: %v", freshTarget, err)
	}
}

func TestLockdown_RecursiveNoReposErrors(t *testing.T) {
	root := t.TempDir()
	err := runLockdownFlags(t, root, false, false, true)
	if err == nil || !strings.Contains(err.Error(), "no git repos") {
		t.Fatalf("expected 'no git repos' error, got %v", err)
	}
}

// TestLockdown_RecursiveReassertsAncestorDeny exercises the
// recursion-root reassertion path. The canonical deny merges in,
// AND the shadowed Bash(gh issue *) allow gets pruned (cli-guard#26).
func TestLockdown_RecursiveReassertsAncestorDeny(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	parentSettings := filepath.Join(root, ".claude", "settings.local.json")
	original := `{
  "permissions": {
    "allow": ["Bash(gh issue *)", "Read(/Users/kai/.claude/**)"]
  }
}`
	if err := os.WriteFile(parentSettings, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(root, "child-repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := runLockdownFlags(t, root, true, false, true); err != nil {
		t.Fatalf("recursive apply errored: %v", err)
	}

	got, err := os.ReadFile(parentSettings)
	if err != nil {
		t.Fatalf("read parent settings: %v", err)
	}
	body := string(got)
	if strings.Contains(body, "Bash(gh issue *)") {
		t.Errorf("shadowed allow Bash(gh issue *) not pruned; got: %s", body)
	}
	if !strings.Contains(body, "Read(/Users/kai/.claude/**)") {
		t.Errorf("non-Bash allow was dropped; got: %s", body)
	}
	if !strings.Contains(body, "Bash(gh:*)") {
		t.Errorf("canonical deny not merged into ancestor; got: %s", body)
	}
}

// TestLockdown_UserApplyMergesAndPrunes pins coily#128: --user merges
// canonical denies into ~/.claude/settings.json and prunes shadowed
// allows, preserving non-permissions top-level keys.
func TestLockdown_UserApplyMergesAndPrunes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	settingsDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	settingsPath := filepath.Join(settingsDir, "settings.json")
	original := `{
  "permissions": {
    "allow": ["Bash(gh issue *)", "Bash(coily:*)", "Read(/Users/kai/.claude/**)"]
  },
  "hooks": {"Stop": []},
  "enabledPlugins": {"foo": true}
}`
	if err := os.WriteFile(settingsPath, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runLockdownUser(t, true); err != nil {
		t.Fatalf("--user --apply errored: %v", err)
	}
	got, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	body := string(got)
	if strings.Contains(body, "Bash(gh issue *)") {
		t.Errorf("shadowed allow Bash(gh issue *) not pruned; got: %s", body)
	}
	if !strings.Contains(body, "Bash(coily:*)") {
		t.Errorf("non-shadowed allow Bash(coily:*) was dropped; got: %s", body)
	}
	if !strings.Contains(body, "Read(/Users/kai/.claude/**)") {
		t.Errorf("non-Bash allow was dropped; got: %s", body)
	}
	if !strings.Contains(body, "Bash(gh:*)") {
		t.Errorf("canonical deny not merged; got: %s", body)
	}
	if !strings.Contains(body, `"enabledPlugins"`) {
		t.Errorf("top-level enabledPlugins key dropped; got: %s", body)
	}
	if !strings.Contains(body, `"hooks"`) {
		t.Errorf("top-level hooks key dropped; got: %s", body)
	}
}

// TestLockdown_UserDryRunDoesNotTouch proves --user without --apply is a no-op write.
func TestLockdown_UserDryRunDoesNotTouch(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	settingsDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	settingsPath := filepath.Join(settingsDir, "settings.json")
	original := `{"permissions":{"allow":["Bash(gh issue *)"]}}`
	if err := os.WriteFile(settingsPath, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runLockdownUser(t, false); err != nil {
		t.Fatalf("dry-run errored: %v", err)
	}
	got, _ := os.ReadFile(settingsPath)
	if string(got) != original {
		t.Errorf("dry-run mutated user settings file: got %q", string(got))
	}
}

// TestLockdown_RecursiveDryRunDoesNotTouchAncestor proves the
// reassertion is gated on --apply.
func TestLockdown_RecursiveDryRunDoesNotTouchAncestor(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	parentSettings := filepath.Join(root, ".claude", "settings.local.json")
	original := `{"permissions":{"allow":["Bash(gh issue *)"]}}`
	if err := os.WriteFile(parentSettings, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(root, "child-repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := runLockdownFlags(t, root, false, false, true); err != nil {
		t.Fatalf("dry-run errored: %v", err)
	}
	got, _ := os.ReadFile(parentSettings)
	if string(got) != original {
		t.Errorf("dry-run mutated ancestor settings file: got %q", string(got))
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

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/urfave/cli/v3"
)

// TestInstallSkillSymlinks_LinksAllStaged pins issue #65: every coily-*
// directory under <prefix>/share/coily/skills/ becomes a symlink at
// ~/.claude/skills/<basename>. The previous shape symlinked one entry;
// the loop is what the brew-shipped skill family needs.
func TestInstallSkillSymlinks_LinksAllStaged(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	prefix := filepath.Join(tmp, "brew")
	binDir := filepath.Join(prefix, "bin")
	stagedRoot := filepath.Join(prefix, "share", "coily", "skills")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	if err := os.MkdirAll(stagedRoot, 0o755); err != nil {
		t.Fatalf("mkdir staged: %v", err)
	}
	for _, name := range []string{"coily-ops-aws-meta", "coily-git-usage", "coily-passthroughs"} {
		if err := os.MkdirAll(filepath.Join(stagedRoot, name), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
	}
	// Non-coily-* directory must be ignored.
	if err := os.MkdirAll(filepath.Join(stagedRoot, "unrelated"), 0o755); err != nil {
		t.Fatalf("mkdir unrelated: %v", err)
	}

	self := filepath.Join(binDir, "coily")
	if err := os.WriteFile(self, []byte("placeholder"), 0o755); err != nil {
		t.Fatalf("write self: %v", err)
	}

	if err := installSkillSymlinks(self); err != nil {
		t.Fatalf("installSkillSymlinks: %v", err)
	}

	for _, name := range []string{"coily-ops-aws-meta", "coily-git-usage", "coily-passthroughs"} {
		link := filepath.Join(tmp, ".claude", "skills", name)
		target, err := os.Readlink(link)
		if err != nil {
			t.Errorf("readlink %s: %v", link, err)
			continue
		}
		want, _ := filepath.EvalSymlinks(filepath.Join(stagedRoot, name))
		got, _ := filepath.EvalSymlinks(target)
		if got != want {
			t.Errorf("%s -> %s, want %s", link, got, want)
		}
	}
	if _, err := os.Lstat(filepath.Join(tmp, ".claude", "skills", "unrelated")); !os.IsNotExist(err) {
		t.Errorf("unrelated dir was symlinked; should have been ignored")
	}
}

// TestInstallSkillSymlinks_ReplacesStaleTarget pins idempotent re-runs:
// an existing symlink pointing at a stale target is replaced; one
// pointing at the right target is left alone.
func TestInstallSkillSymlinks_ReplacesStaleTarget(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	prefix := filepath.Join(tmp, "brew")
	binDir := filepath.Join(prefix, "bin")
	stagedRoot := filepath.Join(prefix, "share", "coily", "skills")
	for _, d := range []string{binDir, filepath.Join(stagedRoot, "coily-git-meta")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	self := filepath.Join(binDir, "coily")
	if err := os.WriteFile(self, []byte("x"), 0o755); err != nil {
		t.Fatalf("write self: %v", err)
	}

	skillsDir := filepath.Join(tmp, ".claude", "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatalf("mkdir skills: %v", err)
	}
	stale := filepath.Join(tmp, "stale")
	if err := os.MkdirAll(stale, 0o755); err != nil {
		t.Fatalf("mkdir stale: %v", err)
	}
	if err := os.Symlink(stale, filepath.Join(skillsDir, "coily-git-meta")); err != nil {
		t.Fatalf("symlink stale: %v", err)
	}

	if err := installSkillSymlinks(self); err != nil {
		t.Fatalf("installSkillSymlinks: %v", err)
	}
	target, err := os.Readlink(filepath.Join(skillsDir, "coily-git-meta"))
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	got, _ := filepath.EvalSymlinks(target)
	want, _ := filepath.EvalSymlinks(filepath.Join(stagedRoot, "coily-git-meta"))
	if got != want {
		t.Errorf("stale symlink not replaced; -> %s, want %s", got, want)
	}
}

// TestInstallSkillSymlinks_DropsLegacyCoilySymlink pins the migration
// step: the old single ~/.claude/skills/coily symlink (pre-2026-05-05
// skill family) is removed when it points at the legacy
// coily-passthroughs target.
func TestInstallSkillSymlinks_DropsLegacyCoilySymlink(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	prefix := filepath.Join(tmp, "brew")
	binDir := filepath.Join(prefix, "bin")
	stagedRoot := filepath.Join(prefix, "share", "coily", "skills")
	stagedPassthrough := filepath.Join(stagedRoot, "coily-passthroughs")
	for _, d := range []string{binDir, stagedPassthrough} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	self := filepath.Join(binDir, "coily")
	if err := os.WriteFile(self, []byte("x"), 0o755); err != nil {
		t.Fatalf("write self: %v", err)
	}

	skillsDir := filepath.Join(tmp, ".claude", "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatalf("mkdir skills: %v", err)
	}
	legacy := filepath.Join(skillsDir, "coily")
	if err := os.Symlink(stagedPassthrough, legacy); err != nil {
		t.Fatalf("symlink legacy: %v", err)
	}

	if err := installSkillSymlinks(self); err != nil {
		t.Fatalf("installSkillSymlinks: %v", err)
	}
	if _, err := os.Lstat(legacy); !os.IsNotExist(err) {
		t.Errorf("legacy ~/.claude/skills/coily symlink not removed")
	}
}

// TestInstallSkillSymlinks_NoStagedDirIsNotAnError covers the dev-build
// case where coily was built from source and brew never ran. setup
// should print a skip line and return nil.
func TestInstallSkillSymlinks_NoStagedDirIsNotAnError(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	binDir := filepath.Join(tmp, "go", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	self := filepath.Join(binDir, "coily")
	if err := os.WriteFile(self, []byte("x"), 0o755); err != nil {
		t.Fatalf("write self: %v", err)
	}
	if err := installSkillSymlinks(self); err != nil {
		t.Errorf("installSkillSymlinks on dev build returned %v, want nil", err)
	}
}

// TestSetupCommand_WorkspaceFlagBindsEnvVar pins issue #120: the workspace
// flag must resolve from COILY_LOCKDOWN_ROOT when no --workspace is passed,
// so brew users can set the env var once in their shell profile and have
// `coily setup` after each brew upgrade target the right tree without
// remembering the flag.
func TestSetupCommand_WorkspaceFlagBindsEnvVar(t *testing.T) {
	r := NewRunner()
	cmd := r.setupCommand()
	var workspace *cli.StringFlag
	for _, f := range cmd.Flags {
		sf, ok := f.(*cli.StringFlag)
		if ok && sf.Name == "workspace" {
			workspace = sf
			break
		}
	}
	if workspace == nil {
		t.Fatal("setup command has no --workspace flag")
	}
	got := workspace.Sources.String()
	if got == "" || !contains(got, "COILY_LOCKDOWN_ROOT") {
		t.Errorf("workspace.Sources = %q, want it to reference COILY_LOCKDOWN_ROOT", got)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

// TestSetupCommand_LockdownRootFlagBindsEnvVar pins issue #120: the
// --lockdown-root flag (renamed from --workspace per #121) must resolve
// from COILY_LOCKDOWN_ROOT when not passed, so brew users can set the
// env var once in their shell profile and have `coily setup` after each
// brew upgrade target the right tree without remembering the flag.
func TestSetupCommand_LockdownRootFlagBindsEnvVar(t *testing.T) {
	r := NewRunner()
	cmd := r.setupCommand()
	var lockdownRoot *cli.StringFlag
	for _, f := range cmd.Flags {
		sf, ok := f.(*cli.StringFlag)
		if ok && sf.Name == "lockdown-root" {
			lockdownRoot = sf
			break
		}
	}
	if lockdownRoot == nil {
		t.Fatal("setup command has no --lockdown-root flag")
	}
	got := lockdownRoot.Sources.String()
	if got == "" || !contains(got, "COILY_LOCKDOWN_ROOT") {
		t.Errorf("lockdown-root.Sources = %q, want it to reference COILY_LOCKDOWN_ROOT", got)
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

// TestUserHookCleanup_AllThreeStates covers coily#247: the runUserHookStep
// cleanup must handle (a) pre-#185 (file + hook entry both present), (b)
// post-#185 clean (neither), (c) partial. All paths preserve unrelated
// settings.json state (env.PATH, theme, other hooks).
func TestUserHookCleanup_AllThreeStates(t *testing.T) {
	makeSettings := func() map[string]any {
		return map[string]any{
			"env": map[string]any{"PATH": "/opt/homebrew/bin:/usr/bin"},
			"hooks": map[string]any{
				"PreToolUse": []any{
					map[string]any{
						"matcher": "Bash",
						"hooks": []any{
							map[string]any{
								"type":    "command",
								"command": "$HOME/.claude/coily-binary-gate.sh",
							},
						},
					},
					map[string]any{
						"matcher": "Edit",
						"hooks": []any{
							map[string]any{"type": "command", "command": "other-hook"},
						},
					},
				},
			},
		}
	}

	writeJSON := func(t *testing.T, path string, v any) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	readJSON := func(t *testing.T, path string) map[string]any {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		var v map[string]any
		if err := json.Unmarshal(data, &v); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		return v
	}

	t.Run("pre-185-both-present", func(t *testing.T) {
		home := t.TempDir()
		scriptPath := filepath.Join(home, ".claude", "coily-binary-gate.sh")
		settingsPath := filepath.Join(home, ".claude", "settings.json")
		_ = os.MkdirAll(filepath.Dir(scriptPath), 0o755)
		_ = os.WriteFile(scriptPath, []byte("legacy"), 0o755)
		writeJSON(t, settingsPath, makeSettings())
		clean := userHookCleanup{ScriptPath: scriptPath, SettingsPath: settingsPath}
		if err := clean.run(os.Stderr); err != nil {
			t.Fatalf("run: %v", err)
		}
		if _, err := os.Stat(scriptPath); !os.IsNotExist(err) {
			t.Errorf("script file still present: %v", err)
		}
		got := readJSON(t, settingsPath)
		env, _ := got["env"].(map[string]any)
		if env["PATH"] != "/opt/homebrew/bin:/usr/bin" {
			t.Errorf("env.PATH clobbered: got %v", env)
		}
		hooks := got["hooks"].(map[string]any)
		pre := hooks["PreToolUse"].([]any)
		if len(pre) != 1 {
			t.Fatalf("PreToolUse length = %d, want 1 (Edit should remain)", len(pre))
		}
		if pre[0].(map[string]any)["matcher"].(string) != "Edit" {
			t.Errorf("surviving matcher = %v, want Edit", pre[0])
		}
	})

	t.Run("post-185-clean", func(t *testing.T) {
		home := t.TempDir()
		clean := userHookCleanup{
			ScriptPath:   filepath.Join(home, ".claude", "coily-binary-gate.sh"),
			SettingsPath: filepath.Join(home, ".claude", "settings.json"),
		}
		if err := clean.run(os.Stderr); err != nil {
			t.Fatalf("run: %v", err)
		}
	})

	t.Run("partial-file-only", func(t *testing.T) {
		home := t.TempDir()
		scriptPath := filepath.Join(home, ".claude", "coily-binary-gate.sh")
		_ = os.MkdirAll(filepath.Dir(scriptPath), 0o755)
		_ = os.WriteFile(scriptPath, []byte("legacy"), 0o755)
		clean := userHookCleanup{
			ScriptPath:   scriptPath,
			SettingsPath: filepath.Join(home, ".claude", "settings.json"),
		}
		if err := clean.run(os.Stderr); err != nil {
			t.Fatalf("run: %v", err)
		}
		if _, err := os.Stat(scriptPath); !os.IsNotExist(err) {
			t.Errorf("script file still present: %v", err)
		}
	})

	t.Run("partial-settings-only", func(t *testing.T) {
		home := t.TempDir()
		settingsPath := filepath.Join(home, ".claude", "settings.json")
		writeJSON(t, settingsPath, makeSettings())
		clean := userHookCleanup{
			ScriptPath:   filepath.Join(home, ".claude", "coily-binary-gate.sh"),
			SettingsPath: settingsPath,
		}
		if err := clean.run(os.Stderr); err != nil {
			t.Fatalf("run: %v", err)
		}
		got := readJSON(t, settingsPath)
		hooks := got["hooks"].(map[string]any)
		pre := hooks["PreToolUse"].([]any)
		if len(pre) != 1 || pre[0].(map[string]any)["matcher"].(string) != "Edit" {
			t.Errorf("PreToolUse after partial cleanup: %v", pre)
		}
	})

	t.Run("drops-pretooluse-when-gate-is-only-entry", func(t *testing.T) {
		home := t.TempDir()
		settingsPath := filepath.Join(home, ".claude", "settings.json")
		writeJSON(t, settingsPath, map[string]any{
			"theme": "dark",
			"hooks": map[string]any{
				"PreToolUse": []any{
					map[string]any{
						"matcher": "Bash",
						"hooks": []any{
							map[string]any{"type": "command", "command": "coily-binary-gate.sh"},
						},
					},
				},
			},
		})
		clean := userHookCleanup{
			ScriptPath:   filepath.Join(home, ".claude", "coily-binary-gate.sh"),
			SettingsPath: settingsPath,
		}
		if err := clean.run(os.Stderr); err != nil {
			t.Fatalf("run: %v", err)
		}
		got := readJSON(t, settingsPath)
		if got["theme"] != "dark" {
			t.Errorf("theme clobbered: got %v", got["theme"])
		}
		hooks, _ := got["hooks"].(map[string]any)
		if pre, ok := hooks["PreToolUse"]; ok {
			t.Errorf("PreToolUse should be absent after dropping only entry, got %v", pre)
		}
	})
}

// TestRunHostBootstrapStep_SkipsWhenBrewfileMissing pins coilysiren/coily#264:
// no Brewfile under <lockdown-root>/agentic-os/brew means the step prints a
// skip and returns nil. Same shape as runLockdownStep on a missing root.
func TestRunHostBootstrapStep_SkipsWhenBrewfileMissing(t *testing.T) {
	root := t.TempDir()
	self := filepath.Join(root, "coily")
	if err := os.WriteFile(self, []byte("#!/bin/sh\nexit 99\n"), 0o755); err != nil {
		t.Fatalf("write self: %v", err)
	}
	if err := runHostBootstrapStep(context.Background(), self, root); err != nil {
		t.Fatalf("runHostBootstrapStep: %v", err)
	}
}

// TestRunHostBootstrapStep_BrewBundleFailureIsWarning pins coilysiren/coily#275:
// a `brew bundle install` non-zero exit (e.g. a single dep collision on
// /opt/homebrew/bin/<name>) must surface as a warning and let the rest of
// setup continue. The step also still invokes the uv tool install step.
func TestRunHostBootstrapStep_BrewBundleFailureIsWarning(t *testing.T) {
	root := t.TempDir()
	brewDir := filepath.Join(root, "agentic-os", "brew")
	if err := os.MkdirAll(brewDir, 0o755); err != nil {
		t.Fatalf("mkdir brew: %v", err)
	}
	if err := os.WriteFile(filepath.Join(brewDir, "Brewfile"), []byte("brew \"jq\"\n"), 0o644); err != nil {
		t.Fatalf("write Brewfile: %v", err)
	}

	// Fake self: exits non-zero for the `brew bundle install` invocation,
	// success for everything else (the uv tool install).
	log := filepath.Join(root, "invocations.log")
	script := "#!/bin/sh\n" +
		"for a in \"$@\"; do echo \"ARG=$a\" >> \"" + log + "\"; done\n" +
		"echo --- >> \"" + log + "\"\n" +
		"case \"$2\" in brew) exit 3 ;; esac\n" +
		"exit 0\n"
	self := filepath.Join(root, "coily")
	if err := os.WriteFile(self, []byte(script), 0o755); err != nil {
		t.Fatalf("write self: %v", err)
	}

	if err := runHostBootstrapStep(context.Background(), self, root); err != nil {
		t.Fatalf("runHostBootstrapStep should warn, not fail: %v", err)
	}

	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "ARG=brew") {
		t.Errorf("expected brew bundle invocation, got:\n%s", got)
	}
	if !strings.Contains(got, "ARG=uv") {
		t.Errorf("expected uv tool install invocation to run despite brew failure, got:\n%s", got)
	}
}

// TestRunHostBootstrapStep_InvokesCoilyPkgInAgenticOSDir pins coilysiren/coily#264:
// when the Brewfile exists, runHostBootstrapStep self-execs
// `coily pkg brew bundle install --file <brewfile>` and
// `coily pkg uv tool install pre-commit --with pre-commit-uv`, both with
// cmd.Dir set to the agentic-os checkout.
func TestRunHostBootstrapStep_InvokesCoilyPkgInAgenticOSDir(t *testing.T) {
	root := t.TempDir()
	brewDir := filepath.Join(root, "agentic-os", "brew")
	if err := os.MkdirAll(brewDir, 0o755); err != nil {
		t.Fatalf("mkdir brew: %v", err)
	}
	if err := os.WriteFile(filepath.Join(brewDir, "Brewfile"), []byte("brew \"jq\"\n"), 0o644); err != nil {
		t.Fatalf("write Brewfile: %v", err)
	}

	// Fake self records argv and cwd of each invocation. One line per arg,
	// CWD marker per invocation, --- separator between invocations.
	log := filepath.Join(root, "invocations.log")
	script := "#!/bin/sh\n" +
		"echo \"CWD=$PWD\" >> \"" + log + "\"\n" +
		"for a in \"$@\"; do echo \"ARG=$a\" >> \"" + log + "\"; done\n" +
		"echo --- >> \"" + log + "\"\n"
	self := filepath.Join(root, "coily")
	if err := os.WriteFile(self, []byte(script), 0o755); err != nil {
		t.Fatalf("write self: %v", err)
	}

	if err := runHostBootstrapStep(context.Background(), self, root); err != nil {
		t.Fatalf("runHostBootstrapStep: %v", err)
	}

	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	got := string(data)
	agenticOS := filepath.Join(root, "agentic-os")
	brewfile := filepath.Join(brewDir, "Brewfile")
	wants := []string{
		"CWD=" + agenticOS,
		"ARG=pkg", "ARG=brew", "ARG=bundle", "ARG=install", "ARG=--file", "ARG=" + brewfile,
		"ARG=uv", "ARG=tool", "ARG=pre-commit", "ARG=--with", "ARG=pre-commit-uv",
	}
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Errorf("log missing %q\nlog:\n%s", w, got)
		}
	}
	if strings.Count(got, "CWD="+agenticOS) != 2 {
		t.Errorf("expected 2 invocations with CWD=%s, got log:\n%s", agenticOS, got)
	}
}

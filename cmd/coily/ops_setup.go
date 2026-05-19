package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// setupCommand runs the post-upgrade rituals brew's sandbox blocks from
// post_install: tab-completion, skill symlinks, lockdown re-baseline, and
// the user-level PreToolUse hook. Idempotent; safe to run any time.
//
// The skill-symlink step (issue #65) walks every coily-* directory the
// brew formula stages under <prefix>/share/coily/skills/ and links each
// into ~/.claude/skills/. This keeps every authored skill visible to the
// harness on a fresh brew install, not just one. Friends installing coily
// via brew get the full set without needing a agentic-os-kai checkout.
func (r *Runner) setupCommand() *cli.Command {
	return &cli.Command{
		Name:  "setup",
		Usage: "Run the post-upgrade rituals: completion, lockdown re-baseline, and user hook.",
		Description: `setup runs four idempotent steps in order:

  1. coily install-completion         (refresh shell tab-completion)
  2. <prefix>/share/coily/skills/*    (symlink every staged coily-* skill
                                       into ~/.claude/skills/ so the harness
                                       picks them up)
  3. coily lockdown --recursive ...   (re-baseline allow/deny lists under the lockdown root)
  4. ~/.claude/coily-binary-gate.sh   (user-level PreToolUse hook that
                                       rejects dev coily binaries from any
                                       cwd; complements per-repo lockdown)

Pass --lockdown-root or set $COILY_LOCKDOWN_ROOT to override the lockdown root
(default: ~/projects/coilysiren). Skips the lockdown step if the root
does not exist, which keeps fresh brew installs on hosts without the default
tree (friends' machines, alternate layouts) silent.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "lockdown-root",
				Usage:   "directory to scan recursively for git repos to lock down. Read from $COILY_LOCKDOWN_ROOT if unset.",
				Value:   "",
				Sources: cli.EnvVars("COILY_LOCKDOWN_ROOT"),
			},
			&cli.BoolFlag{
				Name:  "skip-completion",
				Usage: "skip the install-completion step",
			},
			&cli.BoolFlag{
				Name:  "skip-lockdown",
				Usage: "skip the lockdown re-baseline step",
			},
			&cli.BoolFlag{
				Name:  "skip-skills",
				Usage: "skip the brew-staged coily-* skill symlink step",
			},
			&cli.BoolFlag{
				Name:  "skip-user-hook",
				Usage: "skip the user-level PreToolUse hook install",
			},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name:      "setup",
				SkipScope: true,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{"--lockdown-root": c.String("lockdown-root")}, nil
				},
				Action: setupAction,
			},
			r.Audit,
		),
	}
}

func setupAction(ctx context.Context, c *cli.Command) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("setup: locate self: %w", err)
	}

	if !c.Bool("skip-completion") {
		fmt.Fprintln(os.Stderr, "==> install-completion")
		cmd := exec.CommandContext(ctx, self, "install-completion")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("setup: install-completion: %w", err)
		}
	}

	if !c.Bool("skip-skills") {
		fmt.Fprintln(os.Stderr, "==> skills")
		if err := installSkillSymlinks(self); err != nil {
			return err
		}
	}

	if !c.Bool("skip-lockdown") {
		fmt.Fprintln(os.Stderr, "==> lockdown")
		if err := runLockdownStep(ctx, self, c.String("lockdown-root")); err != nil {
			return err
		}
	}

	if !c.Bool("skip-user-hook") {
		fmt.Fprintln(os.Stderr, "==> user hook")
		if err := runUserHookStep(); err != nil {
			return err
		}
	}

	fmt.Fprintln(os.Stderr, "setup: done")
	return nil
}

// runUserHookStep cleans up the pre-#185 user-wide
// ~/.claude/coily-binary-gate.sh hook. coily#185 moved the binary-path
// check into `agent-guard hook pre-tool-use`, fired from the per-repo
// hook coily writes via the new shim, but the legacy artifact stayed
// on disk on hosts that ran `coily setup` before #185 shipped. This
// step deletes the script and strips the matching PreToolUse entry
// from ~/.claude/settings.json, leaving every other setting (env.PATH,
// theme, permissions, other hooks) untouched. Idempotent: re-running
// on a clean host is a no-op. Per coilysiren/coily#247.
//
// TODO: drop runUserHookStep and the --skip-user-hook flag in the
// release after the migration window closes.
func runUserHookStep() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("user-hook cleanup: home dir: %w", err)
	}
	clean := userHookCleanup{
		ScriptPath:   filepath.Join(home, ".claude", "coily-binary-gate.sh"),
		SettingsPath: filepath.Join(home, ".claude", "settings.json"),
	}
	return clean.run(os.Stderr)
}

type userHookCleanup struct {
	ScriptPath   string
	SettingsPath string
}

func (c userHookCleanup) run(w *os.File) error {
	scriptRemoved, err := c.removeScript()
	if err != nil {
		return err
	}
	entryRemoved, err := c.stripSettingsEntry()
	if err != nil {
		return err
	}
	switch {
	case scriptRemoved && entryRemoved:
		_, _ = fmt.Fprintln(w, "    cleaned: ~/.claude/coily-binary-gate.sh + settings.json PreToolUse entry")
	case scriptRemoved:
		_, _ = fmt.Fprintln(w, "    cleaned: ~/.claude/coily-binary-gate.sh (settings.json had no matching entry)")
	case entryRemoved:
		_, _ = fmt.Fprintln(w, "    cleaned: settings.json PreToolUse entry (script file already absent)")
	default:
		_, _ = fmt.Fprintln(w, "    nothing to clean (legacy user-wide hook is already gone)")
	}
	return nil
}

func (c userHookCleanup) removeScript() (bool, error) {
	if err := os.Remove(c.ScriptPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("user-hook cleanup: remove %s: %w", c.ScriptPath, err)
	}
	return true, nil
}

// stripSettingsEntry rewrites ~/.claude/settings.json to drop any
// hooks.PreToolUse[*].hooks[*] entry whose command references
// coily-binary-gate.sh. If an enclosing matcher object's hooks array
// becomes empty, the matcher object itself is dropped. All other state
// (env.PATH, theme, permissions, unrelated hooks) is preserved.
// Returns true when at least one entry was removed.
func (c userHookCleanup) stripSettingsEntry() (bool, error) {
	data, err := os.ReadFile(c.SettingsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("user-hook cleanup: read %s: %w", c.SettingsPath, err)
	}
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return false, fmt.Errorf("user-hook cleanup: parse %s: %w", c.SettingsPath, err)
	}
	removed := stripCoilyBinaryGate(root)
	if !removed {
		return false, nil
	}
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return false, fmt.Errorf("user-hook cleanup: marshal: %w", err)
	}
	out = append(out, '\n')
	if err := os.WriteFile(c.SettingsPath, out, 0o600); err != nil {
		return false, fmt.Errorf("user-hook cleanup: write %s: %w", c.SettingsPath, err)
	}
	return true, nil
}

// stripCoilyBinaryGate walks hooks.PreToolUse and prunes hook entries
// whose command references coily-binary-gate.sh. Returns true when the
// tree was modified. Pure on input shape: structural in-place edit so
// the caller can decide whether to write.
func stripCoilyBinaryGate(root map[string]any) bool {
	hooks, _ := root["hooks"].(map[string]any)
	if hooks == nil {
		return false
	}
	preToolUse, _ := hooks["PreToolUse"].([]any)
	if preToolUse == nil {
		return false
	}
	var changed bool
	kept := make([]any, 0, len(preToolUse))
	for _, entryAny := range preToolUse {
		entry, ok := entryAny.(map[string]any)
		if !ok {
			kept = append(kept, entryAny)
			continue
		}
		inner, _ := entry["hooks"].([]any)
		if inner == nil {
			kept = append(kept, entryAny)
			continue
		}
		filtered := make([]any, 0, len(inner))
		for _, hAny := range inner {
			h, ok := hAny.(map[string]any)
			if !ok {
				filtered = append(filtered, hAny)
				continue
			}
			cmd, _ := h["command"].(string)
			if strings.Contains(cmd, "coily-binary-gate.sh") {
				changed = true
				continue
			}
			filtered = append(filtered, hAny)
		}
		if len(filtered) == 0 {
			changed = true
			continue
		}
		entry["hooks"] = filtered
		kept = append(kept, entry)
	}
	if !changed {
		return false
	}
	if len(kept) == 0 {
		delete(hooks, "PreToolUse")
	} else {
		hooks["PreToolUse"] = kept
	}
	return true
}

// installSkillSymlinks walks every coily-* directory the brew formula
// stages under <bin>/../share/coily/skills/ and symlinks each one into
// ~/.claude/skills/<basename> so the Claude Code harness picks them up.
// Idempotent: an existing symlink pointing at the right target is left
// alone, an existing symlink pointing at a stale target is replaced, and
// a regular file or directory at the destination is left alone with a
// warning rather than clobbered. Issue #65.
//
// Migration: drops the legacy single ~/.claude/skills/coily symlink if it
// still points at the old coily-passthroughs target. The skills loop
// supersedes that single-symlink shape from before the 2026-05-05 skill
// family landed.
func installSkillSymlinks(self string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("setup: home dir: %w", err)
	}
	dest := filepath.Join(home, ".claude", "skills")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("setup: mkdir %s: %w", dest, err)
	}

	// Resolve any symlinks in the binary path so we land at the real
	// homebrew Cellar location, then walk up to the prefix's share dir.
	realSelf, err := filepath.EvalSymlinks(self)
	if err != nil {
		realSelf = self
	}
	stagedRoot := filepath.Join(filepath.Dir(realSelf), "..", "share", "coily", "skills")
	stagedRoot = filepath.Clean(stagedRoot)
	entries, err := os.ReadDir(stagedRoot)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "    skipped: no staged skills at %s (dev build?)\n", stagedRoot)
			return nil
		}
		return fmt.Errorf("setup: read %s: %w", stagedRoot, err)
	}

	migrateLegacySkillSymlink(dest, stagedRoot)

	linked, replaced, skipped := linkSkillEntries(dest, stagedRoot, entries)
	fmt.Fprintf(os.Stderr, "    %d new, %d replaced, %d unchanged\n", linked, replaced, skipped)
	return nil
}

// linkSkillEntries iterates entries from the brew-staged dir and
// idempotently symlinks each coily-* into dest. Returns (linked, replaced,
// skipped) counts. Pulled out of installSkillSymlinks to keep that
// function under the gocyclo threshold.
func linkSkillEntries(dest, stagedRoot string, entries []os.DirEntry) (int, int, int) {
	linked, replaced, skipped := 0, 0, 0
	for _, entry := range entries {
		if !entry.IsDir() && entry.Type()&os.ModeSymlink == 0 {
			continue
		}
		name := entry.Name()
		if !filepathHasPrefix(name, "coily-") {
			continue
		}
		action, err := ensureSymlink(filepath.Join(dest, name), filepath.Join(stagedRoot, name))
		if err != nil {
			fmt.Fprintf(os.Stderr, "    %s: %v\n", name, err)
			skipped++
			continue
		}
		switch action {
		case "linked":
			linked++
		case "replaced":
			replaced++
		default:
			skipped++
		}
	}
	return linked, replaced, skipped
}

// ensureSymlink creates link -> src, replacing an existing wrong-target
// symlink in place. Returns "linked" (new), "replaced" (stale symlink
// fixed), or "unchanged" (already correct or a non-symlink we won't
// clobber). Errors only on filesystem operations that should not have
// failed at all (mkdir parent, readlink unrelated to ENOENT).
func ensureSymlink(link, src string) (string, error) {
	info, err := os.Lstat(link)
	if os.IsNotExist(err) {
		return "linked", os.Symlink(src, link)
	}
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return "unchanged", fmt.Errorf("destination is not a symlink, leaving alone")
	}
	current, err := os.Readlink(link)
	if err != nil {
		return "", err
	}
	if current == src {
		return "unchanged", nil
	}
	if err := os.Remove(link); err != nil {
		return "", err
	}
	return "replaced", os.Symlink(src, link)
}

// migrateLegacySkillSymlink drops ~/.claude/skills/coily if it points at
// the legacy single-skill target (coily-passthroughs). The pre-2026-05-05
// shape symlinked exactly one skill at this path; the per-skill loop now
// supersedes it. Quiet on absence; harmless if the symlink already points
// at something the user added intentionally.
func migrateLegacySkillSymlink(dest, _ string) {
	legacy := filepath.Join(dest, "coily")
	info, err := os.Lstat(legacy)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return
	}
	target, err := os.Readlink(legacy)
	if err != nil {
		return
	}
	if filepath.Base(target) != "coily-passthroughs" {
		return
	}
	_ = os.Remove(legacy)
	fmt.Fprintf(os.Stderr, "    migrated: removed legacy ~/.claude/skills/coily\n")
}

// filepathHasPrefix is strings.HasPrefix scoped to filenames so the intent
// reads as "is this a coily-* skill name" without dragging in strings just
// for one call.
func filepathHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func runLockdownStep(ctx context.Context, self, lockdownRoot string) error {
	if lockdownRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("setup: home dir: %w", err)
		}
		lockdownRoot = filepath.Join(home, "projects", "coilysiren")
	}
	info, err := os.Stat(lockdownRoot)
	if os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "    skipped: %s does not exist\n", lockdownRoot)
		return nil
	}
	if err != nil {
		return fmt.Errorf("setup: stat lockdown root: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("setup: lockdown root %s is not a directory", lockdownRoot)
	}
	cmd := exec.CommandContext(ctx, self, "lockdown",
		"--recursive", "--apply", "--replace", "--path", lockdownRoot)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("setup: lockdown: %w", err)
	}
	return nil
}

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/coilysiren/cli-guard/verb"
	"github.com/coilysiren/coily/pkg/lockdown"
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
// via brew get the full set without needing a coilyco-ai checkout.
func (r *Runner) setupCommand() *cli.Command {
	return &cli.Command{
		Name:  "setup",
		Usage: "Run the post-upgrade rituals: completion, lockdown re-baseline, and user hook.",
		Description: `setup runs four idempotent steps in order:

  1. coily install-completion         (refresh shell tab-completion)
  2. <prefix>/share/coily/skills/*    (symlink every staged coily-* skill
                                       into ~/.claude/skills/ so the harness
                                       picks them up)
  3. coily lockdown --recursive ...   (re-baseline workspace allow/deny lists)
  4. ~/.claude/coily-binary-gate.sh   (user-level PreToolUse hook that
                                       rejects dev coily binaries from any
                                       cwd; complements per-repo lockdown)

Pass --workspace or set $COILY_LOCKDOWN_ROOT to override the lockdown root
(default: ~/projects/coilysiren). Skips the lockdown step if the workspace
does not exist, which keeps fresh brew installs on hosts without the default
tree (friends' machines, alternate layouts) silent.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "workspace",
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
		Action: verb.Wrap(
			verb.Spec{
				Name:      "setup",
				SkipScope: true,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{"--workspace": c.String("workspace")}, nil
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
		if err := runLockdownStep(ctx, self, c.String("workspace")); err != nil {
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

// runUserHookStep installs ~/.claude/coily-binary-gate.sh and patches
// ~/.claude/settings.json to invoke it via PreToolUse. The gate rejects
// any coily invocation that doesn't resolve to a homebrew install path,
// catching dev binaries built from source even when invoked from a cwd
// that has no per-repo lockdown hook.
func runUserHookStep() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("setup: home dir: %w", err)
	}
	hookPath, changed, err := lockdown.EnsureUserHook(home)
	if err != nil {
		return fmt.Errorf("setup: user hook: %w", err)
	}
	verb := "unchanged"
	if changed {
		verb = "updated"
	}
	fmt.Fprintf(os.Stderr, "    %s (settings.json %s)\n", hookPath, verb)
	return nil
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

func runLockdownStep(ctx context.Context, self, workspace string) error {
	if workspace == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("setup: home dir: %w", err)
		}
		workspace = filepath.Join(home, "projects", "coilysiren")
	}
	info, err := os.Stat(workspace)
	if os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "    skipped: %s does not exist\n", workspace)
		return nil
	}
	if err != nil {
		return fmt.Errorf("setup: stat workspace: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("setup: workspace %s is not a directory", workspace)
	}
	cmd := exec.CommandContext(ctx, self, "lockdown",
		"--recursive", "--apply", "--replace", "--path", workspace)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("setup: lockdown: %w", err)
	}
	return nil
}

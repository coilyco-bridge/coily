package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

// setupCommand runs the post-upgrade rituals brew's sandbox blocks from
// post_install: tab-completion, the skill symlink, and a workspace-wide
// lockdown re-baseline. Idempotent; safe to run any time.
//
// The skill symlink target is hardcoded to <coily-binary>/../share/coily/skill,
// which is where the brew formula stages skill/. That path is not user-supplied,
// so an agent inside lockdown that runs `coily setup` cannot redirect the
// skill to an attacker-chosen location - the threat model that keeps
// install-skill out of prod is preserved.
func (r *Runner) setupCommand() *cli.Command {
	return &cli.Command{
		Name:  "setup",
		Usage: "Run the post-upgrade rituals: completion, skill symlink, and lockdown re-baseline.",
		Description: `setup runs three idempotent steps in order:

  1. coily install-completion         (refresh shell tab-completion)
  2. symlink ~/.claude/skills/coily   (point at the brew-managed skill dir)
  3. coily lockdown --recursive ...   (re-baseline workspace allow/deny lists)

The skill symlink target is derived from the running coily binary's location
(<bin>/../share/coily/skill). It is not user-configurable, by design.

Pass --workspace to override the lockdown root (default: ~/projects/coilysiren).
Skips the lockdown step if the workspace does not exist.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "workspace",
				Usage: "directory to scan recursively for git repos to lock down",
				Value: "",
			},
			&cli.BoolFlag{
				Name:  "skip-completion",
				Usage: "skip the install-completion step",
			},
			&cli.BoolFlag{
				Name:  "skip-skill",
				Usage: "skip the skill-symlink step",
			},
			&cli.BoolFlag{
				Name:  "skip-lockdown",
				Usage: "skip the lockdown re-baseline step",
			},
		},
		Action: verb.Wrap(
			verb.Spec{
				Name: "setup",
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

	if !c.Bool("skip-skill") {
		fmt.Fprintln(os.Stderr, "==> skill symlink")
		if err := installSkillSymlink(self); err != nil {
			return fmt.Errorf("setup: skill symlink: %w", err)
		}
	}

	if !c.Bool("skip-lockdown") {
		fmt.Fprintln(os.Stderr, "==> lockdown")
		if err := runLockdownStep(ctx, self, c.String("workspace")); err != nil {
			return err
		}
	}

	fmt.Fprintln(os.Stderr, "setup: done")
	return nil
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

// installSkillSymlink points ~/.claude/skills/coily at the skill dir staged
// next to the coily binary (<bin>/../share/coily/skill). This matches the
// brew formula's `(pkgshare/"skill").install Dir["skill/*"]` layout.
func installSkillSymlink(selfPath string) error {
	binDir := filepath.Dir(selfPath)
	src := filepath.Join(binDir, "..", "share", "coily", "skill")
	src, err := filepath.Abs(src)
	if err != nil {
		return fmt.Errorf("resolve skill path: %w", err)
	}
	if _, err := os.Stat(filepath.Join(src, "SKILL.md")); err != nil {
		return fmt.Errorf("skill not found at %s (expected when installed via brew). %w", src, err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}
	skillsDir := filepath.Join(home, ".claude", "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", skillsDir, err)
	}
	dst := filepath.Join(skillsDir, "coily")
	if err := os.RemoveAll(dst); err != nil {
		return fmt.Errorf("remove existing %s: %w", dst, err)
	}
	if err := os.Symlink(src, dst); err != nil {
		return fmt.Errorf("symlink: %w", err)
	}
	fmt.Fprintf(os.Stderr, "    %s -> %s\n", dst, src)
	return nil
}

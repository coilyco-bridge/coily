package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/coilysiren/coily/pkg/lockdown"
	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

// setupCommand runs the post-upgrade rituals brew's sandbox blocks from
// post_install: tab-completion, lockdown re-baseline, and the user-level
// PreToolUse hook. Idempotent; safe to run any time.
//
// The coily-passthroughs skill is no longer installed from here. It is a
// generated artifact that lives at coily/skills/coily-passthroughs/ in the
// source tree and is symlinked into ~/.claude/skills/ by
// coilyco-ai/setup.sh, alongside every other authored skill. That keeps
// "where do my skills come from?" answerable in one place
// (coilyco-ai/.claude/skills/) instead of two.
func (r *Runner) setupCommand() *cli.Command {
	return &cli.Command{
		Name:  "setup",
		Usage: "Run the post-upgrade rituals: completion, lockdown re-baseline, and user hook.",
		Description: `setup runs three idempotent steps in order:

  1. coily install-completion         (refresh shell tab-completion)
  2. coily lockdown --recursive ...   (re-baseline workspace allow/deny lists)
  3. ~/.claude/coily-binary-gate.sh   (user-level PreToolUse hook that
                                       rejects dev coily binaries from any
                                       cwd; complements per-repo lockdown)

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
				Name:  "skip-lockdown",
				Usage: "skip the lockdown re-baseline step",
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

//go:build dev

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"
)

// install-skill is only registered in dev builds. Symlinking into
// ~/.claude/skills/coily/ affects future Claude Code sessions, which is
// exactly the kind of "edit that has consequences" the threat model warns
// about. Keeping it out of /usr/local/bin/coily means an agent that lands
// inside the coily allowlist cannot steer its own future sessions.
func init() { registerDevOnlyCommand(installSkillCmd) }

var installSkillCmd = &cli.Command{
	Name:  "install-skill",
	Usage: "Symlink ./skill/ into ~/.claude/skills/coily/ so Claude Code picks up the generated skill. (dev build only)",
	Description: `install-skill wires the generated skill into the Claude Code skills directory.

Must be run from the coily repo root (looks for ./skill/SKILL.md in the CWD). Run
'coily skill-gen' first if ./skill/SKILL.md does not exist yet.

Creates ~/.claude/skills/coily as a symlink to <repo>/skill so that future skill
regenerations are picked up automatically.`,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "force",
			Usage: "remove any existing ~/.claude/skills/coily before creating the symlink",
		},
	},
	Action: func(_ context.Context, c *cli.Command) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		src := filepath.Join(cwd, "skill")
		if _, err := os.Stat(filepath.Join(src, "SKILL.md")); err != nil {
			return fmt.Errorf("install-skill: %s/SKILL.md not found. Run `coily skill-gen` first, or invoke from the coily repo root", src)
		}

		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		skillsDir := filepath.Join(home, ".claude", "skills")
		if err := os.MkdirAll(skillsDir, 0o755); err != nil {
			return fmt.Errorf("install-skill: mkdir %s: %w", skillsDir, err)
		}

		dst := filepath.Join(skillsDir, "coily")
		if _, err := os.Lstat(dst); err == nil {
			if !c.Bool("force") {
				return fmt.Errorf("install-skill: %s already exists. Pass --force to replace", dst)
			}
			if err := os.RemoveAll(dst); err != nil {
				return fmt.Errorf("install-skill: remove existing %s: %w", dst, err)
			}
		}

		if err := os.Symlink(src, dst); err != nil {
			return fmt.Errorf("install-skill: symlink: %w", err)
		}
		fmt.Fprintf(os.Stderr, "symlinked %s -> %s\n", dst, src)
		return nil
	},
}

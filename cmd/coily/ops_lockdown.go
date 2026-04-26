package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/coilysiren/coily/pkg/lockdown"
	"github.com/coilysiren/coily/pkg/skillgen"
	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

// lockdownSkillCommand regenerates skills/coily-passthroughs/SKILL.md by
// walking the in-process cli.Command tree. Sits under `coily lockdown`
// because the skill is the discoverability side of the deny list - same
// event ("the coily command surface changed") regenerates both. CI
// diff-checks the file; the pre-commit hook keeps it fresh locally.
func (r *Runner) lockdownSkillCommand() *cli.Command {
	return &cli.Command{
		Name:  "skill",
		Usage: "Regenerate the coily-passthroughs skill from the in-process command tree.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "out",
				Usage: "output path (defaults to SKILL.md / commands.yaml depending on --format)",
			},
			&cli.StringFlag{
				Name:  "format",
				Usage: "markdown (default; SKILL.md for Claude Code) or yaml (structured tree for programmatic consumers)",
				Value: "markdown",
			},
		},
		Action: verb.Wrap(
			verb.Spec{
				Name: "lockdown.skill",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{"--format": c.String("format"), "--out": c.String("out")}, nil
				},
				Action: func(_ context.Context, c *cli.Command) error {
					format := c.String("format")
					out := c.String("out")
					var body string
					switch format {
					case "markdown", "":
						body = skillgen.RenderPassthroughs(r.builtInCommands())
						if out == "" {
							out = "skills/coily-passthroughs/SKILL.md"
						}
					case "yaml":
						y, err := skillgen.RenderPassthroughsYAML(r.builtInCommands())
						if err != nil {
							return err
						}
						body = y
						if out == "" {
							out = "skills/coily-passthroughs/commands.yaml"
						}
					default:
						return fmt.Errorf("lockdown skill: --format must be markdown or yaml, got %q", format)
					}
					if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
						return fmt.Errorf("lockdown skill: mkdir: %w", err)
					}
					if err := os.WriteFile(out, []byte(body), 0o644); err != nil {
						return fmt.Errorf("lockdown skill: write: %w", err)
					}
					fmt.Fprintln(os.Stderr, "wrote", out)
					return nil
				},
			},
			r.Audit,
		),
	}
}

// lockdownCommand is tiered by blast radius:
//
//   - bare `coily lockdown` prints the plan, no write.
//   - `coily lockdown --apply` writes only if .claude/settings.json is absent.
//     Refuses an existing file. Frictionless bootstrap.
//   - `coily lockdown --apply --replace` overwrites an existing file. This is
//     the path that can clobber custom allow/deny entries the user added by
//     hand.
//
// There is no middle ground between "bootstrap fresh" and "clobber".
func (r *Runner) lockdownCommand() *cli.Command {
	return &cli.Command{
		Name:  "lockdown",
		Usage: "Write per-repo Claude Code permissions that force all ops through coily.",
		Description: `lockdown renders a .claude/settings.json (or settings.local.json) for the
target directory with the canonical allow/deny lists baked into coily.

Three modes, by blast radius:

  coily lockdown                    Print the plan and exit. No write.
  coily lockdown --apply            Write a fresh file. Refuses if one exists.
  coily lockdown --apply --replace  Overwrite an existing settings file.`,
		Commands: []*cli.Command{
			r.lockdownSkillCommand(),
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "path",
				Usage: "directory whose .claude/ subdir to target",
				Value: ".",
			},
			&cli.BoolFlag{
				Name:  "local",
				Usage: "write to .claude/settings.local.json instead of settings.json",
			},
			&cli.BoolFlag{
				Name:  "apply",
				Usage: "actually write the file (default: dry-run)",
			},
			&cli.BoolFlag{
				Name:  "replace",
				Usage: "overwrite an existing settings file (requires --apply)",
			},
		},
		Action: verb.Wrap(
			verb.Spec{
				Name: "lockdown",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--path": c.String("path"),
					}, nil
				},
				Action: lockdownAction,
			},
			r.Audit,
		),
	}
}

func lockdownAction(_ context.Context, c *cli.Command) error {
	apply := c.Bool("apply")
	replace := c.Bool("replace")

	if replace && !apply {
		return fmt.Errorf("lockdown: --replace requires --apply (use `coily lockdown --apply --replace`)")
	}

	d, err := lockdown.LoadDefaults()
	if err != nil {
		return err
	}
	target := lockdown.TargetPath(c.String("path"), c.Bool("local"))
	plan, err := lockdown.BuildPlan(target, d)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "target: %s\n", plan.TargetPath)
	switch {
	case !plan.Existed:
		fmt.Fprintln(os.Stderr, "target does not exist; --apply will create it")
	case replace:
		fmt.Fprintln(os.Stderr, "existing file will be overwritten by --replace")
	default:
		fmt.Fprintln(os.Stderr, "existing file present; --apply alone refuses (use --apply --replace to clobber)")
	}

	if !apply {
		fmt.Fprintln(os.Stderr, "--- plan (dry run, pass --apply to write) ---")
		fmt.Print(string(prettyJSON(plan.After)))
		fmt.Println()
		return nil
	}

	if plan.Existed && !replace {
		return fmt.Errorf("lockdown: %s already exists. Use `coily lockdown --apply --replace` to overwrite", plan.TargetPath)
	}

	return writeLockdown(plan, d)
}

func writeLockdown(plan *lockdown.Plan, d *lockdown.Defaults) error {
	if err := lockdown.Write(plan); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "wrote", plan.TargetPath)
	hookPath, err := lockdown.WriteHook(plan.TargetPath, d)
	if err != nil {
		return fmt.Errorf("lockdown: hook write failed (settings.json was written): %w", err)
	}
	fmt.Fprintln(os.Stderr, "wrote", hookPath)
	return nil
}

func prettyJSON(b []byte) []byte {
	var buf bytes.Buffer
	if err := json.Indent(&buf, b, "", "  "); err != nil {
		return b
	}
	return buf.Bytes()
}

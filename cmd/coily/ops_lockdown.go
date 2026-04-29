package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
				Name:      "lockdown.skill",
				SkipScope: true,
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
  coily lockdown --apply --replace  Overwrite an existing settings file.

Pass --recursive to walk up to 4 directories below --path and lock down each
discovered git repo.`,
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
			&cli.BoolFlag{
				Name:  "recursive",
				Usage: "scan up to 4 directories below --path for git repos and lock down each",
			},
		},
		Action: verb.Wrap(
			verb.Spec{
				Name:      "lockdown",
				SkipScope: true,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--path":      c.String("path"),
						"--recursive": fmt.Sprintf("%t", c.Bool("recursive")),
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
	recursive := c.Bool("recursive")

	if replace && !apply {
		return fmt.Errorf("lockdown: --replace requires --apply (use `coily lockdown --apply --replace`)")
	}

	d, err := lockdown.LoadDefaults()
	if err != nil {
		return err
	}

	root := c.String("path")
	local := c.Bool("local")

	var dirs []string
	if recursive {
		found, err := findGitRepos(root, recursiveScanDepth)
		if err != nil {
			return err
		}
		if len(found) == 0 {
			return fmt.Errorf("lockdown: --recursive found no git repos within %d levels of %s", recursiveScanDepth, root)
		}
		dirs = found
		fmt.Fprintf(os.Stderr, "recursive: found %d git repo(s) under %s\n", len(dirs), displayPath(root))
	} else {
		dirs = []string{root}
	}

	for _, dir := range dirs {
		if err := lockdownOne(dir, local, apply, replace, d); err != nil {
			return err
		}
	}
	return nil
}

func lockdownOne(dir string, local, apply, replace bool, d *lockdown.Defaults) error {
	target := lockdown.TargetPath(dir, local)
	plan, err := lockdown.BuildPlan(target, d)
	if err != nil {
		return err
	}

	disp := displayPath(plan.TargetPath)

	if !apply {
		var verb string
		switch {
		case !plan.Existed:
			verb = "would create"
		case replace:
			verb = "would replace"
		default:
			verb = "would refuse (exists; use --apply --replace to clobber)"
		}
		fmt.Fprintf(os.Stderr, "%s: %s\n", disp, verb)
		return nil
	}

	if plan.Existed && !replace {
		return fmt.Errorf("lockdown: %s already exists. Use `coily lockdown --apply --replace` to overwrite", disp)
	}

	verb := "created"
	if plan.Existed {
		verb = "replaced"
	}
	return writeLockdown(plan, d, verb)
}

// displayPath shortens an absolute path to its cwd-relative form when that
// is shorter and stays inside the working tree. Falls back to the original.
func displayPath(p string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return p
	}
	rel, err := filepath.Rel(cwd, p)
	if err != nil || strings.HasPrefix(rel, "..") {
		return p
	}
	if len(rel) < len(p) {
		return rel
	}
	return p
}

const recursiveScanDepth = 4

// findGitRepos walks root up to maxDepth levels deep looking for directories
// that contain a .git entry (file or dir, to support worktrees and submodules).
// Returns the repo directories themselves, sorted, deduplicated. The .git
// subtree is never descended into.
func findGitRepos(root string, maxDepth int) ([]string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("lockdown: resolve %s: %w", root, err)
	}
	var repos []string
	err = filepath.WalkDir(absRoot, func(path string, de os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !de.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(absRoot, path)
		if err != nil {
			return err
		}
		depth := 0
		if rel != "." {
			depth = len(strings.Split(rel, string(filepath.Separator)))
		}
		if depth > maxDepth {
			return filepath.SkipDir
		}
		if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
			repos = append(repos, path)
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("lockdown: scan %s: %w", absRoot, err)
	}
	return repos, nil
}

func writeLockdown(plan *lockdown.Plan, d *lockdown.Defaults, verb string) error {
	if err := lockdown.Write(plan); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "%s: %s\n", displayPath(plan.TargetPath), verb)
	hookPath, hookExisted, err := lockdown.WriteHook(plan.TargetPath, d)
	if err != nil {
		return fmt.Errorf("lockdown: hook write failed (settings.json was written): %w", err)
	}
	hookVerb := "created"
	if hookExisted {
		hookVerb = "replaced"
	}
	fmt.Fprintf(os.Stderr, "%s: %s\n", displayPath(hookPath), hookVerb)
	return nil
}

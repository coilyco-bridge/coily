package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/coilysiren/coily/pkg/lockdown"
	"github.com/coilysiren/coily/pkg/policy"
	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

func (r *Runner) lockdownCommand() *cli.Command {
	return &cli.Command{
		Name:  "lockdown",
		Usage: "Write per-repo Claude Code permissions that force all ops through coily.",
		Description: `lockdown renders a .claude/settings.json (or settings.local.json) for the
target directory with the canonical allow/deny lists baked into coily.

Without --apply, prints the plan (diff of before/after) and exits. With
--apply, writes the file to disk. Existing allow/deny entries are merged,
not replaced, unless --replace is also set.`,
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
				Usage: "replace existing allow/deny entries instead of merging",
			},
		},
		Action: verb.Wrap(
			verb.Spec{
				Name: "lockdown",
				Kind: policy.ReadOnly,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string, string) {
					return map[string]string{
						"--path": c.String("path"),
					}, nil, ""
				},
				Action: lockdownAction,
			},
			r.Verifier,
			r.Audit,
		),
	}
}

func lockdownAction(_ context.Context, c *cli.Command) error {
	d, err := lockdown.LoadDefaults()
	if err != nil {
		return err
	}
	target := lockdown.TargetPath(c.String("path"), c.Bool("local"))
	plan, err := lockdown.BuildPlan(target, d, c.Bool("replace"))
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "target: %s\n", plan.TargetPath)
	if plan.Existed {
		fmt.Fprintln(os.Stderr, "existing file will be merged (use --replace to clobber)")
	} else {
		fmt.Fprintln(os.Stderr, "target does not exist; will be created")
	}

	if !c.Bool("apply") {
		fmt.Fprintln(os.Stderr, "--- plan (dry run, pass --apply to write) ---")
		fmt.Print(string(prettyJSON(plan.After)))
		fmt.Println()
		return nil
	}

	if err := lockdown.Write(plan); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "wrote", plan.TargetPath)
	return nil
}

func prettyJSON(b []byte) []byte {
	var buf bytes.Buffer
	if err := json.Indent(&buf, b, "", "  "); err != nil {
		return b
	}
	return buf.Bytes()
}

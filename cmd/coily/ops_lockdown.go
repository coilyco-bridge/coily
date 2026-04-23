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

func init() { registerCommand(lockdownCmd) }

// lockdown is tiered by blast radius (resolves docs/unresolved/13-lockdown-token.md):
//
//   - bare `coily lockdown` -> ReadOnly, prints plan, no token.
//   - `coily lockdown --apply` -> ReadOnly, writes only if .claude/settings.json
//     is absent. Refuses an existing file. No token. Frictionless bootstrap.
//   - `coily lockdown --apply --replace` -> Mutating, overwrites an existing
//     file. Token required. This is the path that can clobber custom allow/deny
//     entries the user added by hand.
//
// The previous silent-merge behavior is gone. There is no middle ground
// between "bootstrap fresh" and "clobber".
var lockdownCmd = &cli.Command{
	Name:  "lockdown",
	Usage: "Write per-repo Claude Code permissions that force all ops through coily.",
	Description: `lockdown renders a .claude/settings.json (or settings.local.json) for the
target directory with the canonical allow/deny lists baked into coily.

Three modes, by blast radius:

  coily lockdown                    Print the plan and exit. No write.
  coily lockdown --apply            Write a fresh file. Refuses if one exists.
  coily lockdown --apply --replace  Overwrite an existing file. Requires a token.`,
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
			Usage: "overwrite an existing settings file (requires --apply, requires a token)",
		},
		&cli.StringFlag{
			Name:  "token",
			Usage: "confirmation token scoped to lockdown (only consulted with --apply --replace)",
		},
	},
	Action: verb.Wrap(
		verb.Spec{
			Name: "lockdown",
			// Dynamic classification: only --apply --replace is mutating.
			// Bare invocations and fresh-bootstrap (--apply alone) stay
			// ReadOnly so the safety boundary can be turned on without a
			// token round-trip.
			KindFunc: func(c *cli.Command) policy.Kind {
				if c.Bool("apply") && c.Bool("replace") {
					return policy.Mutating
				}
				return policy.ReadOnly
			},
			ArgsFunc: func(c *cli.Command) (map[string]string, []string, string) {
				return map[string]string{
					"--path": c.String("path"),
				}, nil, c.String("token")
			},
			Action: lockdownAction,
		},
		getRuntime().issuer,
		getRuntime().audit,
	),
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
		return fmt.Errorf("lockdown: %s already exists. Use `coily lockdown --apply --replace` to overwrite (requires a token)", plan.TargetPath)
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

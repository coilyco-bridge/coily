package main

import (
	"context"
	"fmt"
	"os"

	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// upgradeCommand wraps the brew self-update path so the agent has one
// audited verb instead of raw brew exec. Bare brew is denied at the
// lockdown layer (defaults.yaml), and `coily brew upgrade` is the
// general-purpose audited brew wrapper. `coily upgrade` is the narrower
// shorthand bound to coilysiren/tap/coily specifically: brew update +
// brew upgrade coilysiren/tap/coily, no formula argument needed.
//
// Per coilysiren/coily#19. Sits next to versionCommand because the two
// pair operationally: `coily version` says what's installed, `coily
// upgrade` ships a newer one.
func (r *Runner) upgradeCommand() *cli.Command {
	return &cli.Command{
		Name:  "upgrade",
		Usage: "Self-update via brew (coilysiren/tap/coily).",
		Description: `upgrade runs the audited brew sequence:

    brew update
    brew upgrade coilysiren/tap/coily

Pass --dry to see the resolved version diff without installing
(equivalent to ` + "`brew outdated coilysiren/tap/coily`" + `).

Bare brew is denied at the lockdown layer; this verb is the audited
recovery path for an agent that needs a fresh coily binary. The
` + "`coily brew`" + ` wrapper handles the general install/upgrade case for
any tap formula. ` + "`coily upgrade`" + ` is the coily-specific shortcut.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry",
				Usage: "show the resolved version diff without installing",
			},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name:      "upgrade",
				SkipScope: true,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{"--dry": fmt.Sprintf("%t", c.Bool("dry"))}, nil
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return runUpgrade(ctx, r, c.Bool("dry"))
				},
			},
			r.Audit,
		),
	}
}

// upgradeFormula is the qualified brew formula name. Hard-coded rather
// than configurable: this verb's whole point is that it self-updates
// the coily binary specifically, not an arbitrary formula. Use
// `coily brew upgrade <formula>` for the general case.
const upgradeFormula = "coilysiren/tap/coily"

func runUpgrade(ctx context.Context, r *Runner, dry bool) error {
	if dry {
		fmt.Fprintln(os.Stderr, "==> brew outdated", upgradeFormula)
		return r.Runner.Exec(ctx, "brew", "outdated", upgradeFormula)
	}
	fmt.Fprintln(os.Stderr, "==> brew update")
	if err := r.Runner.Exec(ctx, "brew", "update"); err != nil {
		return fmt.Errorf("upgrade: brew update: %w", err)
	}
	fmt.Fprintln(os.Stderr, "==> brew upgrade", upgradeFormula)
	if err := r.Runner.Exec(ctx, "brew", "upgrade", upgradeFormula); err != nil {
		return fmt.Errorf("upgrade: brew upgrade %s: %w", upgradeFormula, err)
	}
	return nil
}

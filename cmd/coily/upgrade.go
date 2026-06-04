package main

import (
	"context"
	"fmt"
	"os"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// upgradeCommand wraps the brew self-update path so the agent has one
// audited verb instead of raw brew exec. Bare brew is denied at the
// lockdown layer (defaults.yaml), and `coily brew upgrade` is the
// general-purpose audited brew wrapper. `coily upgrade` is the narrower
// shorthand bound to coily-the-formula specifically: brew update +
// brew upgrade coilysiren/coily/coily, no formula argument needed.
//
// The formula is the per-repo tap (coilysiren/coily/coily). The umbrella
// tap (coilysiren/tap) was decommissioned in favor of per-repo taps; the
// transitional dual-tap resolution from coilysiren/coily#271 is gone.
// See coilyco-bridge/coily#22.
//
// Per coilysiren/coily#19. Sits next to versionCommand because the two
// pair operationally: `coily version` says what's installed, `coily
// upgrade` ships a newer one.
func (r *Runner) upgradeCommand() *cli.Command {
	return &cli.Command{
		Name:  "upgrade",
		Usage: "Self-update via brew (coilysiren/coily/coily per-repo tap).",
		Description: `upgrade runs the audited brew sequence:

    brew update
    brew upgrade coilysiren/coily/coily

The formula is the per-repo tap coilysiren/coily/coily. Pass --dry to see
the resolved version diff without installing (equivalent to
` + "`brew outdated coilysiren/coily/coily`" + `).

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
				Name: "upgrade",
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

// upgradeFormula is the qualified formula name coily's self-upgrade
// feeds `brew upgrade`. The verb is bound to the coily binary
// specifically (not an arbitrary formula); the formula lives in the
// per-repo tap coilysiren/coily. The umbrella tap (coilysiren/tap) was
// decommissioned in favor of per-repo taps, so coilyco-bridge/coily#22
// dropped the coilysiren/coily#271 runtime dual-tap resolution and
// hardcodes the per-repo name.
const upgradeFormula = "coilysiren/coily/coily"

func runUpgrade(ctx context.Context, r *Runner, dry bool) error {
	formula := upgradeFormula
	if dry {
		fmt.Fprintln(os.Stderr, "==> brew outdated", formula)
		return r.Runner.Exec(ctx, "brew", "outdated", formula)
	}
	fmt.Fprintln(os.Stderr, "==> brew update")
	if err := r.Runner.Exec(ctx, "brew", "update"); err != nil {
		return fmt.Errorf("upgrade: brew update: %w", err)
	}
	fmt.Fprintln(os.Stderr, "==> brew upgrade", formula)
	if err := r.Runner.Exec(ctx, "brew", "upgrade", formula); err != nil {
		return fmt.Errorf("upgrade: brew upgrade %s: %w", formula, err)
	}
	return nil
}

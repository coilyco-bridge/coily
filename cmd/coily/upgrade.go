package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// upgradeCommand wraps the brew self-update path so the agent has one
// audited verb instead of raw brew exec. Bare brew is denied at the
// lockdown layer (defaults.yaml), and `coily brew upgrade` is the
// general-purpose audited brew wrapper. `coily upgrade` is the narrower
// shorthand bound to coily-the-formula specifically: brew update +
// brew upgrade <coily formula>, no formula argument needed. The
// qualified formula name is resolved at runtime from `brew tap` so
// hosts that ship the per-repo tap (coilysiren/coily) and hosts that
// ship the umbrella (coilysiren/tap) both work. See coilysiren/coily#271.
//
// Per coilysiren/coily#19. Sits next to versionCommand because the two
// pair operationally: `coily version` says what's installed, `coily
// upgrade` ships a newer one.
func (r *Runner) upgradeCommand() *cli.Command {
	return &cli.Command{
		Name:  "upgrade",
		Usage: "Self-update via brew (coilysiren tap, per-repo or umbrella).",
		Description: `upgrade runs the audited brew sequence:

    brew update
    brew upgrade <coilysiren tap>/coily

The qualified formula name is resolved from ` + "`brew tap`" + ` and prefers
the per-repo tap (coilysiren/coily/coily) over the umbrella
(coilysiren/tap/coily) when both are installed. Pass --dry to see the
resolved version diff without installing (equivalent to
` + "`brew outdated <resolved formula>`" + `).

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

// Candidate qualified formula names for coily's self-upgrade. The verb
// is bound to the coily binary specifically (not an arbitrary formula);
// the only knob is which coilysiren tap is providing it on this host.
// Per coilysiren/coily#271 hosts may have either the per-repo tap
// (coilysiren/homebrew-coily) or the umbrella (coilysiren/homebrew-tap)
// installed. The per-repo tap is preferred when both are present
// because that is the current direct-tap release flow.
const (
	upgradeFormulaPerRepo  = "coilysiren/coily/coily"
	upgradeFormulaUmbrella = "coilysiren/tap/coily"
	upgradeTapPerRepo      = "coilysiren/coily"
)

// resolveUpgradeFormula picks which qualified formula name to feed
// `brew upgrade`. Enumerates installed taps via `brew tap` and prefers
// the per-repo tap when present. Falls back to the umbrella formula
// when brew tap fails or neither tap is recognized, so the verb still
// produces brew's own clearer error rather than a coily-side guess.
func resolveUpgradeFormula(ctx context.Context, r *Runner) string {
	out, err := r.Runner.Capture(ctx, "brew", "tap")
	if err != nil {
		return upgradeFormulaUmbrella
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) == upgradeTapPerRepo {
			return upgradeFormulaPerRepo
		}
	}
	return upgradeFormulaUmbrella
}

func runUpgrade(ctx context.Context, r *Runner, dry bool) error {
	formula := resolveUpgradeFormula(ctx, r)
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

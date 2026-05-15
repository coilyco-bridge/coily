package main

import (
	"context"
	"fmt"

	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// systemctlCommand is the local-execution sibling of `coily ssh systemctl`.
// Same verb table, same argv shapes, but the action runs against the host
// the binary is on (sudo where mutating, no sudo for status; status reads
// cached systemd state and trips a tty-prompt under sudo on non-tty
// sessions per coilysiren/coily#144).
//
// The intended call-site is the remote coily that local
// `coily ssh kai-server -- coily systemctl <verb> <unit>` dispatches to.
// Direct `coily systemctl ...` use from a Mac is supported but no-ops
// without a local systemd. The point of this verb is to be the migration
// target for coilysiren/coily#187 step 7: prove the passthrough on one
// real call site before tearing down the per-verb ssh wrappers in step 8.
func (r *Runner) systemctlCommand() *cli.Command {
	cmds := make([]*cli.Command, 0, len(systemctlVerbs))
	for _, v := range systemctlVerbs {
		cmds = append(cmds, r.systemctlVerb(v.Name, v.Usage, v.NeedsUnit, v.NoSudo, v.Argv))
	}
	return &cli.Command{
		Name:  "systemctl",
		Usage: "Run a fixed-shape systemctl verb on the local host.",
		Description: `Local-execution sibling of ` + "`coily ssh systemctl`" + `.
Same closed verb set (status/start/stop/restart/enable/disable/
daemon-reload), same argv shapes, but each leaf invokes the local
systemctl directly rather than ssh-dispatching to kai-server. Sudo
discipline matches the ssh path: status runs unprivileged (the read
itself is unprivileged and sudo trips a tty prompt on non-tty
sessions, coilysiren/coily#144), mutating verbs are sudo-prefixed.

Designed for the remote side of ` + "`coily ssh kai-server -- coily systemctl <verb> <unit>`" + `,
which is the coily#187 step 7 migration target.`,
		Commands: cmds,
	}
}

func (r *Runner) systemctlVerb(name, usage string, needsUnit, noSudo bool, build func(string) []string) *cli.Command {
	argsUsage := "<unit>"
	if !needsUnit {
		argsUsage = ""
	}
	return &cli.Command{
		Name:      name,
		Usage:     usage,
		ArgsUsage: argsUsage,
		Action: r.WrapVerb(
			verb.Spec{
				Name: "systemctl." + name,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return nil, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					var unit string
					if needsUnit {
						if c.Args().Len() != 1 {
							return fmt.Errorf("systemctl %s: need exactly one <unit> arg, got %d", name, c.Args().Len())
						}
						unit = c.Args().First()
						if err := validateUnitName(unit); err != nil {
							return err
						}
					} else if c.Args().Len() != 0 {
						return fmt.Errorf("systemctl %s: takes no args, got %d", name, c.Args().Len())
					}
					argv := build(unit)
					if !noSudo {
						argv = append([]string{"sudo"}, argv...)
					}
					return r.Runner.Exec(ctx, argv[0], argv[1:]...)
				},
			},
			r.Audit,
		),
	}
}

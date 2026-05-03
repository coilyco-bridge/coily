package main

import "github.com/urfave/cli/v3"

// gamingCommand is the umbrella verb for kai-server game-server ops.
// It groups the four systemd-managed game servers (eco, core-keeper,
// icarus, factorio) under a single namespace so help output shows them
// together and so future cross-game helpers (per-game backup, drift
// checks) have an obvious home.
//
// The individual game builders (ecoCommand, coreKeeperCommand,
// icarusCommand, factorioCommand) are unchanged: their internal verb
// names ("eco.status" etc.) stay stable so audit log continuity holds
// across the rename. Only the user-visible path changes:
//
//	coily eco status        -> coily gaming eco status
//	coily core-keeper start -> coily gaming core-keeper start
//	coily icarus restart    -> coily gaming icarus restart
//	(new)                      coily gaming factorio status
func (r *Runner) gamingCommand() *cli.Command {
	return &cli.Command{
		Name:  "gaming",
		Usage: "Operate the kai-server game servers (eco, core-keeper, icarus, factorio).",
		Description: `gaming is the umbrella for the four systemd-managed game servers
on kai-server. Each subcommand is a per-game verb tree.

  coily gaming eco {status,tail,start,stop,restart,world,mod}
  coily gaming core-keeper {status,tail,start,stop,restart}
  coily gaming icarus {status,tail,start,stop,restart}
  coily gaming factorio {status,tail,start,stop,restart,update,saves,mods,players}
                          mods has {list,sync}; sync pulls archives into mods/
                          to match mod-list.json via the Factorio mod portal.

Every leaf verb routes through pkg/ssh against kai-server (no ssh
subprocess), with audit + policy enforcement provided by verb.Wrap.`,
		Commands: []*cli.Command{
			r.ecoCommand(),
			r.coreKeeperCommand(),
			r.icarusCommand(),
			r.factorioCommand(),
		},
	}
}

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/coilysiren/coily/pkg/ops/eco"
	"github.com/coilysiren/coily/pkg/policy"
	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

func init() { registerCommand(ecoCmd) }

// ecoCmd wraps the eco game server which runs as a systemd unit on
// kai-server. All verbs ultimately shell out to
// `ssh <user>@<host> sudo <systemctl|journalctl> ... eco-server`.
// The ssh target is taken from embedded config (kai_server.tailscale_host
// and ssh_user).
var ecoCmd = &cli.Command{
	Name:  "eco",
	Usage: "Operate the eco game server (systemd unit on kai-server).",
	Description: `eco wraps systemctl/journalctl calls against the eco-server unit that runs
on kai-server. Destructive verbs (restart, stop) require a confirmation
token. Reads (status, tail) do not.`,
	Commands: []*cli.Command{
		ecoStatusCmd,
		ecoTailCmd,
		ecoRestartCmd,
		ecoStopCmd,
		ecoStartCmd,
		ecoWorldCmd,
	},
}

var ecoStatusCmd = &cli.Command{
	Name:  "status",
	Usage: "Print systemctl status eco-server.",
	Action: verb.Wrap(
		verb.Spec{
			Name:   "eco.status",
			Kind:   policy.ReadOnly,
			Action: ecoRemote([]string{"sudo", "systemctl", "status", "eco-server", "--no-pager"}),
		},
		getRuntime().issuer,
		getRuntime().audit,
	),
}

var ecoTailCmd = &cli.Command{
	Name:  "tail",
	Usage: "Tail eco-server journal logs (journalctl -u eco-server -f).",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:  "lines",
			Usage: "number of lines of history to emit before tailing",
			Value: 200,
		},
		&cli.BoolFlag{
			Name:  "follow",
			Usage: "keep tailing after the initial history (default: true)",
			Value: true,
		},
	},
	Action: verb.Wrap(
		verb.Spec{
			Name: "eco.tail",
			Kind: policy.ReadOnly,
			Action: func(ctx context.Context, c *cli.Command) error {
				args := []string{"sudo", "journalctl", "-u", "eco-server", "-n", fmt.Sprint(c.Int("lines"))}
				if c.Bool("follow") {
					args = append(args, "-f")
				}
				return ecoRemote(args)(ctx, c)
			},
		},
		getRuntime().issuer,
		getRuntime().audit,
	),
}

var ecoRestartCmd = &cli.Command{
	Name:  "restart",
	Usage: "Restart the eco-server systemd unit. Requires a confirmation token.",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "token", Usage: "confirmation token scoped to eco.restart"},
	},
	Action: verb.Wrap(
		verb.Spec{
			Name: "eco.restart",
			Kind: policy.Mutating,
			ArgsFunc: func(c *cli.Command) (map[string]string, []string, string) {
				return nil, nil, c.String("token")
			},
			Action: ecoRemote([]string{"sudo", "systemctl", "restart", "eco-server"}),
		},
		getRuntime().issuer,
		getRuntime().audit,
	),
}

var ecoStopCmd = &cli.Command{
	Name:  "stop",
	Usage: "Stop the eco-server systemd unit. Requires a confirmation token.",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "token", Usage: "confirmation token scoped to eco.stop"},
	},
	Action: verb.Wrap(
		verb.Spec{
			Name: "eco.stop",
			Kind: policy.Mutating,
			ArgsFunc: func(c *cli.Command) (map[string]string, []string, string) {
				return nil, nil, c.String("token")
			},
			Action: ecoRemote([]string{"sudo", "systemctl", "stop", "eco-server"}),
		},
		getRuntime().issuer,
		getRuntime().audit,
	),
}

var ecoStartCmd = &cli.Command{
	Name:  "start",
	Usage: "Start the eco-server systemd unit. Requires a confirmation token.",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "token", Usage: "confirmation token scoped to eco.start"},
	},
	Action: verb.Wrap(
		verb.Spec{
			Name: "eco.start",
			Kind: policy.Mutating,
			ArgsFunc: func(c *cli.Command) (map[string]string, []string, string) {
				return nil, nil, c.String("token")
			},
			Action: ecoRemote([]string{"sudo", "systemctl", "start", "eco-server"}),
		},
		getRuntime().issuer,
		getRuntime().audit,
	),
}

// ecoWorldCmd ports the local-side helpers from
// https://github.com/coilysiren/eco-cycle-prep/blob/main/eco_cycle_prep/worldgen.py
// into coily. These verbs operate on a local checkout of the eco-configs
// repo (Configs/WorldGenerator.eco), not on kai-server.
//
// The remote teardown/restart half of a full world cycle (stop server,
// swap files, start server) is handled by the existing
// `coily eco {stop,start,restart}` verbs. Composing those into a single
// `coily eco world rotate` is deliberately deferred until Kai actually
// runs the cycle and tells me how to chain the steps. See
// docs/unresolved.md.
var ecoWorldCmd = &cli.Command{
	Name:  "world",
	Usage: "Read and modify the local eco-configs WorldGenerator.eco file.",
	Description: `world wraps the helpers from eco-cycle-prep/worldgen.py:
get-seed, set-seed, randomize, snapshot. All four are local file ops on
a checkout of the eco-configs repo. None of them touch kai-server. To
restart the eco-server after rotating the world file, use
'coily eco restart' separately.

The eco-configs checkout is located via --configs-dir, falling back to
config.eco.configs_dir from the embedded config.`,
	Commands: []*cli.Command{
		ecoWorldGetSeedCmd,
		ecoWorldSetSeedCmd,
		ecoWorldRandomizeCmd,
		ecoWorldSnapshotCmd,
	},
}

// configsDirFlag is shared by every eco world verb. Empty value falls back
// to config.eco.configs_dir.
var configsDirFlag = &cli.StringFlag{
	Name:  "configs-dir",
	Usage: "path to the eco-configs checkout. Defaults to config.eco.configs_dir",
}

// resolveConfigsDir picks --configs-dir when set, otherwise the embedded
// config value. Returns an error if both are empty.
func resolveConfigsDir(c *cli.Command) (string, error) {
	if v := c.String("configs-dir"); v != "" {
		return expandHome(v), nil
	}
	if v := getRuntime().cfg.Eco.ConfigsDir; v != "" {
		return expandHome(v), nil
	}
	return "", fmt.Errorf("eco world: pass --configs-dir or set eco.configs_dir in the embedded config")
}

var ecoWorldGetSeedCmd = &cli.Command{
	Name:  "get-seed",
	Usage: "Print the current Seed from Configs/WorldGenerator.eco.",
	Flags: []cli.Flag{configsDirFlag},
	Action: verb.Wrap(
		verb.Spec{
			Name: "eco.world.get-seed",
			Kind: policy.ReadOnly,
			ArgsFunc: func(c *cli.Command) (map[string]string, []string, string) {
				return map[string]string{"--configs-dir": c.String("configs-dir")}, nil, ""
			},
			Action: func(_ context.Context, c *cli.Command) error {
				dir, err := resolveConfigsDir(c)
				if err != nil {
					return err
				}
				seed, err := eco.GetSeed(dir)
				if err != nil {
					return err
				}
				fmt.Println(seed)
				return nil
			},
		},
		getRuntime().issuer,
		getRuntime().audit,
	),
}

var ecoWorldSetSeedCmd = &cli.Command{
	Name:  "set-seed",
	Usage: "Write a specific Seed into Configs/WorldGenerator.eco. Requires a confirmation token.",
	Flags: []cli.Flag{
		configsDirFlag,
		&cli.IntFlag{Name: "seed", Usage: "seed value (1..2,000,000,000)", Required: true},
		&cli.StringFlag{Name: "token", Usage: "confirmation token scoped to eco.world.set-seed"},
	},
	Action: verb.Wrap(
		verb.Spec{
			Name: "eco.world.set-seed",
			Kind: policy.Mutating,
			ArgsFunc: func(c *cli.Command) (map[string]string, []string, string) {
				return map[string]string{
					"--configs-dir": c.String("configs-dir"),
					"--seed":        fmt.Sprint(c.Int("seed")),
				}, nil, c.String("token")
			},
			Action: func(_ context.Context, c *cli.Command) error {
				dir, err := resolveConfigsDir(c)
				if err != nil {
					return err
				}
				if err := eco.SetSeed(dir, int64(c.Int("seed"))); err != nil {
					return err
				}
				fmt.Fprintf(os.Stderr, "wrote seed=%d to %s\n", c.Int("seed"), eco.WorldGenPath(dir))
				return nil
			},
		},
		getRuntime().issuer,
		getRuntime().audit,
	),
}

var ecoWorldRandomizeCmd = &cli.Command{
	Name:  "randomize",
	Usage: "Generate a random seed and write it to Configs/WorldGenerator.eco. Requires a confirmation token.",
	Flags: []cli.Flag{
		configsDirFlag,
		&cli.StringFlag{Name: "token", Usage: "confirmation token scoped to eco.world.randomize"},
	},
	Action: verb.Wrap(
		verb.Spec{
			Name: "eco.world.randomize",
			Kind: policy.Mutating,
			ArgsFunc: func(c *cli.Command) (map[string]string, []string, string) {
				return map[string]string{"--configs-dir": c.String("configs-dir")}, nil, c.String("token")
			},
			Action: func(_ context.Context, c *cli.Command) error {
				dir, err := resolveConfigsDir(c)
				if err != nil {
					return err
				}
				seed, err := eco.RandomSeed()
				if err != nil {
					return err
				}
				if err := eco.SetSeed(dir, seed); err != nil {
					return err
				}
				fmt.Fprintf(os.Stderr, "wrote seed=%d to %s\n", seed, eco.WorldGenPath(dir))
				return nil
			},
		},
		getRuntime().issuer,
		getRuntime().audit,
	),
}

var ecoWorldSnapshotCmd = &cli.Command{
	Name:  "snapshot",
	Usage: "Copy Configs/WorldGenerator.eco to --target. Requires a confirmation token.",
	Flags: []cli.Flag{
		configsDirFlag,
		&cli.StringFlag{Name: "target", Usage: "destination file path", Required: true},
		&cli.StringFlag{Name: "token", Usage: "confirmation token scoped to eco.world.snapshot"},
	},
	Action: verb.Wrap(
		verb.Spec{
			Name: "eco.world.snapshot",
			Kind: policy.Mutating,
			ArgsFunc: func(c *cli.Command) (map[string]string, []string, string) {
				return map[string]string{
					"--configs-dir": c.String("configs-dir"),
					"--target":      c.String("target"),
				}, nil, c.String("token")
			},
			Action: func(_ context.Context, c *cli.Command) error {
				dir, err := resolveConfigsDir(c)
				if err != nil {
					return err
				}
				target := expandHome(c.String("target"))
				if err := eco.Snapshot(dir, target); err != nil {
					return err
				}
				fmt.Fprintf(os.Stderr, "wrote snapshot to %s\n", target)
				return nil
			},
		},
		getRuntime().issuer,
		getRuntime().audit,
	),
}

// ecoRemote returns a cli.ActionFunc that ssh's into kai-server and runs the
// given argv (as a single composed remote command, since ssh's last
// positional is passed to a remote shell). Every element of remoteArgv is
// hardcoded at compile time. No user input reaches here, so no runtime
// metacharacter risk from this path.
func ecoRemote(remoteArgv []string) cli.ActionFunc {
	return func(ctx context.Context, _ *cli.Command) error {
		rt := getRuntime()
		host := rt.cfg.KaiServer.TailscaleHost
		user := rt.cfg.KaiServer.SSHUser
		if host == "" || user == "" {
			return fmt.Errorf("eco: kai_server.tailscale_host or ssh_user not configured")
		}
		target := user + "@" + host
		// ssh takes the remote command as its last argv element. We compose
		// the remote command as space-joined because all remoteArgv elements
		// are compile-time constants in this file. If we ever take user
		// input here, add policy.ValidateArgSlice at the entry point.
		argv := append([]string{target}, remoteArgv...)
		return rt.runner.Exec(ctx, "ssh", argv...)
	}
}

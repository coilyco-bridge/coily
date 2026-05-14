package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coilysiren/cli-guard/verb"
	"github.com/coilysiren/coily/cmd/coily/eco"
	"github.com/urfave/cli/v3"
)

// ecoCommand wraps the eco game server which runs as a systemd unit on
// kai-server. All verbs run a `sudo <systemctl|journalctl> ... eco-server`
// command on kai-server through pkg/ssh, which wraps
// golang.org/x/crypto/ssh. No ssh subprocess is spawned. The ssh target is
// taken from embedded config (kai_server.tailscale_host and ssh_user).
//
// `coily eco world` is a sub-tree of local-side helpers ported from
// eco-cycle-prep/worldgen.py. Those verbs operate on a local checkout of
// the eco-configs repo, not on kai-server.
func (r *Runner) ecoCommand() *cli.Command {
	return &cli.Command{
		Name:  "eco",
		Usage: "Operate the eco game server (systemd unit on kai-server).",
		Description: `eco wraps systemctl/journalctl calls against the eco-server unit that runs
on kai-server.

The 'world' sub-tree wraps the local-side helpers from
eco-cycle-prep/worldgen.py for editing the eco-configs WorldGenerator.eco
file. Those do not touch kai-server.`,
		Commands: []*cli.Command{
			r.ecoStatusCommand(),
			r.ecoTailCommand(),
			r.ecoRestartCommand(),
			r.ecoStopCommand(),
			r.ecoStartCommand(),
			r.ecoWorldCommand(),
			r.ecoModCommand(),
		},
	}
}

func (r *Runner) ecoStatusCommand() *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "Print systemctl status eco-server.",
		Action: r.WrapVerb(
			verb.Spec{
				Name:   "eco.status",
				Action: r.ecoRemote([]string{"sudo", "systemctl", "status", "eco-server", "--no-pager"}),
			},
			r.Audit,
		),
	}
}

func (r *Runner) ecoTailCommand() *cli.Command {
	return &cli.Command{
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
		Action: r.WrapVerb(
			verb.Spec{
				Name: "eco.tail",
				Action: func(ctx context.Context, c *cli.Command) error {
					// No sudo: kai is in the adm group, which has read access
					// to /var/log/journal/ on Ubuntu. journalctl honours that
					// without privilege escalation. systemctl writes still
					// need sudo (start/stop/restart below).
					args := []string{"journalctl", "-u", "eco-server", "-n", fmt.Sprint(c.Int("lines"))}
					if c.Bool("follow") {
						args = append(args, "-f")
					}
					return r.ecoRemote(args)(ctx, c)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) ecoRestartCommand() *cli.Command {
	return &cli.Command{
		Name:  "restart",
		Usage: "Restart the eco-server systemd unit.",
		Action: r.WrapVerb(
			verb.Spec{
				Name:   "eco.restart",
				Action: r.ecoRemote([]string{"sudo", "systemctl", "restart", "eco-server"}),
			},
			r.Audit,
		),
	}
}

func (r *Runner) ecoStopCommand() *cli.Command {
	return &cli.Command{
		Name:  "stop",
		Usage: "Stop the eco-server systemd unit.",
		Action: r.WrapVerb(
			verb.Spec{
				Name:   "eco.stop",
				Action: r.ecoRemote([]string{"sudo", "systemctl", "stop", "eco-server"}),
			},
			r.Audit,
		),
	}
}

func (r *Runner) ecoStartCommand() *cli.Command {
	return &cli.Command{
		Name:  "start",
		Usage: "Start the eco-server systemd unit.",
		Action: r.WrapVerb(
			verb.Spec{
				Name:   "eco.start",
				Action: r.ecoRemote([]string{"sudo", "systemctl", "start", "eco-server"}),
			},
			r.Audit,
		),
	}
}

// ecoWorldCommand ports the local-side helpers from
// https://github.com/coilysiren/eco-cycle-prep/blob/main/eco_cycle_prep/worldgen.py
// into coily. These verbs operate on a local checkout of the eco-configs
// repo (Configs/WorldGenerator.eco), not on kai-server.
func (r *Runner) ecoWorldCommand() *cli.Command {
	return &cli.Command{
		Name:  "world",
		Usage: "Read and modify the local eco-configs WorldGenerator.eco file.",
		Description: `world wraps the helpers from eco-cycle-prep/worldgen.py:
get-seed, set-seed, randomize, snapshot. All four are local file ops on
a checkout of the eco-configs repo. None of them touch kai-server. To
restart the eco-server after rotating the world file, use
'coily gaming eco restart' separately.

The eco-configs checkout is located via --configs-dir, falling back to
config.eco.configs_dir from the embedded config.`,
		Commands: []*cli.Command{
			r.ecoWorldGetSeedCommand(),
			r.ecoWorldSetSeedCommand(),
			r.ecoWorldRandomizeCommand(),
			r.ecoWorldSnapshotCommand(),
		},
	}
}

func configsDirFlag() *cli.StringFlag {
	return &cli.StringFlag{
		Name:  "configs-dir",
		Usage: "path to the eco-configs checkout. Defaults to config.eco.configs_dir",
	}
}

// expandTilde turns a leading "~/" into the user's home dir. Returns the
// input unchanged on any failure or when no leading "~" is present.
func expandTilde(p string) string {
	if p == "" || !strings.HasPrefix(p, "~") {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if p == "~" {
		return home
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:])
	}
	return p
}

// resolveConfigsDir picks --configs-dir when set, otherwise the embedded
// config value. Returns an error if both are empty.
func (r *Runner) resolveConfigsDir(c *cli.Command) (string, error) {
	if v := c.String("configs-dir"); v != "" {
		return expandTilde(v), nil
	}
	if v := r.Cfg.Eco.ConfigsDir; v != "" {
		return expandTilde(v), nil
	}
	return "", fmt.Errorf("eco world: pass --configs-dir or set eco.configs_dir in the embedded config")
}

func (r *Runner) ecoWorldGetSeedCommand() *cli.Command {
	return &cli.Command{
		Name:  "get-seed",
		Usage: "Print the current Seed from Configs/WorldGenerator.eco.",
		Flags: []cli.Flag{configsDirFlag()},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "eco.world.get-seed",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{"--configs-dir": c.String("configs-dir")}, nil
				},
				Action: func(_ context.Context, c *cli.Command) error {
					dir, err := r.resolveConfigsDir(c)
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
			r.Audit,
		),
	}
}

func (r *Runner) ecoWorldSetSeedCommand() *cli.Command {
	return &cli.Command{
		Name:  "set-seed",
		Usage: "Write a specific Seed into Configs/WorldGenerator.eco.",
		Flags: []cli.Flag{
			configsDirFlag(),
			&cli.IntFlag{Name: "seed", Usage: "seed value (1..2,000,000,000)", Required: true},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "eco.world.set-seed",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--configs-dir": c.String("configs-dir"),
						"--seed":        fmt.Sprint(c.Int("seed")),
					}, nil
				},
				Action: func(_ context.Context, c *cli.Command) error {
					dir, err := r.resolveConfigsDir(c)
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
			r.Audit,
		),
	}
}

func (r *Runner) ecoWorldRandomizeCommand() *cli.Command {
	return &cli.Command{
		Name:  "randomize",
		Usage: "Generate a random seed and write it to Configs/WorldGenerator.eco.",
		Flags: []cli.Flag{configsDirFlag()},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "eco.world.randomize",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{"--configs-dir": c.String("configs-dir")}, nil
				},
				Action: func(_ context.Context, c *cli.Command) error {
					dir, err := r.resolveConfigsDir(c)
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
			r.Audit,
		),
	}
}

func (r *Runner) ecoWorldSnapshotCommand() *cli.Command {
	return &cli.Command{
		Name:  "snapshot",
		Usage: "Copy Configs/WorldGenerator.eco to --target.",
		Flags: []cli.Flag{
			configsDirFlag(),
			&cli.StringFlag{Name: "target", Usage: "destination file path", Required: true},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "eco.world.snapshot",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--configs-dir": c.String("configs-dir"),
						"--target":      c.String("target"),
					}, nil
				},
				Action: func(_ context.Context, c *cli.Command) error {
					dir, err := r.resolveConfigsDir(c)
					if err != nil {
						return err
					}
					target := expandTilde(c.String("target"))
					if err := eco.Snapshot(dir, target); err != nil {
						return err
					}
					fmt.Fprintf(os.Stderr, "wrote snapshot to %s\n", target)
					return nil
				},
			},
			r.Audit,
		),
	}
}

// ecoRemote returns a cli.ActionFunc that runs the given argv on kai-server
// through pkg/ssh (golang.org/x/crypto/ssh under the hood, no ssh
// subprocess). The remote command is composed as a single space-joined
// string because crypto/ssh's session API takes one string (which the
// remote shell parses), the same shape ssh(1) uses. Every element of
// remoteArgv is hardcoded at compile time in this file. No user input
// reaches here, so no runtime metacharacter risk from this path. If we ever
// take user input, add policy.ValidateArgSlice at the entry point.
func (r *Runner) ecoRemote(remoteArgv []string) cli.ActionFunc {
	return func(ctx context.Context, _ *cli.Command) error {
		host := r.Cfg.KaiServer.TailscaleHost
		user := r.Cfg.KaiServer.SSHUser
		if host == "" || user == "" {
			return fmt.Errorf("eco: kai_server.tailscale_host or ssh_user not configured")
		}
		cmd := strings.Join(remoteArgv, " ")
		// Stream stdout/stderr live. Some eco verbs (`tail --follow`) run
		// indefinitely, so buffering the whole output is wrong.
		if err := r.SSH.Stream(ctx, host, user, cmd, os.Stdout, os.Stderr); err != nil {
			return fmt.Errorf("eco: remote %s: %w", remoteArgv[0], err)
		}
		return nil
	}
}

package main

import (
	"context"
	"fmt"

	"github.com/coilysiren/coily/pkg/policy"
	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

// ecoCommand wraps the eco game server which runs as a systemd unit on
// kai-server. All verbs ultimately shell out to
// `ssh <user>@<host> sudo <systemctl|journalctl> ... eco-server`.
// The ssh target is taken from embedded config (kai_server.tailscale_host
// and ssh_user).
func (r *Runner) ecoCommand() *cli.Command {
	return &cli.Command{
		Name:  "eco",
		Usage: "Operate the eco game server (systemd unit on kai-server).",
		Description: `eco wraps systemctl/journalctl calls against the eco-server unit that runs
on kai-server. Destructive verbs (restart, stop) require a confirmation
token. Reads (status, tail) do not.`,
		Commands: []*cli.Command{
			r.ecoStatusCommand(),
			r.ecoTailCommand(),
			r.ecoRestartCommand(),
			r.ecoStopCommand(),
			r.ecoStartCommand(),
		},
	}
}

func (r *Runner) ecoStatusCommand() *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "Print systemctl status eco-server.",
		Action: verb.Wrap(
			verb.Spec{
				Name:   "eco.status",
				Kind:   policy.ReadOnly,
				Action: r.ecoRemote([]string{"sudo", "systemctl", "status", "eco-server", "--no-pager"}),
			},
			r.Verifier,
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
		Action: verb.Wrap(
			verb.Spec{
				Name: "eco.tail",
				Kind: policy.ReadOnly,
				Action: func(ctx context.Context, c *cli.Command) error {
					args := []string{"sudo", "journalctl", "-u", "eco-server", "-n", fmt.Sprint(c.Int("lines"))}
					if c.Bool("follow") {
						args = append(args, "-f")
					}
					return r.ecoRemote(args)(ctx, c)
				},
			},
			r.Verifier,
			r.Audit,
		),
	}
}

func (r *Runner) ecoRestartCommand() *cli.Command {
	return &cli.Command{
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
				Action: r.ecoRemote([]string{"sudo", "systemctl", "restart", "eco-server"}),
			},
			r.Verifier,
			r.Audit,
		),
	}
}

func (r *Runner) ecoStopCommand() *cli.Command {
	return &cli.Command{
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
				Action: r.ecoRemote([]string{"sudo", "systemctl", "stop", "eco-server"}),
			},
			r.Verifier,
			r.Audit,
		),
	}
}

func (r *Runner) ecoStartCommand() *cli.Command {
	return &cli.Command{
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
				Action: r.ecoRemote([]string{"sudo", "systemctl", "start", "eco-server"}),
			},
			r.Verifier,
			r.Audit,
		),
	}
}

// ecoRemote returns a cli.ActionFunc that ssh's into kai-server and runs the
// given argv (as a single composed remote command, since ssh's last
// positional is passed to a remote shell). Every element of remoteArgv is
// hardcoded at compile time. No user input reaches here, so no runtime
// metacharacter risk from this path.
func (r *Runner) ecoRemote(remoteArgv []string) cli.ActionFunc {
	return func(ctx context.Context, _ *cli.Command) error {
		host := r.Cfg.KaiServer.TailscaleHost
		user := r.Cfg.KaiServer.SSHUser
		if host == "" || user == "" {
			return fmt.Errorf("eco: kai_server.tailscale_host or ssh_user not configured")
		}
		target := user + "@" + host
		// ssh takes the remote command as its last argv element. We compose
		// the remote command as space-joined because all remoteArgv elements
		// are compile-time constants in this file. If we ever take user
		// input here, add policy.ValidateArgSlice at the entry point.
		argv := append([]string{target}, remoteArgv...)
		return r.Runner.Exec(ctx, "ssh", argv...)
	}
}

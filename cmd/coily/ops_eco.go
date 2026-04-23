package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/coilysiren/coily/pkg/policy"
	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

// ecoMutatingScope is the required scope for any eco systemd state change.
// restart/stop/start are all writes, not deletes - no eco state is being
// destroyed, just re-cycled. coily-native verbs follow the same scope shape
// as pass-throughs (`<binary>.<service>:<bucket>`) with binary "coily".
const ecoMutatingScope = "coily.eco:write"

// ecoCommand wraps the eco game server which runs as a systemd unit on
// kai-server. All verbs run a `sudo <systemctl|journalctl> ... eco-server`
// command on kai-server through pkg/ssh, which wraps
// golang.org/x/crypto/ssh. No ssh subprocess is spawned. The ssh target is
// taken from embedded config (kai_server.tailscale_host and ssh_user).
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
				Scope:  "coily.eco:read",
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
				Name:  "eco.tail",
				Kind:  policy.ReadOnly,
				Scope: "coily.eco:read",
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
			&cli.StringFlag{Name: "token", Usage: "confirmation token scoped to coily.eco:write"},
		},
		Action: verb.Wrap(
			verb.Spec{
				Name:  "eco.restart",
				Kind:  policy.Mutating,
				Scope: ecoMutatingScope,
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
			&cli.StringFlag{Name: "token", Usage: "confirmation token scoped to coily.eco:write"},
		},
		Action: verb.Wrap(
			verb.Spec{
				Name:  "eco.stop",
				Kind:  policy.Mutating,
				Scope: ecoMutatingScope,
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
			&cli.StringFlag{Name: "token", Usage: "confirmation token scoped to coily.eco:write"},
		},
		Action: verb.Wrap(
			verb.Spec{
				Name:  "eco.start",
				Kind:  policy.Mutating,
				Scope: ecoMutatingScope,
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

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

func init() { registerCommand(ecoCmd) }

// ecoCmd wraps the eco game server which runs as a systemd unit on
// kai-server. All verbs run a `sudo <systemctl|journalctl> ... eco-server`
// command on kai-server through pkg/ssh, which wraps
// golang.org/x/crypto/ssh. No ssh subprocess is spawned. The ssh target
// is taken from embedded config (kai_server.tailscale_host and
// ssh_user).
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

// ecoRemote returns a cli.ActionFunc that runs the given argv on
// kai-server through pkg/ssh. The remote command is composed as a single
// space-joined string because crypto/ssh's session API takes one string
// (which the remote shell parses), the same shape ssh(1) uses. Every
// element of remoteArgv is hardcoded at compile time in this file. No
// user input reaches here, so no runtime metacharacter risk from this
// path. If we ever take user input, add policy.ValidateArgSlice at the
// entry point.
func ecoRemote(remoteArgv []string) cli.ActionFunc {
	return func(ctx context.Context, _ *cli.Command) error {
		rt := getRuntime()
		host := rt.cfg.KaiServer.TailscaleHost
		user := rt.cfg.KaiServer.SSHUser
		if host == "" || user == "" {
			return fmt.Errorf("eco: kai_server.tailscale_host or ssh_user not configured")
		}
		cmd := strings.Join(remoteArgv, " ")
		// Stream stdout/stderr live. Some eco verbs (`tail --follow`) run
		// indefinitely, so buffering the whole output is wrong.
		err := rt.ssh.Stream(ctx, host, user, cmd, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("eco: remote %s: %w", remoteArgv[0], err)
		}
		return nil
	}
}

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

// sirensDiscordOpsCommand wraps the sirens-discord-ops bot which runs as a
// systemd unit on kai-server. Same shape as ecoCommand: status/tail/start/
// stop/restart routed through pkg/ssh against kai-server. Verb names use
// the full repo name with no shortcut so audit lines are unambiguous.
//
// The bot has its own auto-update timer (sirens-discord-ops-update.timer)
// that polls origin/main and restarts on change. These verbs are for
// out-of-band ops (manual restart after a config change, status checks,
// log tails) and are not part of the deploy path.
func (r *Runner) sirensDiscordOpsCommand() *cli.Command {
	return &cli.Command{
		Name:  "sirens-discord-ops",
		Usage: "Operate the sirens-discord-ops bot (systemd unit on kai-server).",
		Description: `sirens-discord-ops wraps systemctl/journalctl calls against the
sirens-discord-ops unit that runs on kai-server.

The bot auto-updates from origin/main via a paired systemd timer, so
restart is rarely needed for code changes. Use these verbs for config
rotations (SSM-backed env vars) or diagnostic access.`,
		Commands: []*cli.Command{
			r.sirensDiscordOpsStatusCommand(),
			r.sirensDiscordOpsTailCommand(),
			r.sirensDiscordOpsRestartCommand(),
			r.sirensDiscordOpsStopCommand(),
			r.sirensDiscordOpsStartCommand(),
		},
	}
}

func (r *Runner) sirensDiscordOpsStatusCommand() *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "Print systemctl status sirens-discord-ops.",
		Action: verb.Wrap(
			verb.Spec{
				Name:   "sirens-discord-ops.status",
				Action: r.sirensDiscordOpsRemote([]string{"sudo", "systemctl", "status", "sirens-discord-ops", "--no-pager"}),
			},
			r.Audit,
		),
	}
}

func (r *Runner) sirensDiscordOpsTailCommand() *cli.Command {
	return &cli.Command{
		Name:  "tail",
		Usage: "Tail sirens-discord-ops journal logs (journalctl -u sirens-discord-ops -f).",
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
				Name: "sirens-discord-ops.tail",
				Action: func(ctx context.Context, c *cli.Command) error {
					args := []string{"journalctl", "-u", "sirens-discord-ops", "-n", fmt.Sprint(c.Int("lines"))}
					if c.Bool("follow") {
						args = append(args, "-f")
					}
					return r.sirensDiscordOpsRemote(args)(ctx, c)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) sirensDiscordOpsRestartCommand() *cli.Command {
	return &cli.Command{
		Name:  "restart",
		Usage: "Restart the sirens-discord-ops systemd unit.",
		Action: verb.Wrap(
			verb.Spec{
				Name:   "sirens-discord-ops.restart",
				Action: r.sirensDiscordOpsRemote([]string{"sudo", "systemctl", "restart", "sirens-discord-ops"}),
			},
			r.Audit,
		),
	}
}

func (r *Runner) sirensDiscordOpsStopCommand() *cli.Command {
	return &cli.Command{
		Name:  "stop",
		Usage: "Stop the sirens-discord-ops systemd unit.",
		Action: verb.Wrap(
			verb.Spec{
				Name:   "sirens-discord-ops.stop",
				Action: r.sirensDiscordOpsRemote([]string{"sudo", "systemctl", "stop", "sirens-discord-ops"}),
			},
			r.Audit,
		),
	}
}

func (r *Runner) sirensDiscordOpsStartCommand() *cli.Command {
	return &cli.Command{
		Name:  "start",
		Usage: "Start the sirens-discord-ops systemd unit.",
		Action: verb.Wrap(
			verb.Spec{
				Name:   "sirens-discord-ops.start",
				Action: r.sirensDiscordOpsRemote([]string{"sudo", "systemctl", "start", "sirens-discord-ops"}),
			},
			r.Audit,
		),
	}
}

// sirensDiscordOpsRemote mirrors ecoRemote: every element of remoteArgv is
// hardcoded at compile time, so no runtime metacharacter risk on the SSH
// path. If we ever take user input here, add policy.ValidateArgSlice at
// the entry point.
func (r *Runner) sirensDiscordOpsRemote(remoteArgv []string) cli.ActionFunc {
	return func(ctx context.Context, _ *cli.Command) error {
		host := r.Cfg.KaiServer.TailscaleHost
		user := r.Cfg.KaiServer.SSHUser
		if host == "" || user == "" {
			return fmt.Errorf("sirens-discord-ops: kai_server.tailscale_host or ssh_user not configured")
		}
		cmd := strings.Join(remoteArgv, " ")
		if err := r.SSH.Stream(ctx, host, user, cmd, os.Stdout, os.Stderr); err != nil {
			return fmt.Errorf("sirens-discord-ops: remote %s: %w", remoteArgv[0], err)
		}
		return nil
	}
}

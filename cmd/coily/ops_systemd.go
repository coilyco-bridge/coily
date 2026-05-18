package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// systemdUnit describes one kai-server systemd unit wrapped as a coily
// subcommand tree. The five verbs (status/tail/restart/stop/start) mirror
// the pattern set by `coily eco` and port the per-unit invoke tasks from
// infrastructure/src/{backend,core_keeper,icarus}.py.
//
// Field notes:
//   - VerbName is the top-level coily verb: `coily backend`, `coily icarus`.
//   - UnitName is the systemd unit id on kai-server: "coilysiren-backend",
//     "icarus-server", etc.
//   - RestartDaemonReload runs `systemctl daemon-reload` before `restart`,
//     matching the icarus/backend invoke tasks that edit unit files.
//   - StartEnables / StopDisables add `systemctl enable|disable` after the
//     transition verb, matching the core-keeper/icarus invoke tasks.
type systemdUnit struct {
	VerbName            string
	UnitName            string
	Description         string
	RestartDaemonReload bool
	StartEnables        bool
	StopDisables        bool
}

func (r *Runner) systemdUnitCommand(u systemdUnit) *cli.Command {
	return &cli.Command{
		Name:  u.VerbName,
		Usage: fmt.Sprintf("Operate the %s systemd unit on kai-server.", u.UnitName),
		Description: fmt.Sprintf(`%s wraps systemctl/journalctl calls against the %s unit
that runs on kai-server. Every call goes through cli-guard/ssh; no ssh
subprocess is spawned.`, u.VerbName, u.UnitName),
		Commands: []*cli.Command{
			r.systemdStatus(u),
			r.systemdTail(u),
			r.systemdRestart(u),
			r.systemdStop(u),
			r.systemdStart(u),
		},
	}
}

func (r *Runner) systemdStatus(u systemdUnit) *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: fmt.Sprintf("Print systemctl status %s.", u.UnitName),
		Action: r.WrapVerb(
			verb.Spec{
				Name:   u.VerbName + ".status",
				Action: r.systemdRemote([][]string{{"sudo", "systemctl", "status", u.UnitName, "--no-pager"}}),
			},
			r.Audit,
		),
	}
}

func (r *Runner) systemdTail(u systemdUnit) *cli.Command {
	return &cli.Command{
		Name:  "tail",
		Usage: fmt.Sprintf("Tail %s journal logs (journalctl -u %s -f).", u.UnitName, u.UnitName),
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "lines", Usage: "number of lines of history before tailing", Value: 200},
			&cli.BoolFlag{Name: "follow", Usage: "keep tailing after initial history", Value: true},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: u.VerbName + ".tail",
				Action: func(ctx context.Context, c *cli.Command) error {
					// No sudo: kai is in the adm group, which has read access
					// to /var/log/journal/ on Ubuntu.
					args := []string{"journalctl", "-u", u.UnitName, "-n", fmt.Sprint(c.Int("lines"))}
					if c.Bool("follow") {
						args = append(args, "-f")
					}
					return r.systemdRemote([][]string{args})(ctx, c)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) systemdRestart(u systemdUnit) *cli.Command {
	var argvs [][]string
	if u.RestartDaemonReload {
		argvs = append(argvs, []string{"sudo", "systemctl", "daemon-reload"})
	}
	argvs = append(argvs, []string{"sudo", "systemctl", "restart", u.UnitName})
	return &cli.Command{
		Name:  "restart",
		Usage: fmt.Sprintf("Restart the %s unit.", u.UnitName),
		Action: r.WrapVerb(
			verb.Spec{
				Name:   u.VerbName + ".restart",
				Action: r.systemdRemote(argvs),
			},
			r.Audit,
		),
	}
}

func (r *Runner) systemdStop(u systemdUnit) *cli.Command {
	argvs := [][]string{{"sudo", "systemctl", "stop", u.UnitName}}
	if u.StopDisables {
		argvs = append(argvs, []string{"sudo", "systemctl", "disable", u.UnitName})
	}
	return &cli.Command{
		Name:  "stop",
		Usage: fmt.Sprintf("Stop the %s unit.", u.UnitName),
		Action: r.WrapVerb(
			verb.Spec{
				Name:   u.VerbName + ".stop",
				Action: r.systemdRemote(argvs),
			},
			r.Audit,
		),
	}
}

func (r *Runner) systemdStart(u systemdUnit) *cli.Command {
	argvs := [][]string{{"sudo", "systemctl", "start", u.UnitName}}
	if u.StartEnables {
		argvs = append(argvs, []string{"sudo", "systemctl", "enable", u.UnitName})
	}
	return &cli.Command{
		Name:  "start",
		Usage: fmt.Sprintf("Start the %s unit.", u.UnitName),
		Action: r.WrapVerb(
			verb.Spec{
				Name:   u.VerbName + ".start",
				Action: r.systemdRemote(argvs),
			},
			r.Audit,
		),
	}
}

// systemdRemote runs one or more argv lines on kai-server in sequence.
// When invoked on kai-server itself (detected via hostNameMatches), the
// argvs are exec'd locally to skip an ssh-to-self that doesn't carry
// authentication in non-interactive environments like the
// claude-remote-control daemon. Per coilysiren/coily#135.
//
// Local mode still needs sudo for mutating verbs; the operator must
// have a sudoers entry for each systemctl verb (tracked in the
// infrastructure repo). Read-only verbs (status, tail) work without
// sudo on a host where kai is in the adm group.
//
// Elements come from compile-time literals; no user input reaches this
// path.
func (r *Runner) systemdRemote(argvs [][]string) cli.ActionFunc {
	return func(ctx context.Context, _ *cli.Command) error {
		host := r.Cfg.KaiServer.TailscaleHost
		if host == "" {
			return fmt.Errorf("systemd: kai_server.tailscale_host not configured")
		}
		if hostIsLocal(host) {
			return r.systemdRemoteLocal(ctx, argvs)
		}
		user := r.Cfg.KaiServer.SSHUser
		if user == "" {
			return fmt.Errorf("systemd: kai_server.ssh_user not configured")
		}
		parts := make([]string, 0, len(argvs))
		for _, a := range argvs {
			parts = append(parts, strings.Join(a, " "))
		}
		cmd := strings.Join(parts, " && ")
		if err := r.SSH.Stream(ctx, host, user, cmd, os.Stdout, os.Stderr); err != nil {
			return fmt.Errorf("systemd: remote exec: %w", err)
		}
		return nil
	}
}

// systemdRemoteLocal exec's each argv directly on the local host using
// the runner's shell.Runner. Stdout/stderr forward to the operator's
// terminal. Stops at the first failure to mirror the `&&` chaining the
// ssh path uses.
func (r *Runner) systemdRemoteLocal(ctx context.Context, argvs [][]string) error {
	for _, a := range argvs {
		if len(a) == 0 {
			continue
		}
		if err := r.Runner.Exec(ctx, a[0], a[1:]...); err != nil {
			return fmt.Errorf("systemd: local exec %s: %w", strings.Join(a, " "), err)
		}
	}
	return nil
}

// hostIsLocal reports whether target names the host this binary is
// running on. Matches on the leading hostname segment so target=
// "kai-server" matches local "kai-server", "kai-server.local",
// "kai-server.tail-scale.ts.net", etc. False on os.Hostname errors so
// the safe default is to ssh (the original behavior).
func hostIsLocal(target string) bool {
	if target == "" {
		return false
	}
	h, err := os.Hostname()
	if err != nil {
		return false
	}
	local := strings.SplitN(h, ".", 2)[0]
	want := strings.SplitN(target, ".", 2)[0]
	return strings.EqualFold(local, want)
}

func (r *Runner) coreKeeperCommand() *cli.Command {
	return r.systemdUnitCommand(systemdUnit{
		VerbName:     "core-keeper",
		UnitName:     "core-keeper-server",
		StartEnables: true,
		StopDisables: true,
	})
}

func (r *Runner) icarusCommand() *cli.Command {
	return r.systemdUnitCommand(systemdUnit{
		VerbName:            "icarus",
		UnitName:            "icarus-server",
		RestartDaemonReload: true,
		StartEnables:        true,
		StopDisables:        true,
	})
}

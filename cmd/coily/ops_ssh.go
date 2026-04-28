package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

// sshCommand is the named-verb ssh wrapper for kai-server. Free-form
// `coily ssh exec <argv>` is intentionally absent: it would let any holder
// of the binary run arbitrary commands as kai on the homelab, which is the
// exact escape the lockdown that blocks raw `ssh` is meant to close. The
// only ways to reach the remote shell through coily are:
//
//   - coily ssh copy                 sftp upload (constrained to file xfer)
//   - coily ssh systemctl <verb>     fixed verb tree mirroring systemctl
//   - coily ssh rm-unit <unit>       remove a /etc/systemd/system unit file
//   - coily ssh git <verb> <path>    fixed verb tree of read/fast-forward git ops
//   - coily ssh deploy <name>        allowlisted (repo, install-script) pair; fast-forwards source then runs the installer as root via sudo -n with an interactive /dev/tty fallback
//
// All of them take fixed argv shapes; nothing inside the wrapper joins user
// strings into a remote shell command. For the genuinely one-off case where
// none of the named verbs fit, drop out to raw `ssh kai@kai-server` (which
// the lockdown denies, requiring an explicit override) instead of widening
// this surface.
//
// The host/user defaults come from kai_server.tailscale_host and ssh_user
// in embedded config and are exposed as `--host` / `--user` flags on each
// leaf so the same wrapper works against a sibling box without rebuilding.
func (r *Runner) sshCommand() *cli.Command {
	return &cli.Command{
		Name:  "ssh",
		Usage: "Run named operations on kai-server over ssh.",
		Description: `ssh wraps golang.org/x/crypto/ssh. The default target is
kai_server.tailscale_host as kai_server.ssh_user (override per-call with
--host / --user). Free-form remote exec was removed in favor of named
verbs; see the package doc on ops_ssh.go for the rationale.`,
		Commands: []*cli.Command{
			r.sshCopyCommand(),
			r.sshSystemctlCommand(),
			r.sshRmUnitCommand(),
			r.sshGitCommand(),
			r.sshDeployCommand(),
		},
	}
}

// sshHostUserFlags returns the flag pair every ssh leaf accepts. Defaults
// come from embedded config so the common case is flag-free.
func (r *Runner) sshHostUserFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  "host",
			Usage: "remote ssh host (defaults to kai_server.tailscale_host)",
			Value: r.Cfg.KaiServer.TailscaleHost,
		},
		&cli.StringFlag{
			Name:  "user",
			Usage: "remote ssh user (defaults to kai_server.ssh_user)",
			Value: r.Cfg.KaiServer.SSHUser,
		},
	}
}

func sshTarget(c *cli.Command) (host, user string, err error) {
	host = c.String("host")
	user = c.String("user")
	if host == "" || user == "" {
		return "", "", fmt.Errorf("ssh: --host and --user must resolve (config or flag)")
	}
	return host, user, nil
}

func (r *Runner) sshCopyCommand() *cli.Command {
	return &cli.Command{
		Name:      "copy",
		Usage:     "Upload a local file to the remote via sftp.",
		ArgsUsage: "<local-path> <remote-path>",
		Flags:     r.sshHostUserFlags(),
		Action: verb.Wrap(
			verb.Spec{
				Name: "ssh.copy",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
							"--host": c.String("host"),
							"--user": c.String("user"),
						},
						c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					argv := c.Args().Slice()
					if len(argv) != 2 {
						return fmt.Errorf("ssh copy: need <local-path> <remote-path>, got %d arg(s)", len(argv))
					}
					host, user, err := sshTarget(c)
					if err != nil {
						return err
					}
					if err := r.SSH.CopyTo(ctx, host, user, argv[0], argv[1]); err != nil {
						return fmt.Errorf("ssh copy: %w", err)
					}
					fmt.Fprintf(os.Stderr, "uploaded %s -> %s:%s\n", argv[0], host, argv[1])
					return nil
				},
			},
			r.Audit,
		),
	}
}

// systemctlVerbs is the closed set of systemctl actions exposed through
// coily ssh systemctl. Each one has a fixed argv shape that takes either
// no argument (daemon-reload) or exactly one unit name. New verbs land
// here, not as a free-form pass-through.
var systemctlVerbs = []struct {
	Name      string
	Usage     string
	NeedsUnit bool
	// Argv builds the remote argv. Receives the unit name (or "" when
	// !NeedsUnit). Output is appended after `sudo`.
	Argv func(unit string) []string
}{
	{"status", "Print systemctl status of <unit>.", true, func(u string) []string { return []string{"systemctl", "status", u, "--no-pager"} }},
	{"start", "Start <unit>.", true, func(u string) []string { return []string{"systemctl", "start", u} }},
	{"stop", "Stop <unit>.", true, func(u string) []string { return []string{"systemctl", "stop", u} }},
	{"restart", "Restart <unit>.", true, func(u string) []string { return []string{"systemctl", "restart", u} }},
	{"enable", "Enable <unit>.", true, func(u string) []string { return []string{"systemctl", "enable", u} }},
	{"disable", "Disable <unit>.", true, func(u string) []string { return []string{"systemctl", "disable", u} }},
	{"daemon-reload", "Run systemctl daemon-reload.", false, func(string) []string { return []string{"systemctl", "daemon-reload"} }},
}

func (r *Runner) sshSystemctlCommand() *cli.Command {
	cmds := make([]*cli.Command, 0, len(systemctlVerbs))
	for _, v := range systemctlVerbs {
		cmds = append(cmds, r.sshSystemctlVerb(v.Name, v.Usage, v.NeedsUnit, v.Argv))
	}
	return &cli.Command{
		Name:        "systemctl",
		Usage:       "Run a fixed-shape systemctl verb on the remote.",
		Description: "Each leaf maps to one systemctl call (sudo-prefixed). Mirrors systemctl's own verb names; no free-form passthrough.",
		Commands:    cmds,
	}
}

func (r *Runner) sshSystemctlVerb(name, usage string, needsUnit bool, build func(string) []string) *cli.Command {
	argsUsage := "<unit>"
	if !needsUnit {
		argsUsage = ""
	}
	return &cli.Command{
		Name:      name,
		Usage:     usage,
		ArgsUsage: argsUsage,
		Flags:     r.sshHostUserFlags(),
		Action: verb.Wrap(
			verb.Spec{
				Name: "ssh.systemctl." + name,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--host": c.String("host"),
						"--user": c.String("user"),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					var unit string
					if needsUnit {
						if c.Args().Len() != 1 {
							return fmt.Errorf("ssh systemctl %s: need exactly one <unit> arg, got %d", name, c.Args().Len())
						}
						unit = c.Args().First()
						if err := validateUnitName(unit); err != nil {
							return err
						}
					} else if c.Args().Len() != 0 {
						return fmt.Errorf("ssh systemctl %s: takes no args, got %d", name, c.Args().Len())
					}
					host, user, err := sshTarget(c)
					if err != nil {
						return err
					}
					argv := append([]string{"sudo"}, build(unit)...)
					return r.SSH.Stream(ctx, host, user, strings.Join(argv, " "), os.Stdout, os.Stderr)
				},
			},
			r.Audit,
		),
	}
}

// sshRmUnitCommand removes a /etc/systemd/system unit file and reloads the
// daemon. Captures the cleanup pattern from the issue without re-opening
// free-form rm: the path is fully derived from the validated unit name.
func (r *Runner) sshRmUnitCommand() *cli.Command {
	return &cli.Command{
		Name:      "rm-unit",
		Usage:     "Remove /etc/systemd/system/<unit>.service and reload systemd.",
		ArgsUsage: "<unit>",
		Flags:     r.sshHostUserFlags(),
		Action: verb.Wrap(
			verb.Spec{
				Name: "ssh.rm-unit",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--host": c.String("host"),
						"--user": c.String("user"),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 1 {
						return fmt.Errorf("ssh rm-unit: need exactly one <unit> arg, got %d", c.Args().Len())
					}
					unit := c.Args().First()
					if err := validateUnitName(unit); err != nil {
						return err
					}
					host, user, err := sshTarget(c)
					if err != nil {
						return err
					}
					path := "/etc/systemd/system/" + unit
					if !strings.HasSuffix(path, ".service") {
						path += ".service"
					}
					argv := []string{
						"sudo", "rm", "-f", path, "&&",
						"sudo", "systemctl", "daemon-reload",
					}
					return r.SSH.Stream(ctx, host, user, strings.Join(argv, " "), os.Stdout, os.Stderr)
				},
			},
			r.Audit,
		),
	}
}

// validateUnitName rejects anything that isn't a sane systemd unit name.
// systemd allows [A-Za-z0-9:_.\\-] in names; we add a length cap and
// disallow leading "-" to avoid argv-as-flag confusion.
func validateUnitName(unit string) error {
	if unit == "" {
		return fmt.Errorf("ssh: unit name is empty")
	}
	if len(unit) > 128 {
		return fmt.Errorf("ssh: unit name too long")
	}
	if strings.HasPrefix(unit, "-") {
		return fmt.Errorf("ssh: unit name must not start with '-'")
	}
	for _, r := range unit {
		switch {
		case r >= 'A' && r <= 'Z',
			r >= 'a' && r <= 'z',
			r >= '0' && r <= '9',
			r == ':', r == '_', r == '.', r == '-', r == '@':
			// ok
		default:
			return fmt.Errorf("ssh: unit name contains invalid character %q", r)
		}
	}
	return nil
}

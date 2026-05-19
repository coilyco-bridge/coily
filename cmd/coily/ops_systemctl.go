package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// systemctlVerbs is the closed set of systemctl actions exposed through
// `coily systemctl`. Each entry has a fixed argv shape that takes either
// no argument (daemon-reload) or exactly one unit name. New verbs land
// here, not as a free-form pass-through.
//
// NoSudo flags read-only verbs that don't need privilege. systemctl
// status reads cached state from systemd; running it under sudo trips
// "a terminal is required to read the password" on non-tty sessions
// even though the read itself is unprivileged (coilysiren/coily#144).
// Mutating verbs (start/stop/restart/enable/disable/daemon-reload) stay
// sudo-prefixed because they write to runtime state or
// /etc/systemd/system.
var systemctlVerbs = []struct {
	Name      string
	Usage     string
	NeedsUnit bool
	NoSudo    bool
	// Argv builds the remote argv. Receives the unit name (or "" when
	// !NeedsUnit). Output is appended after `sudo` unless NoSudo is set.
	Argv func(unit string) []string
}{
	{"status", "Print systemctl status of <unit>.", true, true, func(u string) []string { return []string{"systemctl", "status", u, "--no-pager"} }},
	{"start", "Start <unit>.", true, false, func(u string) []string { return []string{"systemctl", "start", u} }},
	{"stop", "Stop <unit>.", true, false, func(u string) []string { return []string{"systemctl", "stop", u} }},
	{"restart", "Restart <unit>.", true, false, func(u string) []string { return []string{"systemctl", "restart", u} }},
	{"enable", "Enable <unit>.", true, false, func(u string) []string { return []string{"systemctl", "enable", u} }},
	{"disable", "Disable <unit>.", true, false, func(u string) []string { return []string{"systemctl", "disable", u} }},
	{"daemon-reload", "Run systemctl daemon-reload.", false, false, func(string) []string { return []string{"systemctl", "daemon-reload"} }},
}

// validateUnitName rejects anything that isn't a sane systemd unit name.
// systemd allows [A-Za-z0-9:_.\-@] in names; we add a length cap and
// disallow leading "-" to avoid argv-as-flag confusion.
func validateUnitName(unit string) error {
	if unit == "" {
		return fmt.Errorf("systemctl: unit name is empty")
	}
	if len(unit) > 128 {
		return fmt.Errorf("systemctl: unit name too long")
	}
	if strings.HasPrefix(unit, "-") {
		return fmt.Errorf("systemctl: unit name must not start with '-'")
	}
	for _, r := range unit {
		switch {
		case r >= 'A' && r <= 'Z',
			r >= 'a' && r <= 'z',
			r >= '0' && r <= '9',
			r == ':', r == '_', r == '.', r == '-', r == '@':
			// ok
		default:
			return fmt.Errorf("systemctl: unit name contains invalid character %q", r)
		}
	}
	return nil
}

// systemctlCommand is the local-execution systemctl verb tree. The
// intended call-site is the remote coily that local
// `coily ssh <alias> -- coily systemctl <verb> <unit>` dispatches to.
// Direct `coily systemctl ...` use from a Mac is a no-op without local
// systemd.
//
// Sudo discipline (coilysiren/coily#203): mutating verbs self-elevate by
// re-execing the coily binary under outer sudo, then run `systemctl ...`
// directly inside the root call. This relies on a broad
// `(ALL) NOPASSWD: <coily-path>` sudoers rule on the host: coily is the
// security boundary, so per-unit sudoers carveouts duplicate the gate
// and drift. Status reads cached systemd state unprivileged (sudo trips
// a tty prompt on non-tty sessions, coilysiren/coily#144).
func (r *Runner) systemctlCommand() *cli.Command {
	cmds := make([]*cli.Command, 0, len(systemctlVerbs))
	for _, v := range systemctlVerbs {
		cmds = append(cmds, r.systemctlVerb(v.Name, v.Usage, v.NeedsUnit, v.NoSudo, v.Argv))
	}
	return &cli.Command{
		Name:  "systemctl",
		Usage: "Run a fixed-shape systemctl verb on the local host.",
		Description: `Closed verb set (status/start/stop/restart/enable/
disable/daemon-reload). Status runs unprivileged (sudo trips a tty
prompt on non-tty sessions, coilysiren/coily#144); mutating verbs are
sudo-prefixed. Intended for the remote side of
` + "`coily ssh <alias> -- coily systemctl <verb> <unit>`" + `.`,
		Commands: cmds,
	}
}

// buildSelfElevateArgv composes the outer-sudo re-exec argv. Pure helper
// so the shape is unit-testable without a real sudo. Pinned form:
//
//	sudo --non-interactive <coilyPath> systemctl <verb> [<unit>]
//
// No inner sudo, no shell, no `sudo -i`. --non-interactive is explicit so
// a host missing the NOPASSWD grant fails fast instead of dangling on a
// hidden password prompt (the coily#203 motivating bug).
func buildSelfElevateArgv(coilyPath, name, unit string, needsUnit bool) []string {
	argv := []string{"sudo", "--non-interactive", coilyPath, "systemctl", name}
	if needsUnit {
		argv = append(argv, unit)
	}
	return argv
}

// systemctlSelfElevate re-execs the coily binary under outer sudo so the
// inner systemctl call can run as root without a per-unit sudoers rule.
// The audit row for the outer invocation is already in flight by the
// time we get here (WrapVerb wraps Action); the inner root coily writes
// its own row when it runs. Two rows, both reconstructable, mirror the
// privilege transition.
//
// Path resolution: exec.LookPath("coily") so we pick the PATH-resolved
// symlink (e.g. /home/linuxbrew/.linuxbrew/bin/coily on kai-server) that
// the sudoers rule strict-matches. Falls back to os.Executable() if
// LookPath misses, with a clear error if both fail.
func (r *Runner) systemctlSelfElevate(ctx context.Context, name, unit string, needsUnit bool) error {
	coilyPath, err := exec.LookPath("coily")
	if err != nil {
		coilyPath, err = os.Executable()
		if err != nil {
			return fmt.Errorf("systemctl %s: locate coily binary for sudo re-exec: %w", name, err)
		}
	}
	argv := buildSelfElevateArgv(coilyPath, name, unit, needsUnit)
	if err := r.Runner.Exec(ctx, argv[0], argv[1:]...); err != nil {
		return fmt.Errorf("systemctl %s: sudo re-exec failed (does this host grant `NOPASSWD: %s` for the invoking user?): %w", name, coilyPath, err)
	}
	return nil
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
					if !noSudo && os.Geteuid() != 0 {
						return r.systemctlSelfElevate(ctx, name, unit, needsUnit)
					}
					return r.Runner.Exec(ctx, argv[0], argv[1:]...)
				},
			},
			r.Audit,
		),
	}
}

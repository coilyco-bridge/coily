package main

import (
	"context"
	"fmt"
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
// systemd. Sudo discipline: status reads cached systemd state
// unprivileged (sudo trips a tty prompt on non-tty sessions, per
// coilysiren/coily#144); mutating verbs are sudo-prefixed.
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
					if !noSudo {
						argv = append([]string{"sudo"}, argv...)
					}
					return r.Runner.Exec(ctx, argv[0], argv[1:]...)
				},
			},
			r.Audit,
		),
	}
}

package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// sshJournalctlCommand wraps `journalctl -u <unit> -n <N> --no-pager`
// over ssh. Mirrors the systemctl shape: takes a single validated unit
// name and one numeric flag. No free-form journalctl pass-through; if
// you need other flags, add them here as named flags so the argv
// remains a fixed shape.
func (r *Runner) sshJournalctlCommand() *cli.Command {
	return &cli.Command{
		Name:      "journalctl",
		Usage:     "Run journalctl -u <unit> -n <lines> --no-pager on the remote.",
		ArgsUsage: "<unit>",
		Flags: append(
			r.sshHostUserFlags(),
			&cli.IntFlag{
				Name:  "lines",
				Usage: "number of recent journal lines to print (-n)",
				Value: 50,
			},
		),
		Action: verb.Wrap(
			verb.Spec{
				Name: "ssh.journalctl",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
							"--host":  c.String("host"),
							"--user":  c.String("user"),
							"--lines": strconv.Itoa(c.Int("lines")),
						},
						c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 1 {
						return fmt.Errorf("ssh journalctl: need exactly one <unit> arg, got %d", c.Args().Len())
					}
					unit := c.Args().First()
					if err := validateUnitName(unit); err != nil {
						return err
					}
					lines := c.Int("lines")
					if lines < 1 || lines > 10000 {
						return fmt.Errorf("ssh journalctl: --lines must be in [1, 10000], got %d", lines)
					}
					host, user, err := sshTarget(c)
					if err != nil {
						return err
					}
					// No sudo: kai is in the adm group, which has read access
					// to /var/log/journal/ on Ubuntu. The systemctl wrapper
					// in ops_ssh_systemctl.go still uses sudo for writes.
					argv := []string{
						"journalctl",
						"-u", unit,
						"-n", strconv.Itoa(lines),
						"--no-pager",
					}
					return r.SSH.Stream(ctx, host, user, strings.Join(argv, " "), os.Stdout, os.Stderr)
				},
			},
			r.Audit,
		),
	}
}

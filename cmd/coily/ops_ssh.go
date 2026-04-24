package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

// sshCommand is the generic ssh wrapper for kai-server. Mirrors ssh(1)'s
// shape: `coily ssh exec <argv...>` runs argv as a single remote command,
// `coily ssh copy <local> <remote>` uploads via sftp. The host and user
// come from embedded config (kai_server.tailscale_host, ssh_user); arbitrary
// targets are intentionally not exposed, since the lockdown rule that blocks
// raw `ssh` is what makes this wrapper the right path.
//
// All work routes through pkg/ssh (golang.org/x/crypto/ssh). No ssh
// subprocess is spawned. Per pkg/ssh's contract, every positional arg is
// validated by policy.ValidateArgSlice via verb.Wrap before reaching the
// remote shell.
func (r *Runner) sshCommand() *cli.Command {
	return &cli.Command{
		Name:  "ssh",
		Usage: "Run commands or upload files to kai-server over ssh.",
		Description: `ssh wraps golang.org/x/crypto/ssh. The target host and user are taken
from embedded config (kai_server.tailscale_host, ssh_user); this wrapper
exists so the lockdown that blocks raw 'ssh' / 'scp' has a sanctioned path
for the kai-server case.`,
		Commands: []*cli.Command{
			r.sshExecCommand(),
			r.sshCopyCommand(),
		},
	}
}

func (r *Runner) sshExecCommand() *cli.Command {
	return &cli.Command{
		Name:      "exec",
		Usage:     "Run a command on kai-server. Streams stdout/stderr live.",
		ArgsUsage: "<command> [args...]",
		Action: verb.Wrap(
			verb.Spec{
				Name: "ssh.exec",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return nil, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					argv := c.Args().Slice()
					if len(argv) == 0 {
						return fmt.Errorf("ssh exec: no command supplied")
					}
					host, user, err := r.kaiServerTarget("ssh exec")
					if err != nil {
						return err
					}
					cmd := strings.Join(argv, " ")
					if err := r.SSH.Stream(ctx, host, user, cmd, os.Stdout, os.Stderr); err != nil {
						return fmt.Errorf("ssh exec: %w", err)
					}
					return nil
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) sshCopyCommand() *cli.Command {
	return &cli.Command{
		Name:      "copy",
		Usage:     "Upload a local file to kai-server via sftp.",
		ArgsUsage: "<local-path> <remote-path>",
		Action: verb.Wrap(
			verb.Spec{
				Name: "ssh.copy",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return nil, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					argv := c.Args().Slice()
					if len(argv) != 2 {
						return fmt.Errorf("ssh copy: need <local-path> <remote-path>, got %d arg(s)", len(argv))
					}
					local, remote := argv[0], argv[1]
					host, user, err := r.kaiServerTarget("ssh copy")
					if err != nil {
						return err
					}
					if err := r.SSH.CopyTo(ctx, host, user, local, remote); err != nil {
						return fmt.Errorf("ssh copy: %w", err)
					}
					fmt.Fprintf(os.Stderr, "uploaded %s -> %s:%s\n", local, host, remote)
					return nil
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) kaiServerTarget(verbName string) (host, user string, err error) {
	host = r.Cfg.KaiServer.TailscaleHost
	user = r.Cfg.KaiServer.SSHUser
	if host == "" || user == "" {
		return "", "", fmt.Errorf("%s: kai_server.tailscale_host or ssh_user not configured", verbName)
	}
	return host, user, nil
}

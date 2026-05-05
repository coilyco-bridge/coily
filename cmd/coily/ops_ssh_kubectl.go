package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

// sshKubectlCommand wraps `sudo k3s kubectl <args>` over ssh to kai-server.
// Mirrors the local `coily ops kubectl` passthrough: argv forwards verbatim,
// readonly-vs-mutator gating is enforced at the lockdown deny list (e.g.
// Bash(coily ssh kubectl get:*) allow / Bash(coily ssh kubectl apply:*)
// deny), not inside coily. Replaces the server-side k3s-readonly-kubectl
// wrapper.
//
// SkipFlagParsing is on so kubectl's own flags (-A, -n, -o, --context, ...)
// flow through verbatim. That means the standard sshHostUserFlags() pair
// can't be exposed here without colliding with kubectl flag names; host
// and user resolve from embedded config (kai_server.tailscale_host /
// ssh_user). For ad-hoc retargeting, use bare ssh.
//
// Each argv element is POSIX single-quoted before being joined into the
// remote command string, because golang.org/x/crypto/ssh's session.Run /
// .Start hands the command string to the remote login shell. Without
// quoting, kubectl args with shell metacharacters (jsonpath {.x.y},
// label selectors with commas/=, --selector 'a in (b,c)') would be
// re-interpreted by bash on kai-server.
func (r *Runner) sshKubectlCommand() *cli.Command {
	return &cli.Command{
		Name:            "kubectl",
		Usage:           "Run `sudo k3s kubectl <args>` on kai-server.",
		ArgsUsage:       "[kubectl args...]",
		SkipFlagParsing: true,
		Action: verb.Wrap(
			verb.Spec{
				Name: "ssh.kubectl",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return nil, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					host := r.Cfg.KaiServer.TailscaleHost
					user := r.Cfg.KaiServer.SSHUser
					if host == "" || user == "" {
						return fmt.Errorf("ssh kubectl: kai_server.tailscale_host / ssh_user must be set in config")
					}
					args := c.Args().Slice()
					if len(args) == 0 {
						return fmt.Errorf("ssh kubectl: need at least one kubectl arg")
					}
					parts := []string{"sudo", "k3s", "kubectl"}
					for _, a := range args {
						parts = append(parts, posixShellQuote(a))
					}
					return r.SSH.Stream(ctx, host, user, strings.Join(parts, " "), os.Stdout, os.Stderr)
				},
			},
			r.Audit,
		),
	}
}

// posixShellQuote wraps s in POSIX single quotes. Embedded single quotes
// are encoded as '\” (close-quote, escaped quote, reopen-quote). The
// result is safe to splice into a shell command line.
func posixShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

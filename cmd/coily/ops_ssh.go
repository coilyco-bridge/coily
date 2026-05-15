package main

import (
	"github.com/urfave/cli/v3"
)

// sshCommand is the free-form ssh passthrough surface. The local coily
// dials the alias's target via cli-guard/ssh and ships any coily argv
// after `--` to a remote coily, where the remote's own lockdown is the
// security boundary. See coilysiren/coily#187 for the design (NOPASSWD-
// for-coily sudoers stance, audit-row parent chain via --audit-parent).
//
// The per-verb wrappers (coily ssh systemctl, copy, deploy, git,
// journalctl, kubectl, fs, rm-unit) lived here through step 7 of #187
// and got deleted in step 8 once the passthrough was proven end-to-end
// on the systemctl call site (coilysiren/coily#191). Replacement form
// for every previous named verb is:
//
//	coily ssh <alias> -- coily <subcommand> <args>
//
// where <subcommand> is the local-execution sibling that the remote
// coily already exposes (e.g. `coily systemctl` for the old `coily ssh
// systemctl`, `coily ops kubectl` for the old `coily ssh kubectl`,
// etc.).
func (r *Runner) sshCommand() *cli.Command {
	return &cli.Command{
		Name:  "ssh",
		Usage: "Free-form passthrough to a configured host alias.",
		Description: `ssh wraps golang.org/x/crypto/ssh. Resolves <alias>
from .coily/coily.yaml ssh.targets and ships the args after -- to a
remote coily. Remote coily's lockdown is the security boundary, not
this wrapper. The local audit row pre-allocates an id and ships it
to the remote via --audit-parent so forensic walks (forward and
backward) line up. See coilysiren/coily#187 for the design.

Usage:

  coily ssh kai-server -- coily systemctl status eco-server
  coily ssh kai-server -- coily ops kubectl get pods -A
  coily ssh kai-server -- coily ops aws sts get-caller-identity`,
		Action: r.sshPassthroughAction,
	}
}

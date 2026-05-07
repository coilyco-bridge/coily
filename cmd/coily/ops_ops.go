package main

import "github.com/urfave/cli/v3"

// opsCommand is the umbrella for external-system pass-throughs (cloud,
// repo, cluster). Collapses three top-level verbs (aws, gh, kubectl)
// under one named group so the top-level surface stays small enough to
// describe in one breath, and so the group name itself signals "this is
// the privileged-op gate."
//
// Game-server pass-throughs live under `gaming` instead - server admin
// is a different mental category from cloud + repo + cluster ops.
//
// Audit verb names live under "ops.<bin>" (e.g. "ops.aws") so the log
// reflects the user-visible path. Old "aws" / "gh" / "kubectl" rows are
// not migrated.
func (r *Runner) opsCommand() *cli.Command {
	return &cli.Command{
		Name:  "ops",
		Usage: "External-system pass-throughs (aws, gh, kubectl).",
		Description: `ops is the umbrella for cloud + repo + cluster pass-throughs. Each
subcommand forwards verbatim to the underlying binary, gated by argv
shell-metacharacter rejection and audit logging.

  coily ops aws <args>      passthrough to aws
  coily ops gh <args>       passthrough to gh
  coily ops kubectl <args>  passthrough to kubectl

Game-server pass-throughs live under coily gaming instead.`,
		Commands: r.passthroughCommands(ptOps),
	}
}

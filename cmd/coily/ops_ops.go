package main

import "github.com/urfave/cli/v3"

// opsCommand is the umbrella for external-system integrations: CLI
// pass-throughs (aws, gh, kubectl) plus REST API wrappers (modio,
// discord, sentry, trello-api). Collapses them under one named
// group so the top-level surface stays small enough to describe in one
// breath, and so the group name itself signals "this is the
// privileged-op gate."
//
// Game-server pass-throughs live under `gaming` instead - server admin
// is a different mental category. Package-directory wrappers (glama,
// skillsmp) live under `pkg` - they are catalog/discovery surfaces, not
// privileged ops.
//
// Audit verb names live under "ops.<area>" so the log reflects the
// user-visible path. CLI pass-throughs use "ops.<bin>" (e.g. "ops.aws"),
// REST wrappers use "ops.<pkg>.<group>.<op>" (e.g. "ops.modio.games
// .get-games").
func (r *Runner) opsCommand() *cli.Command {
	cmds := r.passthroughCommands(ptOps)
	cmds = append(cmds,
		r.modioCommand(),
		r.discordCommand(),
		r.sentryCommand(),
		r.trelloCommand(),
		r.forgejoCommand(),
		r.claudeRemoteControlCommand(),
		r.personalDashboardCommand(),
	)
	return &cli.Command{
		Name:  "ops",
		Usage: "External-system integrations (CLI pass-throughs + REST wrappers).",
		Description: `ops is the umbrella for cloud + repo + cluster pass-throughs and the
REST-API wrappers. Pass-throughs forward verbatim to the underlying
binary; REST wrappers issue HTTP requests via SSM-resolved auth.

CLI pass-throughs:
  coily ops aws <args>      passthrough to aws
  coily ops gh <args>       passthrough to gh
  coily ops kubectl <args>  passthrough to kubectl

REST wrappers (one operation per subcommand, generated from each
service's OpenAPI spec by scripts/openapi-to-coily.py):
  coily ops modio    mod.io v1 (Eco mods)
  coily ops discord  Discord HTTP API (bot auth)
  coily ops sentry   Sentry Public API
  coily ops trello   Trello REST API (key+token auth)

systemd-unit wrappers on kai-server (non-game services):
  coily ops claude-remote-control {status,tail,start,stop,restart}

Game-server pass-throughs live under coily gaming instead.
Package-directory wrappers (glama, skillsmp) live under coily pkg.`,
		Commands: cmds,
	}
}

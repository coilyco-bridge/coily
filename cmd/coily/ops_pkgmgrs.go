package main

import "github.com/urfave/cli/v3"

// pkgCommand groups package-manager pass-throughs and package-directory
// REST wrappers under a single `coily pkg <tool>` namespace. The
// pass-through set itself lives in ptPkg in passthroughs.go; the REST
// wrappers (glama, skillsmp) are catalog/discovery surfaces for MCP
// servers and Claude skills.
func (r *Runner) pkgCommand() *cli.Command {
	cmds := r.passthroughCommands(ptPkg)
	cmds = append(cmds,
		r.pkgBrewCommand(),
		r.pkgScoopCommand(),
		r.glamaCommand(),
		r.skillsmpCommand(),
	)
	return &cli.Command{
		Name:  "pkg",
		Usage: "Package-manager pass-throughs + package-directory REST wrappers.",
		Description: `pkg groups the thin pass-through wrappers around language package
managers (pip, npm, cargo, brew, etc.) alongside the REST wrappers for
package directories (glama for MCP servers, skillsmp for Claude
skills). Each subcommand emits an audit record, so 'coily pkg pip
install foo' runs 'pip install foo' under coily's audit + scope rules.

Package-directory wrappers (read-only catalogs):
  coily pkg glama     Glama MCP directory + telemetry
  coily pkg skillsmp  skillsmp.com v1 read API for skill discovery`,
		Commands: cmds,
	}
}

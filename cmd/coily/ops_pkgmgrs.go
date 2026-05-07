package main

import "github.com/urfave/cli/v3"

// pkgCommand groups every package-manager pass-through under a single
// `coily pkg <tool>` namespace, e.g. `coily pkg pip install foo`. The set
// itself lives in ptPkg in passthroughs.go.
func (r *Runner) pkgCommand() *cli.Command {
	return &cli.Command{
		Name:  "pkg",
		Usage: "Audited pass-throughs for language package managers.",
		Description: `pkg groups the thin pass-through wrappers around language package
managers (pip, npm, cargo, brew, etc.). Each subcommand forwards its
arguments to the underlying binary while emitting an audit record, so
'coily pkg pip install foo' runs 'pip install foo' under coily's
audit + scope rules.`,
		Commands: r.passthroughCommands(ptPkg),
	}
}

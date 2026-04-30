package main

import (
	"github.com/coilysiren/coily/pkg/ops/passthrough"
	"github.com/urfave/cli/v3"
)

// pkgmgrBinaries is the set of package managers coily wraps as thin
// pass-throughs. Order is the priority order from issue #22: how often
// the binary shows up in coilysiren/* repos, plus how dangerous a
// missed-audit invocation would be.
//
// Skipped intentionally:
//   - deno, go install / go run: already denied at the lockdown layer
//     and not used as package-installation paths in the workspace.
var pkgmgrBinaries = []string{
	"pnpm",
	"npm",
	"yarn",
	"bun",
	"uv",
	"pip",
	"pipx",
	"poetry",
	"cargo",
	"gem",
	"bundle",
	"brew",
}

// pkgCommand groups every package-manager pass-through under a single
// `coily pkg <tool>` namespace, e.g. `coily pkg pip install foo`.
func (r *Runner) pkgCommand() *cli.Command {
	subs := make([]*cli.Command, 0, len(pkgmgrBinaries))
	for _, bin := range pkgmgrBinaries {
		subs = append(subs, passthrough.Command(bin, r.Runner, r.Audit))
	}
	return &cli.Command{
		Name:  "pkg",
		Usage: "Audited pass-throughs for language package managers.",
		Description: `pkg groups the thin pass-through wrappers around language package
managers (pip, npm, cargo, brew, etc.). Each subcommand forwards its
arguments to the underlying binary while emitting an audit record, so
'coily pkg pip install foo' runs 'pip install foo' under coily's
audit + scope rules.`,
		Commands: subs,
	}
}

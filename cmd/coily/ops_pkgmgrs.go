package main

import (
	"github.com/coilysiren/coily/pkg/ops/pkgmgr"
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

func (r *Runner) pkgmgrCommands() []*cli.Command {
	out := make([]*cli.Command, 0, len(pkgmgrBinaries))
	for _, bin := range pkgmgrBinaries {
		out = append(out, pkgmgr.Command(bin, r.Runner, r.Audit))
	}
	return out
}

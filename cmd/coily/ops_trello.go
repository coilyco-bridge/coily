package main

import (
	"github.com/coilysiren/coily/pkg/ops/trello"
	"github.com/urfave/cli/v3"
)

// trelloCommand uses RepoRunner because npm is unpinned in coily's tools
// manifest and the underlying scripts live in a sibling repo (message-ops),
// not in the privileged-passthrough surface.
func (r *Runner) trelloCommand() *cli.Command {
	runner := r.RepoRunner
	if runner == nil {
		runner = r.Runner
	}
	return trello.Command(runner, r.Audit)
}

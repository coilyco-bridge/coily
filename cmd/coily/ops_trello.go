package main

import (
	"github.com/coilysiren/coily/cmd/coily/trello"
	"github.com/urfave/cli/v3"
)

func (r *Runner) trelloCommand() *cli.Command {
	return trello.Command(r.Runner, r.Audit)
}

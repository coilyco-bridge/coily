package main

import (
	"github.com/coilysiren/coily/cmd/coily/glama"
	"github.com/urfave/cli/v3"
)

func (r *Runner) glamaCommand() *cli.Command {
	return glama.Command(r.Runner, r.Audit)
}

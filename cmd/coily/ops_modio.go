package main

import (
	"github.com/coilysiren/coily/cmd/coily/modio"
	"github.com/urfave/cli/v3"
)

func (r *Runner) modioCommand() *cli.Command {
	return modio.Command(r.Runner, r.Audit)
}

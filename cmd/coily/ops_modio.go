package main

import (
	"forgejo.coilysiren.me/coilyco-bridge/coily/cmd/coily/modio"
	"github.com/urfave/cli/v3"
)

func (r *Runner) modioCommand() *cli.Command {
	return modio.Command(r.Runner, r.Audit)
}

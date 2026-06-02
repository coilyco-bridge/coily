package main

import (
	"forgejo.coilysiren.me/coilyco-bridge/coily/cmd/coily/skillsmp"
	"github.com/urfave/cli/v3"
)

func (r *Runner) skillsmpCommand() *cli.Command {
	return skillsmp.Command(r.Runner, r.Audit)
}

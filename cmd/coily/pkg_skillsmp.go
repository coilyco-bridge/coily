package main

import (
	"github.com/coilysiren/coily/cmd/coily/skillsmp"
	"github.com/urfave/cli/v3"
)

func (r *Runner) skillsmpCommand() *cli.Command {
	return skillsmp.Command(r.Runner, r.Audit)
}

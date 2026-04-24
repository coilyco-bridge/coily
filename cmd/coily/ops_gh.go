package main

import (
	"github.com/coilysiren/coily/pkg/ops/gh"
	"github.com/urfave/cli/v3"
)

func (r *Runner) ghCommand() *cli.Command {
	return gh.Command(r.Runner, r.Audit)
}

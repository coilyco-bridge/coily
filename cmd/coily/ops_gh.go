package main

import (
	"github.com/coilysiren/coily/pkg/ops/passthrough"
	"github.com/urfave/cli/v3"
)

func (r *Runner) ghCommand() *cli.Command {
	return passthrough.Command("gh", r.Runner, r.Audit)
}

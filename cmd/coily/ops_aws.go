package main

import (
	"github.com/coilysiren/coily/pkg/ops/passthrough"
	"github.com/urfave/cli/v3"
)

func (r *Runner) awsCommand() *cli.Command {
	return passthrough.Command("aws", r.Runner, r.Audit,
		passthrough.WithSkipPolicy(),
		passthrough.WithVerbName("ops.aws"),
	)
}

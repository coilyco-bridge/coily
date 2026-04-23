package main

import (
	"github.com/coilysiren/coily/pkg/ops/aws"
	"github.com/urfave/cli/v3"
)

func (r *Runner) awsCommand() *cli.Command {
	return aws.Command(r.Runner, r.Verifier, r.Audit)
}

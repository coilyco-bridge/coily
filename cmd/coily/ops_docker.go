package main

import (
	"github.com/coilysiren/coily/pkg/ops/passthrough"
	"github.com/urfave/cli/v3"
)

func (r *Runner) dockerCommand() *cli.Command {
	return passthrough.Command("docker", r.Runner, r.Audit)
}

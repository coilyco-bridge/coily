package main

import (
	"github.com/coilysiren/coily/pkg/ops/docker"
	"github.com/urfave/cli/v3"
)

func (r *Runner) dockerCommand() *cli.Command {
	return docker.Command(r.Runner, r.Audit)
}

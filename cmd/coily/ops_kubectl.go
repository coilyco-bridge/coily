package main

import (
	"github.com/coilysiren/coily/pkg/ops/passthrough"
	"github.com/urfave/cli/v3"
)

func (r *Runner) kubectlCommand() *cli.Command {
	return passthrough.Command("kubectl", r.Runner, r.Audit)
}

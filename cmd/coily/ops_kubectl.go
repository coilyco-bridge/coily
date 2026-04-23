package main

import (
	"github.com/coilysiren/coily/pkg/ops/kubectl"
	"github.com/urfave/cli/v3"
)

func (r *Runner) kubectlCommand() *cli.Command {
	return kubectl.Command(r.Runner, r.Verifier, r.Audit)
}

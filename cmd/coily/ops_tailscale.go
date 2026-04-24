package main

import (
	"github.com/coilysiren/coily/pkg/ops/tailscale"
	"github.com/urfave/cli/v3"
)

func (r *Runner) tailscaleCommand() *cli.Command {
	return tailscale.Command(r.Runner, r.Audit)
}

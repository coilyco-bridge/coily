package main

import (
	"github.com/coilysiren/coily/pkg/ops/passthrough"
	"github.com/urfave/cli/v3"
)

func (r *Runner) tailscaleCommand() *cli.Command {
	return passthrough.Command("tailscale", r.Runner, r.Audit)
}

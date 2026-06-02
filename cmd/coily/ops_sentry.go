package main

import (
	"forgejo.coilysiren.me/coilyco-bridge/coily/cmd/coily/sentry"
	"github.com/urfave/cli/v3"
)

func (r *Runner) sentryCommand() *cli.Command {
	return sentry.Command(r.Runner, r.Audit)
}

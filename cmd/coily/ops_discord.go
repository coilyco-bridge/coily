package main

import (
	"forgejo.coilysiren.me/coilyco-bridge/coily/cmd/coily/discord"
	"github.com/urfave/cli/v3"
)

func (r *Runner) discordCommand() *cli.Command {
	return discord.Command(r.Runner, r.Audit)
}

package main

import (
	"github.com/coilysiren/coily/cmd/coily/trelloapi"
	"github.com/urfave/cli/v3"
)

// trelloapiCommand mounts the generated Trello REST wrapper under
// `coily ops trello-api`. Distinct from `coily trello`, which is the
// hand-written CLI wrapper around message-ops/scripts/trello/* for the
// recruiter pipeline. trello-api is for raw Trello REST coverage; the
// existing top-level command stays for the curated workflow surface.
func (r *Runner) trelloapiCommand() *cli.Command {
	c := trelloapi.Command(r.Runner, r.Audit)
	c.Name = "trello-api"
	c.Usage = "Generated Trello REST API wrapper (key+token auth)."
	return c
}

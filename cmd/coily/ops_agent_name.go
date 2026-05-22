package main

import (
	"context"
	"fmt"
	"os"

	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// agentNameCommand prints this agent's self-name. coily is the canonical
// source of the name; the agentic-os status line and SessionStart scripts
// shell out to this verb rather than recomputing the scheme themselves.
func (r *Runner) agentNameCommand() *cli.Command {
	return &cli.Command{
		Name:  "agent-name",
		Usage: "Print this agent's self-name: claude-<os>-<hostname>-<tag>.",
		Description: `agent-name prints the stable self-name of the agent running this
coily invocation, in the form claude-<os>-<hostname>-<tag>, where <tag>
is the last 4 chars of the Claude Code session id.

The session id is read from --session-id, or from $CLAUDE_CODE_SESSION_ID
when the flag is absent. Status line and SessionStart hook integrations
pass --session-id, since those contexts receive the id on stdin, not in
the environment.

Pure local lookup: no aws/kubectl/gh calls, unlike coily whoami.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "session-id",
				Usage: "session id to derive the tag from; defaults to $CLAUDE_CODE_SESSION_ID",
			},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name:      "agent-name",
				SkipScope: true,
				Action: func(_ context.Context, c *cli.Command) error {
					sid := c.String("session-id")
					if sid == "" {
						sid = os.Getenv(sessionEnvVar)
					}
					fmt.Println(resolveAgentIdentity(sid).name)
					return nil
				},
			},
			r.Audit,
		),
	}
}

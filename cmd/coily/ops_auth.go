package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/coilysiren/coily/pkg/policy"
	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

func init() { registerCommand(authCmd) }

var authCmd = &cli.Command{
	Name:  "auth",
	Usage: "Issue and verify confirmation tokens for destructive verbs.",
	Description: `Mutating verbs (coily aws route53 change-resource-record-sets, coily k8s
rollout restart, coily eco restart, etc.) require a short-lived confirmation
token. Tokens are scoped to one verb and expire after --ttl.

Flow: Kai runs 'coily auth issue --scope <verb> --ttl 5m', gets a token,
pastes it into the mutating invocation as --token or $COILY_TOKEN.

The token issuer's key is stored at the path configured in
config.tokens.issuer_key_path. Anyone with read access to that file can
forge tokens. This is intentional. The threat model is "prevent a confused
agent from calling a destructive verb by accident," not "prevent a local
attacker with shell access." Review the audit log for any unexpected Issue
entries.`,
	Commands: []*cli.Command{
		authIssueCmd,
		authVerifyCmd,
	},
}

var authIssueCmd = &cli.Command{
	Name:  "issue",
	Usage: "Issue a confirmation token for a named verb scope.",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "scope",
			Usage:    "verb scope this token authorizes, e.g. aws.route53.change-resource-record-sets",
			Required: true,
		},
		&cli.DurationFlag{
			Name:  "ttl",
			Usage: "how long the token is valid",
			Value: 5 * time.Minute,
		},
	},
	Action: verb.Wrap(
		verb.Spec{
			Name: "auth.issue",
			Kind: policy.ReadOnly,
			ArgsFunc: func(c *cli.Command) (map[string]string, []string, string) {
				return map[string]string{"--scope": c.String("scope")}, nil, ""
			},
			Action: func(_ context.Context, c *cli.Command) error {
				scope := c.String("scope")
				ttl := c.Duration("ttl")
				tok, err := getRuntime().issuer.Issue(scope, ttl)
				if err != nil {
					return err
				}
				fmt.Println(tok)
				fmt.Fprintf(os.Stderr, "issued %s-scoped token, expires in %s\n", scope, ttl)
				return nil
			},
		},
		getRuntime().issuer,
		getRuntime().audit,
	),
}

var authVerifyCmd = &cli.Command{
	Name:  "verify",
	Usage: "Verify a token against a scope. Exit 0 on valid, non-zero on any failure.",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "scope",
			Usage:    "scope the token should cover",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "token",
			Usage:    "the token to verify",
			Required: true,
		},
	},
	Action: verb.Wrap(
		verb.Spec{
			Name: "auth.verify",
			Kind: policy.ReadOnly,
			ArgsFunc: func(c *cli.Command) (map[string]string, []string, string) {
				return map[string]string{"--scope": c.String("scope")}, nil, ""
			},
			Action: func(_ context.Context, c *cli.Command) error {
				if err := getRuntime().issuer.Verify(c.String("scope"), c.String("token")); err != nil {
					return err
				}
				fmt.Println("ok")
				return nil
			},
		},
		getRuntime().issuer,
		getRuntime().audit,
	),
}

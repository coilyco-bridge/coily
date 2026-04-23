package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/coilysiren/coily/pkg/auth"
	"github.com/coilysiren/coily/pkg/policy"
	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

func (r *Runner) authCommand() *cli.Command {
	return &cli.Command{
		Name:  "auth",
		Usage: "Issue and verify confirmation tokens for destructive verbs.",
		Description: `Mutating verbs (coily aws route53 change-resource-record-sets, coily k8s
rollout restart, coily eco restart, etc.) require a short-lived confirmation
token. Tokens are scoped to one bucket per service and expire after --ttl.

Scope strings have the form <binary>.<service>:<bucket> where bucket is
read|write|delete. Examples: aws.route53:write, aws.route53:delete,
gh.pr:write, kubectl.rollout:write, coily.eco:write. write subsumes read
on the same service. delete is its own bucket and is not subsumed by
write. To grant all three, pass --scope aws.route53:write,aws.route53:delete.

Flow: Kai runs 'coily auth issue --scope <scope> --ttl 5m', gets a token,
pastes it into the mutating invocation as --token or $COILY_TOKEN.

The token issuer's key is stored at the path configured in
config.tokens.issuer_key_path. Anyone with read access to that file can
forge tokens. This is intentional. The threat model is "prevent a confused
agent from calling a destructive verb by accident," not "prevent a local
attacker with shell access." Review the audit log for any unexpected Issue
entries.`,
		Commands: []*cli.Command{
			r.authIssueCommand(),
			r.authVerifyCommand(),
		},
	}
}

func (r *Runner) authIssueCommand() *cli.Command {
	return &cli.Command{
		Name:  "issue",
		Usage: "Issue a confirmation token for a named verb scope.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "scope",
				Usage:    "scope this token authorizes, e.g. aws.route53:write. Comma-separate to bind multiple.",
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
					if err := policy.ValidateScopeList(scope); err != nil {
						return err
					}
					issuer, ok := r.Verifier.(*auth.Issuer)
					if !ok {
						return fmt.Errorf("auth issue: verifier is not an *auth.Issuer; cannot mint tokens")
					}
					tok, err := issuer.Issue(scope, ttl)
					if err != nil {
						return err
					}
					fmt.Println(tok)
					fmt.Fprintf(os.Stderr, "issued %s-scoped token, expires in %s\n", scope, ttl)
					return nil
				},
			},
			r.Verifier,
			r.Audit,
		),
	}
}

func (r *Runner) authVerifyCommand() *cli.Command {
	return &cli.Command{
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
					issuer, ok := r.Verifier.(*auth.Issuer)
					if !ok {
						return fmt.Errorf("auth verify: verifier is not an *auth.Issuer; cannot verify tokens")
					}
					if err := issuer.Verify(c.String("scope"), c.String("token")); err != nil {
						return err
					}
					fmt.Println("ok")
					return nil
				},
			},
			r.Verifier,
			r.Audit,
		),
	}
}

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

// forgejoCommand wraps the in-pod `forgejo` CLI on the kai-server k3s deploy.
// Each leaf renders a fixed-shape `k3s kubectl -n forgejo exec deploy/forgejo
// -- forgejo <verb...>` and runs it through the same audit + ssh path as
// `coily ssh kubectl`. Namespace and pod selector are pinned in code: no
// free-form passthrough, no flag overrides for the target.
//
// This is the readonly slice of #57 (#91): list and diagnostic verbs only.
// Mutating verbs (admin user create, regenerate, dump, manager flush-queues,
// etc.) land in followup issues.
//
// The harness deny-rule on raw `kubectl exec` stays. This wrapper is the
// affordance, not a bypass.
func (r *Runner) forgejoCommand() *cli.Command {
	return &cli.Command{
		Name:  "forgejo",
		Usage: "Run readonly forgejo CLI verbs against the kai-server forgejo pod.",
		Description: `forgejo wraps the in-pod forgejo CLI. Each leaf is a fixed-shape
k3s kubectl -n forgejo exec deploy/forgejo -- forgejo <verb...>, run via
ssh to kai-server. Namespace and pod selector are pinned in code.

Readonly slice (#91, parent #57):
  coily ops forgejo admin user list
  coily ops forgejo admin auth list
  coily ops forgejo doctor check --run <name>`,
		Commands: []*cli.Command{
			{
				Name:  "admin",
				Usage: "Forgejo admin verbs.",
				Commands: []*cli.Command{
					{
						Name:  "user",
						Usage: "Forgejo admin user verbs.",
						Commands: []*cli.Command{
							r.forgejoAdminUserListCommand(),
						},
					},
					{
						Name:  "auth",
						Usage: "Forgejo admin auth verbs.",
						Commands: []*cli.Command{
							r.forgejoAdminAuthListCommand(),
						},
					},
				},
			},
			{
				Name:  "doctor",
				Usage: "Forgejo doctor verbs.",
				Commands: []*cli.Command{
					r.forgejoDoctorCheckCommand(),
				},
			},
		},
	}
}

func (r *Runner) forgejoAdminUserListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List forgejo users.",
		Action: verb.Wrap(
			verb.Spec{
				Name: "ops.forgejo.admin.user.list",
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo admin user list: takes no args, got %d", c.Args().Len())
					}
					return r.runForgejo(ctx, []string{"admin", "user", "list"})
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) forgejoAdminAuthListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List forgejo auth sources.",
		Action: verb.Wrap(
			verb.Spec{
				Name: "ops.forgejo.admin.auth.list",
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo admin auth list: takes no args, got %d", c.Args().Len())
					}
					return r.runForgejo(ctx, []string{"admin", "auth", "list"})
				},
			},
			r.Audit,
		),
	}
}

// forgejoDoctorCheckCommand pins the readonly form: --run <name> only. The
// underlying `forgejo doctor check` accepts --fix, which would mutate state.
// That flag is intentionally not exposed here; --fix-shaped verbs land in a
// followup issue.
func (r *Runner) forgejoDoctorCheckCommand() *cli.Command {
	return &cli.Command{
		Name:  "check",
		Usage: "Run a forgejo doctor check (readonly; --fix not exposed).",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "run",
				Usage:    "doctor check name (e.g. db-version, paths)",
				Required: true,
			},
		},
		Action: verb.Wrap(
			verb.Spec{
				Name: "ops.forgejo.doctor.check",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{"--run": c.String("run")}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo doctor check: takes no positional args, got %d", c.Args().Len())
					}
					name := c.String("run")
					if err := validateForgejoDoctorCheckName(name); err != nil {
						return err
					}
					return r.runForgejo(ctx, []string{"doctor", "check", "--run", name})
				},
			},
			r.Audit,
		),
	}
}

// runForgejo is the single ssh-stream call shared by every forgejo leaf.
// Pinned to namespace=forgejo, pod-selector=deploy/forgejo. The verb argv is
// the forgejo subcommand (e.g. ["admin", "user", "list"]).
func (r *Runner) runForgejo(ctx context.Context, forgejoArgs []string) error {
	host := r.Cfg.KaiServer.TailscaleHost
	user := r.Cfg.KaiServer.SSHUser
	if host == "" || user == "" {
		return fmt.Errorf("ops forgejo: kai_server.tailscale_host / ssh_user must be set in config")
	}
	return r.SSH.Stream(ctx, host, user, renderForgejoCmd(forgejoArgs), os.Stdout, os.Stderr)
}

// renderForgejoCmd assembles the remote command string sent to kai-server.
// Pulled out as a helper so the pinned-target + quoted-argv property is
// unit-testable without spinning up an ssh fake. The shape is:
//
//	k3s kubectl -n forgejo exec deploy/forgejo -- forgejo <quoted args...>
//
// Each forgejo arg is POSIX single-quoted before joining, same rationale as
// renderSSHKubectlCmd: the remote login shell would otherwise re-interpret
// shell metacharacters.
func renderForgejoCmd(forgejoArgs []string) string {
	parts := []string{"k3s", "kubectl", "-n", "forgejo", "exec", "deploy/forgejo", "--", "forgejo"}
	for _, a := range forgejoArgs {
		parts = append(parts, posixShellQuote(a))
	}
	return strings.Join(parts, " ")
}

// validateForgejoDoctorCheckName rejects anything that isn't a sane forgejo
// doctor check name. Forgejo's check names are kebab-case identifiers
// (db-version, paths, storages, etc.); we add a length cap and disallow
// leading "-" to avoid argv-as-flag confusion.
func validateForgejoDoctorCheckName(name string) error {
	if name == "" {
		return fmt.Errorf("ops forgejo doctor check: --run is empty")
	}
	if len(name) > 64 {
		return fmt.Errorf("ops forgejo doctor check: --run name too long")
	}
	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("ops forgejo doctor check: --run name must not start with '-'")
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-', r == '_':
			// ok
		default:
			return fmt.Errorf("ops forgejo doctor check: --run name contains invalid character %q", r)
		}
	}
	return nil
}

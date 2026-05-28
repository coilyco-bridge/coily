package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/coilysiren/cli-guard/audit"
	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// stampPolicySkipped marks the audit row to show the shell-metacharacter
// gate was deliberately skipped for this verb. Used as the OnComplete for
// the forgejo create/edit verbs: their free-text args (issue/PR/milestone
// titles, label/release names) are marshaled into an HTTP JSON body via
// forgejoAPIDo and never reach a shell or even execve, so the gate's threat
// model (argv re-evaluated by a downstream shell) does not apply. Skipping
// it lets a title like "feat(dispatch): x" through verbatim instead of
// bouncing or being mangled, and the stamp keeps the audit trail honest
// about the relaxed policy. Mirrors the allow_metacharacters opt-out for
// user exec verbs (SECURITY.md).
func stampPolicySkipped(rec *audit.Record) { rec.PolicySkipped = true }

// forgejoCommand wraps the in-pod `forgejo` CLI on the kai-server k3s deploy.
// Each leaf renders a fixed-shape `k3s kubectl -n forgejo exec deploy/forgejo
// -- forgejo <verb...>` and runs it through the same audit + ssh path as
// `coily ssh kubectl`. Namespace and pod selector are pinned in code: no
// free-form passthrough, no flag overrides for the target.
//
// Slices landed so far (parent #57):
//   - #91 readonly: admin user list, admin auth list, doctor check
//   - #241 onboard a peer admin: admin user create (random-password +
//     must-change-password pinned on, no --password flag exposed)
//
// The harness deny-rule on raw `kubectl exec` stays. This wrapper is the
// affordance, not a bypass.
func (r *Runner) forgejoCommand() *cli.Command {
	return &cli.Command{
		Name:    "forgejo",
		Aliases: []string{"fj"},
		Usage:   "Run forgejo CLI verbs against the kai-server forgejo pod.",
		Description: `forgejo wraps the in-pod forgejo CLI. Each leaf is a fixed-shape
k3s kubectl -n forgejo exec deploy/forgejo -- forgejo <verb...>, run via
ssh to kai-server. Namespace and pod selector are pinned in code.

Verbs:
  coily ops forgejo admin user list
  coily ops forgejo admin user create --username <name> --email <addr> [--admin]
  coily ops forgejo admin auth list
  coily ops forgejo doctor check --run <name>
  coily ops forgejo issue create --repo <owner/name> --title <t> --body-file <path>
  coily ops forgejo label create --repo <owner/name> --name <n> --color <hex>
  coily ops forgejo milestone create --repo <owner/name> --title <t> [--due-on YYYY-MM-DD]
  coily ops forgejo release create --repo <owner/name> --tag <tag> [--body-file <path>]
  coily ops forgejo repo view --repo <owner/name>
  coily ops forgejo pr list --repo <owner/name>`,
		Commands: []*cli.Command{
			r.forgejoIssueCommand(),
			r.forgejoLabelCommand(),
			r.forgejoMilestoneCommand(),
			r.forgejoReleaseCommand(),
			r.forgejoRepoCommand(),
			r.forgejoPRCommand(),
			r.forgejoActionsCommand(),
			{
				Name:  "admin",
				Usage: "Forgejo admin verbs.",
				Commands: []*cli.Command{
					{
						Name:  "user",
						Usage: "Forgejo admin user verbs.",
						Commands: []*cli.Command{
							r.forgejoAdminUserListCommand(),
							r.forgejoAdminUserCreateCommand(),
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
		Action: r.WrapVerb(
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

// forgejoAdminUserCreateCommand onboards a new forgejo user. --random-password
// and --must-change-password are pinned on in code, not exposed as flags: the
// generated password streams out of the pod on stdout, the operator hands it
// to the new user out-of-band, and the user is forced to rotate on first
// login. No --password flag is exposed, so the secret never reaches coily's
// argv or audit log.
func (r *Runner) forgejoAdminUserCreateCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a forgejo user with a random password and forced first-login rotation.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username", Usage: "forgejo username", Required: true},
			&cli.StringFlag{Name: "email", Usage: "user email address", Required: true},
			&cli.BoolFlag{Name: "admin", Usage: "grant site-admin privileges"},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.admin.user.create",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					args := map[string]string{
						"--username": c.String("username"),
						"--email":    c.String("email"),
					}
					if c.Bool("admin") {
						args["--admin"] = "true"
					}
					return args, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo admin user create: takes no positional args, got %d", c.Args().Len())
					}
					username := c.String("username")
					email := c.String("email")
					if err := validateForgejoUsername(username); err != nil {
						return err
					}
					if err := validateForgejoEmail(email); err != nil {
						return err
					}
					argv := []string{
						"admin", "user", "create",
						"--username", username,
						"--email", email,
						"--random-password",
						"--must-change-password",
					}
					if c.Bool("admin") {
						argv = append(argv, "--admin")
					}
					return r.runForgejo(ctx, argv)
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
		Action: r.WrapVerb(
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
		Action: r.WrapVerb(
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

// runForgejo dispatches every forgejo leaf to either a local exec or an
// ssh stream to kai-server. Pinned to namespace=forgejo,
// pod-selector=deploy/forgejo. The verb argv is the forgejo subcommand
// (e.g. ["admin", "user", "list"]).
//
// When invoked on kai-server itself (detected via hostIsLocal), the
// kubectl command runs directly through r.Runner.Exec to skip an
// ssh-to-self that doesn't carry authentication in non-interactive
// environments. Same pattern as systemdRemote (coilysiren/coily#135),
// closing coilysiren/coily#260.
func (r *Runner) runForgejo(ctx context.Context, forgejoArgs []string) error {
	host := r.Cfg.KaiServer.TailscaleHost
	if host == "" {
		return fmt.Errorf("ops forgejo: kai_server.tailscale_host must be set in config")
	}
	if hostIsLocal(host) {
		argv := forgejoLocalArgv(forgejoArgs)
		return r.Runner.Exec(ctx, argv[0], argv[1:]...)
	}
	user := r.Cfg.KaiServer.SSHUser
	if user == "" {
		return fmt.Errorf("ops forgejo: kai_server.ssh_user must be set in config")
	}
	return r.SSH.Stream(ctx, host, user, renderForgejoCmd(forgejoArgs), os.Stdout, os.Stderr)
}

// forgejoLocalArgv returns the argv used when forgejo runs on kai-server
// itself. No shell layer involved, so no POSIX quoting; the args go
// straight into execve. Shape mirrors renderForgejoCmd's pinned target.
func forgejoLocalArgv(forgejoArgs []string) []string {
	argv := []string{"k3s", "kubectl", "-n", "forgejo", "exec", "deploy/forgejo", "--", "forgejo"}
	return append(argv, forgejoArgs...)
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

// posixShellQuote wraps s in POSIX single quotes. Embedded single quotes
// are encoded as '\” (close-quote, escaped quote, reopen-quote). The
// result is safe to splice into a shell command line. Distinct from
// ssh_passthrough.go's posixQuote, which fast-paths safe strings and is
// tested against that behavior; this one always quotes, matching the
// kubectl-exec-shell discipline that the forgejo verb inherited from
// the old `coily ssh kubectl` path.
func posixShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// validateForgejoUsername enforces forgejo's username character set
// (alnum plus `-` `_` `.`) with a length cap and a leading-dash reject
// to avoid argv-as-flag confusion. Real validation (uniqueness, reserved
// names) is forgejo's job; this is the shell-safety + sanity layer.
func validateForgejoUsername(name string) error {
	if name == "" {
		return fmt.Errorf("ops forgejo admin user create: --username is empty")
	}
	if len(name) > 40 {
		return fmt.Errorf("ops forgejo admin user create: --username too long")
	}
	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("ops forgejo admin user create: --username must not start with '-'")
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-', r == '_', r == '.':
		default:
			return fmt.Errorf("ops forgejo admin user create: --username contains invalid character %q", r)
		}
	}
	return nil
}

// validateForgejoEmail is a loose shape check: non-empty, no shell-meta or
// whitespace, exactly one `@` with non-empty local and domain parts, a `.`
// in the domain. Forgejo does the real validation; this rejects the
// argv-as-flag and shell-injection shapes before the remote shell sees it.
func validateForgejoEmail(email string) error {
	switch {
	case email == "":
		return fmt.Errorf("ops forgejo admin user create: --email is empty")
	case len(email) > 254:
		return fmt.Errorf("ops forgejo admin user create: --email too long")
	case strings.HasPrefix(email, "-"):
		return fmt.Errorf("ops forgejo admin user create: --email must not start with '-'")
	}
	if bad := firstInvalidEmailChar(email); bad != 0 {
		return fmt.Errorf("ops forgejo admin user create: --email contains invalid character %q", bad)
	}
	at := strings.IndexByte(email, '@')
	if at <= 0 || at != strings.LastIndexByte(email, '@') || at == len(email)-1 {
		return fmt.Errorf("ops forgejo admin user create: --email must have exactly one '@' with non-empty local and domain parts")
	}
	if !strings.Contains(email[at+1:], ".") {
		return fmt.Errorf("ops forgejo admin user create: --email domain must contain '.'")
	}
	return nil
}

func firstInvalidEmailChar(email string) rune {
	for _, r := range email {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-', r == '_', r == '.', r == '+', r == '@':
		default:
			return r
		}
	}
	return 0
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

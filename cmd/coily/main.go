package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/exitcode"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

// Version is injected at build time via -ldflags "-X main.Version=<sha>".
var Version = "dev"

func main() {
	r := NewRunner()
	err := run(r, os.Args)
	if err != nil {
		rc := classifyExit(err)
		emitErrorEnvelope(os.Stderr, err, rc, r.Cfg.Audit.LogPath)
		os.Exit(rc)
	}
}

// classifyExit walks the error chain looking for a coded error. Falls
// back to UpstreamFailed for *exec.ExitError (the underlying tool ran
// and returned non-zero) and Generic for anything else. See
// cli-guard/exitcode for the public contract.
func classifyExit(err error) int {
	if c := exitcode.From(err); c != nil {
		return c.Code()
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return exitcode.UpstreamFailed
	}
	return exitcode.Generic
}

// errorEnvelope is the structured failure shape coily writes to stderr
// alongside the human-readable error line. Stable across every refusal /
// failure path so an external consumer can pattern-match on `kind`
// instead of stderr regex. See SECURITY.md / docs for the contract.
type errorEnvelope struct {
	Kind         string `yaml:"kind"`
	Message      string `yaml:"message"`
	Hint         string `yaml:"hint,omitempty"`
	Reason       string `yaml:"reason,omitempty"`
	ExitCode     int    `yaml:"exit_code"`
	AuditLogPath string `yaml:"audit_log_path,omitempty"`
	Timestamp    int64  `yaml:"timestamp"`
}

func emitErrorEnvelope(w *os.File, err error, rc int, auditPath string) {
	env := errorEnvelope{
		Kind:         kindFor(err, rc),
		Message:      err.Error(),
		ExitCode:     rc,
		AuditLogPath: auditPath,
		Timestamp:    time.Now().Unix(),
	}
	if c := exitcode.From(err); c != nil {
		if ce, ok := c.(interface{ HintText() string }); ok {
			env.Hint = ce.HintText()
		}
	}
	var rsn exitcode.Reasoner
	if errors.As(err, &rsn) {
		env.Reason = rsn.Reason()
	}
	if env.Reason == "" {
		env.Reason = reasonFor(env.Kind)
	}
	// Write the human line first so it stays visible even if a downstream
	// pipe truncates; envelope follows for programmatic consumers.
	_, _ = fmt.Fprintln(w, "coily:", err)
	out, mErr := yaml.Marshal(map[string]any{"error": env})
	if mErr != nil {
		return
	}
	_, _ = fmt.Fprint(w, string(out))
}

func kindFor(err error, rc int) string {
	if c := exitcode.From(err); c != nil {
		return c.Kind()
	}
	switch rc {
	case exitcode.UpstreamFailed:
		return "upstream_failed"
	default:
		return "generic"
	}
}

// run wires a Runner into the urfave/cli v3 root command and executes it
// against argv. Split out from main() so tests can drive a Runner with fake
// dependencies through a real cli.Command tree.
func run(r *Runner, argv []string) error {
	// Spill an inline `ops gh ... --jq <expr>` onto the gate-safe --jq-file
	// rail before the metachar gate sees it (coilyco-bridge/coily#30). No-op
	// for every other invocation.
	argv, cleanupJQ := normalizeGHJQInline(argv)
	defer cleanupJQ()

	builtIns := r.builtInCommands()
	repoResult, execCmd := r.loadRepoExecCommand()

	// execCmd is always non-nil: loadRepoExecCommand returns a stub `exec`
	// command with a UserError Action when no .coily/coily.yaml is in scope,
	// so the verb is unconditionally visible in --help and --tree.
	all := append([]*cli.Command{}, builtIns...)
	all = append(all, execCmd)

	cmd := &cli.Command{
		Name:                  "coily",
		Usage:                 "Operator CLI for Kai's homelab.",
		Version:               Version,
		Commands:              all,
		EnableShellCompletion: true,
		// Default urfave/cli ExitErrHandler calls os.Exit(1) directly,
		// short-circuiting our coded-exit + yaml-envelope handling in
		// main(). Replace with a no-op so the error bubbles up.
		ExitErrHandler: func(_ context.Context, _ *cli.Command, _ error) {},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "list",
				Usage: "print every top-level command coily can run (built-in + repo) and exit",
			},
			&cli.BoolFlag{
				Name:  "tree",
				Usage: "print the full command tree (every subcommand, recursively) and exit",
			},
			&cli.BoolFlag{
				Name: "audit-override-dirty",
				Usage: "bypass the clean+synced gate on repo verbs declared in " +
					".coily/coily.yaml. Tags the audit row with audit_override=true " +
					"and captures the working tree status. For genuine emergencies " +
					"only: the gate exists so audit rows can be reconstructed from " +
					"git history.",
			},
			&cli.StringFlag{
				Name: "cwd",
				Usage: "chdir to this path before any other processing. " +
					"Internal flag used by `coily systemctl` self-elevation " +
					"(coily#245) so the sudo'd child lands in the same git " +
					"toplevel the outer was running in, so its audit row's " +
					"RepoRoot matches the outer's. " +
					"Operators rarely need this directly.",
			},
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			// --cwd chdirs before any subcommand action so scope.RepoRoot
			// (which reads os.Getwd) sees the post-chdir path. Powers
			// the systemctl self-elevation invariant in coily#245: the
			// sudo'd child lands in the outer's git toplevel before its
			// audit row's RepoRoot is stamped.
			if dir := c.String("cwd"); dir != "" {
				if err := os.Chdir(dir); err != nil {
					return ctx, fmt.Errorf("coily: --cwd=%q: %w", dir, err)
				}
			}
			return ctx, nil
		},
		Action: func(_ context.Context, c *cli.Command) error {
			if c.Bool("list") {
				listCommand(builtIns, execCmd, repoResult)
				return nil
			}
			if c.Bool("tree") {
				treeCommand(builtIns, execCmd, repoResult)
				return nil
			}
			return cli.ShowAppHelp(c)
		},
	}

	return cmd.Run(context.Background(), argv)
}

// builtInCommands returns the prod-build verbs in registration order. Each
// verb file contributes one builder method; this list is the single place
// they are wired in. Adding a verb means writing the file and appending its
// builder here. Top-level pass-throughs come from ptTopLevel in
// passthroughs.go - those don't get their own per-binary file.
func (r *Runner) builtInCommands() []*cli.Command {
	cmds := []*cli.Command{
		r.versionCommand(),
		r.upgradeCommand(),
		r.whoamiCommand(),
		r.agentNameCommand(),
		r.lockdownCommand(),
		r.installCompletionCommand(),
		r.setupCommand(),
		r.gamingCommand(),
		r.opsCommand(),
		r.systemctlCommand(),
		r.auditCommand(),
		r.gitCommand(),
		r.dispatchCommand(),
		r.pkgCommand(),
		r.hookCommand(),
		r.lintCommand(),
		r.sessionCommand(),
	}
	cmds = append(cmds, r.passthroughCommands(ptTopLevel)...)
	return cmds
}

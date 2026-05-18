package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/coilysiren/cli-guard/exitcode"
	"github.com/coilysiren/cli-guard/verb"
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

// liftCommitScope pulls --commit-scope out of any position in argv and
// reinserts it as a global flag (right after argv[0]). urfave/cli requires
// global flags to precede the verb chain, but passthrough verbs use
// SkipFlagParsing, so a flag placed after the verb (the natural spot) gets
// consumed as part of the wrapped binary's argv. Pre-scanning here makes
// --commit-scope positionally flexible (closes #101).
//
// Recognized forms:
//
//	--commit-scope=<value>   single token
//	--commit-scope <value>   two tokens; the next token is treated as the
//	                         value only when it does not itself look like a
//	                         flag (a bare --commit-scope is dropped).
//
// Last occurrence wins, matching urfave/cli's default for repeated flags.
// All instances are removed from their original positions.
//
// No coily-fronted binary (gh, aws, kubectl, docker, tailscale, the package
// managers) defines a --commit-scope flag, so lifting cannot collide with a
// legitimate wrapped-tool flag. A user-supplied argv value that happens to
// contain the literal string "--commit-scope" inside a quoted body remains a
// single argv element and does not match the token check.
func liftCommitScope(argv []string) []string {
	if len(argv) < 2 {
		return argv
	}
	const flag = "--commit-scope"
	var liftedValue string
	var liftedFound bool
	var anyConsumed bool
	out := make([]string, 0, len(argv))
	out = append(out, argv[0])
	i := 1
	for i < len(argv) {
		tok := argv[i]
		switch {
		case tok == flag:
			anyConsumed = true
			if i+1 < len(argv) && !strings.HasPrefix(argv[i+1], "-") {
				liftedValue = argv[i+1]
				liftedFound = true
				i += 2
				continue
			}
			i++
			continue
		case strings.HasPrefix(tok, flag+"="):
			anyConsumed = true
			liftedValue = strings.TrimPrefix(tok, flag+"=")
			liftedFound = true
			i++
			continue
		}
		out = append(out, tok)
		i++
	}
	if !anyConsumed {
		return argv
	}
	if !liftedFound {
		return out
	}
	result := make([]string, 0, len(out)+1)
	result = append(result, out[0], flag+"="+liftedValue)
	result = append(result, out[1:]...)
	return result
}

// run wires a Runner into the urfave/cli v3 root command and executes it
// against argv. Split out from main() so tests can drive a Runner with fake
// dependencies through a real cli.Command tree.
func run(r *Runner, argv []string) error {
	argv = liftCommitScope(argv)
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
				Name: verb.CommitScopeFlag,
				Usage: "bind audit rows to a commit scope. " +
					"`auto` (default) = git toplevel of cwd; " +
					"`<path>` = explicit repo path. " +
					"There is no opt-out: every invocation must bind to a real repo. " +
					"Read from $COILY_COMMIT_SCOPE if unset.",
				Value:   "auto",
				Sources: cli.EnvVars("COILY_COMMIT_SCOPE"),
			},
			&cli.StringFlag{
				Name: verb.AuditParentFlag,
				Usage: "record this invocation's audit row as a child of <id>. " +
					"Set by `coily ssh <alias> -- <args>` on the remote invocation so " +
					"the remote row links back to the local row (coilysiren/coily#187). " +
					"Read from $COILY_AUDIT_PARENT if unset. " +
					"Empty in the common single-host case.",
				Sources: cli.EnvVars(verb.AuditParentEnvVar),
			},
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
		r.lockdownCommand(),
		r.installCompletionCommand(),
		r.setupCommand(),
		r.gamingCommand(),
		r.opsCommand(),
		r.sshCommand(),
		r.systemctlCommand(),
		r.auditCommand(),
		r.gitCommand(),
		r.dispatchCommand(),
		r.pkgCommand(),
		r.brewCommand(),
		r.lintCommand(),
		r.sessionCommand(),
	}
	cmds = append(cmds, r.passthroughCommands(ptTopLevel)...)
	return cmds
}

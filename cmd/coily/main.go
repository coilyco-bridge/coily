package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/coilysiren/coily/pkg/exitcode"
	"github.com/coilysiren/coily/pkg/verb"
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
// pkg/exitcode for the public contract.
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
	builtIns := r.builtInCommands()

	reserved := map[string]bool{}
	for _, c := range builtIns {
		reserved[c.Name] = true
	}
	repoCfg, repoCmds := r.loadRepoCommands(reserved)

	cmd := &cli.Command{
		Name:                  "coily",
		Usage:                 "Operator CLI for Kai's homelab.",
		Version:               Version,
		Commands:              append(append([]*cli.Command{}, builtIns...), repoCmds...),
		EnableShellCompletion: true,
		// Default urfave/cli ExitErrHandler calls os.Exit(1) directly,
		// short-circuiting our coded-exit + yaml-envelope handling in
		// main(). Replace with a no-op so the error bubbles up.
		ExitErrHandler: func(_ context.Context, _ *cli.Command, _ error) {},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "list",
				Usage: "print every command coily can run (built-in + repo) and exit",
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
		},
		Action: func(_ context.Context, c *cli.Command) error {
			if c.Bool("list") {
				listCommand(builtIns, repoCmds, repoCfg)
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
		r.whoamiCommand(),
		r.lockdownCommand(),
		r.installCompletionCommand(),
		r.setupCommand(),
		r.gamingCommand(),
		r.opsCommand(),
		r.sshCommand(),
		r.auditCommand(),
		r.gitCommand(),
		r.pkgCommand(),
	}
	cmds = append(cmds, r.passthroughCommands(ptTopLevel)...)
	return cmds
}

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/config"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/exitcode"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/profiles"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// sessionEnvVar is the Claude Code-injected env var coily reads to bind
// session state. coily attaches; Claude Code owns the lifecycle.
const sessionEnvVar = "CLAUDE_CODE_SESSION_ID"

// sessionCommand groups the per-session lockdown-profile verbs. Phase 2
// of coilysiren/coily#150: observable state only, no gate behavior.
// The sentinel at ~/.coily/audit/sessions/<id>/profile records which
// named profile this Claude Code session is operating under; phase 4
// wires that name through to the lockdown evaluator.
func (r *Runner) sessionCommand() *cli.Command {
	return &cli.Command{
		Name:  "session",
		Usage: "Per-session lockdown-profile state for Claude Code sessions.",
		Description: `session manages the per-Claude-Code-session sentinel that records
which named lockdown profile the current session is operating under.
coily attaches to the lifecycle via $CLAUDE_CODE_SESSION_ID; Claude
Code owns start and end.

The sentinel lives at:

    ~/.coily/audit/sessions/<CLAUDE_CODE_SESSION_ID>/profile

containing the profile name as a single line of plain text.

In phase 2 the sentinel is observable state only. No gate behavior
changes until phase 4 (coilysiren/coily#150).`,
		Commands: []*cli.Command{
			r.sessionUseCommand(),
			r.sessionShowCommand(),
			r.sessionClearCommand(),
			r.sessionEndCommand(),
		},
	}
}

func (r *Runner) sessionUseCommand() *cli.Command {
	return &cli.Command{
		Name:      "use",
		Usage:     "Record the active lockdown profile for this Claude Code session.",
		ArgsUsage: "<profile-name>",
		Action: r.WrapVerb(
			verb.Spec{
				Name: "session.use",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return nil, c.Args().Slice()
				},
				Action: func(_ context.Context, c *cli.Command) error {
					return sessionUseAction(c)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) sessionShowCommand() *cli.Command {
	return &cli.Command{
		Name:  "show",
		Usage: "Print the active profile and (phase-2) the would-be strictest axis tiers.",
		Action: r.WrapVerb(
			verb.Spec{
				Name: "session.show",
				Action: func(_ context.Context, _ *cli.Command) error {
					return sessionShowAction(os.Stdout)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) sessionClearCommand() *cli.Command {
	return &cli.Command{
		Name:  "clear",
		Usage: "Remove the per-session sentinel. No-op if absent.",
		Action: r.WrapVerb(
			verb.Spec{
				Name: "session.clear",
				Action: func(_ context.Context, _ *cli.Command) error {
					return sessionClearAction()
				},
			},
			r.Audit,
		),
	}
}

// sessionEndCommand terminates the Claude Code session this invocation
// runs inside. It is the honest, operator-authorized self-termination
// path for a dispatched sidequest session that has finished its work
// (coilysiren/coily#309).
//
// Nothing here is disguised. The verb name, this help text, and the
// `session.end` audit row all state plainly that it ends the session by
// signalling the claude process. It is allowlisted because Kai authorized
// this exact, accurately-described capability - not because a classifier
// misreads it. The bare `kill $PPID` it replaces reads to the harness
// auto-mode classifier as unrequested and disruptive, with no signal that
// it is the sanctioned end-of-sidequest move.
func (r *Runner) sessionEndCommand() *cli.Command {
	return &cli.Command{
		Name:  "end",
		Usage: "Terminate the current Claude Code session (SIGTERM to the claude process).",
		Description: `end terminates the Claude Code session this invocation is running
inside, by sending SIGTERM to the claude CLI process. The terminal then
shows the "process exited" banner.

This is the honest self-termination path for a dispatched sidequest
session that has genuinely finished its work - committed, pushed,
merged to main, checks green, no human decision left to make. Instead
of a bare 'kill $PPID', the finished session runs 'coily session end'.

What it does, step by step:

  1. Confirms it is running inside a Claude Code session
     ($CLAUDE_CODE_SESSION_ID must be set).
  2. Walks the process ancestry of this invocation to find the
     claude CLI process.
  3. Sends that process SIGTERM so it shuts down cleanly.

The audit row records verb 'session.end': a session was deliberately
ended. Name, help text, and audit row all describe terminating the
session - the value of this verb does not depend on any classifier
misreading what it does.

Pass --dry-run to print the claude PID that would be signalled without
actually sending SIGTERM.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "print the claude PID that would be signalled, do not send SIGTERM",
			},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "session.end",
				Action: func(_ context.Context, c *cli.Command) error {
					return sessionEndAction(c, os.Stdout)
				},
			},
			r.Audit,
		),
	}
}

// sessionEndAction confirms a Claude Code session is active, locates the
// claude process in this invocation's ancestry, and SIGTERMs it (or just
// prints the target under --dry-run).
func sessionEndAction(c *cli.Command, w io.Writer) error {
	sessionID, err := requireSessionID()
	if err != nil {
		return err
	}
	claudePID, err := findClaudePID(os.Getppid())
	if err != nil {
		return exitcode.New(exitcode.Generic, "session_end_no_target", err,
			"run `coily session end` directly from a Claude Code session's Bash tool, not from a nested shell")
	}
	if c.Bool("dry-run") {
		_, _ = fmt.Fprintf(w, "session end (dry-run): would send SIGTERM to claude pid %d (session %s)\n", claudePID, sessionID)
		return nil
	}
	_, _ = fmt.Fprintf(w, "session end: terminating Claude Code session %s (SIGTERM to claude pid %d)\n", sessionID, claudePID)
	if err := killProcess(claudePID); err != nil {
		return fmt.Errorf("session end: SIGTERM claude pid %d: %w", claudePID, err)
	}
	return nil
}

// maxAncestryHops bounds the parent-chain walk in findClaudePID so a
// malformed process tree cannot loop forever.
const maxAncestryHops = 32

// findClaudePID walks the process ancestry starting at startPID and
// returns the PID of the nearest ancestor (or startPID itself) whose
// command is the claude CLI. When coily runs from a Claude Code Bash
// tool the chain is claude -> shell -> coily, so startPID is the shell
// and its parent is claude. Errors if no claude process is found.
func findClaudePID(startPID int) (int, error) {
	pid := startPID
	for hops := 0; hops < maxAncestryHops && pid > 1; hops++ {
		ppid, comm, err := processParent(pid)
		if err != nil {
			return 0, err
		}
		if isClaudeComm(comm) {
			return pid, nil
		}
		pid = ppid
	}
	return 0, errors.New("no claude process found in this invocation's ancestry")
}

// isClaudeComm reports whether a `ps`-reported command names the claude
// CLI. macOS prints a full path, Linux prints the bare command, and a
// login shell shows a leading '-'; normalize all three before matching.
func isClaudeComm(comm string) bool {
	base := filepath.Base(strings.TrimPrefix(strings.TrimSpace(comm), "-"))
	return base == "claude"
}

// processParent returns the parent PID and command name of pid. It shells
// out to `ps`, which is present and accepts this flag shape on both macOS
// and Linux (the two platforms that run dispatched sidequests). Declared
// as a var so tests can swap in a fake process tree.
var processParent = func(pid int) (ppid int, comm string, err error) {
	out, err := exec.Command("ps", "-o", "ppid=,comm=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return 0, "", fmt.Errorf("ps -p %d: %w", pid, err)
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) < 2 {
		return 0, "", fmt.Errorf("ps -p %d: unexpected output %q", pid, string(out))
	}
	ppid, err = strconv.Atoi(fields[0])
	if err != nil {
		return 0, "", fmt.Errorf("ps -p %d: parse ppid %q: %w", pid, fields[0], err)
	}
	return ppid, strings.Join(fields[1:], " "), nil
}

// killProcess sends SIGTERM to pid. Declared as a var so tests can
// record the target without ending a real process.
var killProcess = func(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Signal(syscall.SIGTERM)
}

// sessionUseAction validates argv, ensures the env var is set, then
// writes the sentinel. Profile names are validated against a narrow
// kebab-case allowlist so a malformed name cannot land on disk and
// confuse phase 3's loader.
func sessionUseAction(c *cli.Command) error {
	args := c.Args().Slice()
	if len(args) != 1 {
		return exitcode.New(exitcode.UserError, "user_error",
			errors.New("session use: exactly one positional argument required: <profile-name>"),
			"example: `coily session use mobile`")
	}
	name := args[0]
	if err := validateProfileName(name); err != nil {
		return exitcode.New(exitcode.UserError, "user_error", err,
			"profile names are lowercase kebab-case (a-z0-9-), 1-40 chars")
	}
	sessionID, err := requireSessionID()
	if err != nil {
		return err
	}
	target, err := config.SessionProfilePath(sessionID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return fmt.Errorf("session use: mkdir: %w", err)
	}
	if err := os.WriteFile(target, []byte(name+"\n"), 0o600); err != nil {
		return fmt.Errorf("session use: write %s: %w", target, err)
	}
	fmt.Fprintf(os.Stderr, "session use: wrote %s\n", target)
	return nil
}

// sessionShowAction prints the active profile and its resolved
// Coordinate. The Coordinate comes from profiles.Resolve so the
// operator sees exactly what tiers a future gate evaluation would
// apply, including the source line that distinguishes
// "override-hit" from "deny-everything fallback".
func sessionShowAction(w io.Writer) error {
	sessionID, err := requireSessionID()
	if err != nil {
		return err
	}
	target, err := config.SessionProfilePath(sessionID)
	if err != nil {
		return err
	}
	var sentinelName string
	active, readErr := readSentinel(target)
	switch {
	case errors.Is(readErr, os.ErrNotExist):
		active = "(unset; run `coily session use <profile>` to set)"
	case readErr != nil:
		return readErr
	default:
		sentinelName = active
	}
	res, err := profiles.Resolve(sentinelName)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(w, "session_id: %s\n", sessionID)
	_, _ = fmt.Fprintf(w, "sentinel: %s\n", target)
	_, _ = fmt.Fprintf(w, "active_profile: %s\n", active)
	_, _ = fmt.Fprintf(w, "source: %s\n", res.Source)
	_, _ = fmt.Fprintf(w, "note: %s\n", res.Note)
	_, _ = fmt.Fprintln(w, "axes:")
	_, _ = fmt.Fprintf(w, "  data_security: %s\n", res.Coord.DataSecurity)
	_, _ = fmt.Fprintf(w, "  blast_radius: %s\n", res.Coord.BlastRadius)
	_, _ = fmt.Fprintf(w, "  network_egress: %s\n", res.Coord.NetworkEgress)
	_, _ = fmt.Fprintf(w, "  filesystem_reach: %s\n", res.Coord.FilesystemReach)
	return nil
}

func sessionClearAction() error {
	sessionID, err := requireSessionID()
	if err != nil {
		return err
	}
	target, err := config.SessionProfilePath(sessionID)
	if err != nil {
		return err
	}
	if err := os.Remove(target); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "session clear: %s already absent\n", target)
			return nil
		}
		return fmt.Errorf("session clear: %w", err)
	}
	fmt.Fprintf(os.Stderr, "session clear: removed %s\n", target)
	return nil
}

// requireSessionID resolves $CLAUDE_CODE_SESSION_ID. Returns a
// UserError if unset since the session verbs are meaningless outside
// a Claude Code-managed lifecycle.
func requireSessionID() (string, error) {
	sid := strings.TrimSpace(os.Getenv(sessionEnvVar))
	if sid == "" {
		return "", exitcode.New(exitcode.UserError, "user_error",
			errors.New("session: $"+sessionEnvVar+" is not set"),
			"run inside a Claude Code session, or export "+sessionEnvVar+"=<id> to simulate one")
	}
	if err := validateSessionID(sid); err != nil {
		return "", err
	}
	return sid, nil
}

// readSentinel reads the profile-name sentinel and trims trailing
// whitespace. Errors propagate (caller distinguishes ErrNotExist).
func readSentinel(path string) (string, error) {
	b, err := os.ReadFile(path) //nolint:gosec // path derived from $CLAUDE_CODE_SESSION_ID via config.SessionProfilePath
	if err != nil {
		return "", err
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return "", fmt.Errorf("session: sentinel at %s is empty", path)
	}
	if err := validateProfileName(s); err != nil {
		return "", fmt.Errorf("session: sentinel at %s: %w", path, err)
	}
	return s, nil
}

// validateProfileName enforces the narrow kebab-case shape phase 3's
// coily.yaml loader will key on. Keeps the on-disk surface boring.
func validateProfileName(s string) error {
	if s == "" {
		return errors.New("profile name is empty")
	}
	if len(s) > 40 {
		return fmt.Errorf("profile name %q exceeds 40 chars", s)
	}
	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-':
			if i == 0 || i == len(s)-1 {
				return fmt.Errorf("profile name %q must not start or end with '-'", s)
			}
		default:
			return fmt.Errorf("profile name %q contains invalid char %q", s, r)
		}
	}
	return nil
}

// validateSessionID rejects anything that would let a value escape the
// per-session directory. Claude Code IDs are UUIDs in practice, but be
// defensive: the env var is attacker-controllable if a session ever
// runs untrusted input through it.
func validateSessionID(s string) error {
	if len(s) > 128 {
		return fmt.Errorf("session id exceeds 128 chars")
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_':
		default:
			return fmt.Errorf("session id contains invalid char %q", r)
		}
	}
	return nil
}

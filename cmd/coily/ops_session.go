package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/coilysiren/cli-guard/exitcode"
	"github.com/coilysiren/cli-guard/verb"
	"github.com/coilysiren/coily/pkg/config"
	"github.com/urfave/cli/v3"
)

// sessionEnvVar is the Claude Code-injected env var coily reads to bind
// session state. coily attaches; Claude Code owns the lifecycle.
const sessionEnvVar = "CLAUDE_CODE_SESSION_ID"

// wouldBeStrictest is the phase-2 placeholder for the active coordinate
// `coily session show` prints. Once cli-guard/profile lands in coily's
// go.mod (phase 3) this gets replaced by profile.Strictest() and the
// per-axis lookup. Kept as a string slice so the show output is stable
// across the two phases and the format does not churn when phase 3 wires
// up the real import.
var wouldBeStrictest = [][2]string{
	{"data_security", "max"},
	{"blast_radius", "low"},
	{"network_egress", "air-gapped"},
	{"filesystem_reach", "repo-only"},
}

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
		},
	}
}

func (r *Runner) sessionUseCommand() *cli.Command {
	return &cli.Command{
		Name:      "use",
		Usage:     "Record the active lockdown profile for this Claude Code session.",
		ArgsUsage: "<profile-name>",
		Action: verb.Wrap(
			verb.Spec{
				Name:      "session.use",
				SkipScope: true,
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
		Action: verb.Wrap(
			verb.Spec{
				Name:      "session.show",
				SkipScope: true,
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
		Action: verb.Wrap(
			verb.Spec{
				Name:      "session.clear",
				SkipScope: true,
				Action: func(_ context.Context, _ *cli.Command) error {
					return sessionClearAction()
				},
			},
			r.Audit,
		),
	}
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

// sessionShowAction prints active profile + would-be strictest tiers.
// In phase 2 the tier list is hardcoded; phase 3 replaces with
// profile.Strictest() once cli-guard/profile is in go.mod.
func sessionShowAction(w io.Writer) error {
	sessionID, err := requireSessionID()
	if err != nil {
		return err
	}
	target, err := config.SessionProfilePath(sessionID)
	if err != nil {
		return err
	}
	active, readErr := readSentinel(target)
	switch {
	case errors.Is(readErr, os.ErrNotExist):
		active = "(unset; run `coily session use <profile>` to set)"
	case readErr != nil:
		return readErr
	}
	_, _ = fmt.Fprintf(w, "session_id: %s\n", sessionID)
	_, _ = fmt.Fprintf(w, "sentinel: %s\n", target)
	_, _ = fmt.Fprintf(w, "active_profile: %s\n", active)
	_, _ = fmt.Fprintln(w, "would_be_axes:")
	for _, kv := range wouldBeStrictest {
		_, _ = fmt.Fprintf(w, "  %s: %s\n", kv[0], kv[1])
	}
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

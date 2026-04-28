package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/coilysiren/coily/pkg/sudo"
	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

// deployTarget binds a deploy verb name to the on-server (repo, install
// script) pair it drives. The table is hardcoded; there is no free-form
// "run any installer" verb because that would re-open the same
// arbitrary-exec hole that the rest of `coily ssh` exists to close.
type deployTarget struct {
	// Name is the user-facing leaf, e.g. `repo-recall` for
	// `coily ssh deploy repo-recall`.
	Name string
	// RepoDir is the absolute path on the remote host of the git
	// checkout that should be fast-forwarded before the installer runs.
	RepoDir string
	// Script is the absolute path on the remote host of the install
	// script that runs as root via sudo. Each entry needs a matching
	// NOPASSWD sudoers line for zero-prompt repeat deploys; mismatch
	// fails closed (the -n attempt is denied and we fall back to
	// interactive prompt).
	Script string
}

// deployTargets is the closed allowlist of deploy verbs. New entries
// land here as new lines (and a matching sudoers fragment in
// infrastructure).
var deployTargets = []deployTarget{
	{
		Name:    "repo-recall",
		RepoDir: "/home/kai/projects/coilysiren/repo-recall",
		Script:  "/home/kai/projects/coilysiren/infrastructure/scripts/install-repo-recall.sh",
	},
	{
		// eco-telemetry deploys from the latest GitHub release zip,
		// not from a source checkout. RepoDir points at infrastructure
		// so the install script itself is fast-forwarded on each
		// deploy; the install script then curls the release asset.
		Name:    "eco-telemetry",
		RepoDir: "/home/kai/projects/coilysiren/infrastructure",
		Script:  "/home/kai/projects/coilysiren/infrastructure/scripts/install-eco-telemetry.sh",
	},
}

func (r *Runner) sshDeployCommand() *cli.Command {
	cmds := make([]*cli.Command, 0, len(deployTargets))
	for _, t := range deployTargets {
		cmds = append(cmds, r.sshDeployTarget(t))
	}
	return &cli.Command{
		Name:  "deploy",
		Usage: "Pull source and run the installer for a known deploy target.",
		Description: `deploy fast-forwards a checkout (git pull --ff-only) and then
runs an install script as root. Targets are a closed allowlist; the
script path is never user-supplied. Sudo elevation is tried as
non-interactive (-n) first; if NOPASSWD is not configured for the
script, deploy falls back to prompting on /dev/tty and piping the
password into a fresh sudo -S over the ssh transport. The password
never appears in argv or audit.`,
		Commands: cmds,
	}
}

func (r *Runner) sshDeployTarget(t deployTarget) *cli.Command {
	return &cli.Command{
		Name:  t.Name,
		Usage: fmt.Sprintf("Fast-forward %s and run %s as root.", t.RepoDir, t.Script),
		Flags: r.sshHostUserFlags(),
		Action: verb.Wrap(
			verb.Spec{
				Name: "ssh.deploy." + t.Name,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--host": c.String("host"),
						"--user": c.String("user"),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ssh deploy %s: takes no positional args, got %d", t.Name, c.Args().Len())
					}
					if err := validateRepoPath(t.RepoDir); err != nil {
						return fmt.Errorf("ssh deploy %s: repo dir invalid: %w", t.Name, err)
					}
					if err := validateRepoPath(t.Script); err != nil {
						return fmt.Errorf("ssh deploy %s: script path invalid: %w", t.Name, err)
					}
					host, user, err := sshTarget(c)
					if err != nil {
						return err
					}
					return r.runDeploy(ctx, host, user, t)
				},
			},
			r.Audit,
		),
	}
}

// runDeploy is the body of every deploy leaf: fast-forward the checkout,
// then run the installer as root with the -n / -S sudo dance.
func (r *Runner) runDeploy(ctx context.Context, host, user string, t deployTarget) error {
	// Phase 1: git pull --ff-only as the ssh user (no sudo).
	pullArgv := []string{"git", "-C", t.RepoDir, "pull", "--ff-only"}
	fmt.Fprintf(os.Stderr, "deploy %s: %s\n", t.Name, strings.Join(pullArgv, " "))
	if err := r.SSH.Stream(ctx, host, user, strings.Join(pullArgv, " "), os.Stdout, os.Stderr); err != nil {
		return fmt.Errorf("ssh deploy %s: git pull: %w", t.Name, err)
	}

	// Phase 2a: try sudo -n. If NOPASSWD covers the script, this
	// streams installer output and returns the script's exit status.
	nArgv := []string{"sudo", "-n", "bash", t.Script}
	fmt.Fprintf(os.Stderr, "deploy %s: %s\n", t.Name, strings.Join(nArgv, " "))
	var errBuf bytes.Buffer
	stderrTee := io.MultiWriter(os.Stderr, &errBuf)
	err := r.SSH.Stream(ctx, host, user, strings.Join(nArgv, " "), os.Stdout, stderrTee)
	if err == nil {
		return nil
	}

	// Distinguish "sudo -n needs a password" from "the installer ran
	// and failed". sudo -n exits with the script's status code when it
	// runs the script; only the password-required path matches the
	// sentinel strings.
	if !sudo.PasswordRequired(errBuf.String()) {
		return fmt.Errorf("ssh deploy %s: install (sudo -n): %w", t.Name, err)
	}

	// Phase 2b: prompt on /dev/tty and pipe the password into a fresh
	// `sudo -S` over the ssh transport. The password reaches sudo over
	// stdin (encrypted by ssh), never appears in argv (no ps leak), and
	// never reaches the audit log (which records argv only).
	fmt.Fprintf(os.Stderr, "deploy %s: sudo -n denied, falling back to interactive password\n", t.Name)
	pw, err := sudo.ReadPassword("[sudo via coily] password: ")
	if err != nil {
		if errors.Is(err, sudo.ErrNoTTY) {
			return fmt.Errorf("ssh deploy %s: sudo -n denied and no tty for fallback prompt: %w", t.Name, err)
		}
		return fmt.Errorf("ssh deploy %s: read password: %w", t.Name, err)
	}
	defer sudo.Zero(pw)

	// `-S` reads the password from stdin; `-p ''` suppresses the local
	// "[sudo] password:" prompt (ours already appeared on /dev/tty).
	// Append a newline so sudo's getline() returns.
	sArgv := []string{"sudo", "-S", "-p", "''", "bash", t.Script}
	fmt.Fprintf(os.Stderr, "deploy %s: %s\n", t.Name, strings.Join(sArgv, " "))
	pwLine := append(append([]byte{}, pw...), '\n')
	defer sudo.Zero(pwLine)
	if err := r.SSH.StreamStdin(ctx, host, user, strings.Join(sArgv, " "), bytes.NewReader(pwLine), os.Stdout, os.Stderr); err != nil {
		return fmt.Errorf("ssh deploy %s: install (sudo -S): %w", t.Name, err)
	}
	return nil
}

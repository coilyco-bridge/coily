package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/coilysiren/cli-guard/verb"
	"github.com/coilysiren/coily/pkg/sudo"
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
	// script. When NoSudo is false the script is invoked as
	// `sudo bash <script>` and needs a matching NOPASSWD sudoers line
	// for zero-prompt repeat deploys; when NoSudo is true the script
	// runs as the ssh user.
	Script string
	// ScriptArgs are positional arguments appended after the script
	// path (e.g. a mod name for the parameterized eco-mod installers).
	// Each element must already pass policy.ValidateArg before reaching
	// here; runDeploy does not re-validate.
	ScriptArgs []string
	// NoSudo skips the sudo dance and runs the install script as the
	// plain ssh user. Use this for installers whose file ops all live
	// in user-owned paths (e.g. dropping a mod into /home/kai/...).
	NoSudo bool
}

// deployTargets is the closed allowlist of deploy verbs. Parameterized
// shapes (eco-mod, eco-mod-source) are wired separately in
// sshDeployCommand because they take a positional arg.
var deployTargets = []deployTarget{
	{
		Name:    "repo-recall",
		RepoDir: "/home/kai/projects/coilysiren/repo-recall",
		Script:  "/home/kai/projects/coilysiren/infrastructure/scripts/install-repo-recall.sh",
	},
}

func (r *Runner) sshDeployCommand() *cli.Command {
	cmds := make([]*cli.Command, 0, len(deployTargets)+2)
	for _, t := range deployTargets {
		cmds = append(cmds, r.sshDeployTarget(t))
	}
	cmds = append(cmds, r.sshDeployEcoModCommand(), r.sshDeployEcoModSourceCommand())
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

// sshDeployEcoModCommand wires `coily ssh deploy eco-mod NAME`. NAME is
// the mod name as published in a coilysiren GitHub release (asset glob
// NAME-*.zip). The install script looks across coilysiren/eco-mods,
// coilysiren/eco-mods-public, and coilysiren/NAME for matching release
// assets and unzips each into the EcoServer tree; unzip -o handles
// merging when the same mod ships from multiple repos. NoSudo is
// implicit because everything lands under /home/kai/.
func (r *Runner) sshDeployEcoModCommand() *cli.Command {
	const (
		repoDir = "/home/kai/projects/coilysiren/infrastructure"
		script  = "/home/kai/projects/coilysiren/infrastructure/scripts/install-eco-mod.sh"
	)
	return &cli.Command{
		Name:      "eco-mod",
		Usage:     "Fetch the latest release zip(s) of <name> and unzip into the EcoServer tree.",
		ArgsUsage: "<name>",
		Description: `eco-mod NAME deploys a release-zip Eco mod by fast-forwarding the
infrastructure checkout and running install-eco-mod.sh NAME on the
server. The script looks for NAME-*.zip release assets across
coilysiren/eco-mods, coilysiren/eco-mods-public, and coilysiren/NAME
(whichever exist), downloads each match, and unzips them into
config.eco.server_dir. The asset existing on a coilysiren-owned repo IS
the security check; coily does not maintain a per-mod allowlist.

Files land under kai-owned paths so no sudo is involved. The verb does
not restart the Eco server - run 'coily gaming eco restart' separately when
the new mod should take effect.`,
		Flags: r.sshHostUserFlags(),
		Action: verb.Wrap(
			verb.Spec{
				Name: "ssh.deploy.eco-mod",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--host": c.String("host"),
						"--user": c.String("user"),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 1 {
						return fmt.Errorf("ssh deploy eco-mod: takes exactly one positional <name>, got %d", c.Args().Len())
					}
					name := c.Args().First()
					if err := validateEcoModName(name); err != nil {
						return fmt.Errorf("ssh deploy eco-mod: %w", err)
					}
					t := deployTarget{
						Name:       "eco-mod",
						RepoDir:    repoDir,
						Script:     script,
						ScriptArgs: []string{name},
						NoSudo:     true,
					}
					if err := validateRepoPath(t.RepoDir); err != nil {
						return fmt.Errorf("ssh deploy eco-mod: repo dir invalid: %w", err)
					}
					if err := validateRepoPath(t.Script); err != nil {
						return fmt.Errorf("ssh deploy eco-mod: script path invalid: %w", err)
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

// sshDeployEcoModSourceCommand wires `coily ssh deploy eco-mod-source
// NAME`. Source-tree mods (the ones that live under Mods/UserCode/<NAME>
// in coilysiren/eco-mods or coilysiren/eco-mods-public) deploy by
// rsync, not by release zip - the install script fast-forwards each of
// the two source repos that contains a matching dir and copies the
// subtree into config.eco.server_dir. eco-configs is intentionally not
// covered: configs are pushed by a different surface.
func (r *Runner) sshDeployEcoModSourceCommand() *cli.Command {
	const (
		repoDir = "/home/kai/projects/coilysiren/infrastructure"
		script  = "/home/kai/projects/coilysiren/infrastructure/scripts/install-eco-mod-source.sh"
	)
	return &cli.Command{
		Name:      "eco-mod-source",
		Usage:     "Rsync the source-tree Eco mod <name> from eco-mods / eco-mods-public into the EcoServer tree.",
		ArgsUsage: "<name>",
		Description: `eco-mod-source NAME deploys a source-tree Eco mod by fast-forwarding
the infrastructure checkout and running install-eco-mod-source.sh NAME
on the server. The script looks for Mods/UserCode/NAME or Mods/NAME
inside ~/projects/coilysiren/eco-mods and ~/projects/coilysiren/eco-mods-public,
fast-forwards each repo that has a match, and rsyncs the matched
subtree into config.eco.server_dir.

Files land under kai-owned paths so no sudo is involved. The verb does
not restart the Eco server - run 'coily gaming eco restart' separately when
the new mod should take effect.`,
		Flags: r.sshHostUserFlags(),
		Action: verb.Wrap(
			verb.Spec{
				Name: "ssh.deploy.eco-mod-source",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--host": c.String("host"),
						"--user": c.String("user"),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 1 {
						return fmt.Errorf("ssh deploy eco-mod-source: takes exactly one positional <name>, got %d", c.Args().Len())
					}
					name := c.Args().First()
					if err := validateEcoModName(name); err != nil {
						return fmt.Errorf("ssh deploy eco-mod-source: %w", err)
					}
					t := deployTarget{
						Name:       "eco-mod-source",
						RepoDir:    repoDir,
						Script:     script,
						ScriptArgs: []string{name},
						NoSudo:     true,
					}
					if err := validateRepoPath(t.RepoDir); err != nil {
						return fmt.Errorf("ssh deploy eco-mod-source: repo dir invalid: %w", err)
					}
					if err := validateRepoPath(t.Script); err != nil {
						return fmt.Errorf("ssh deploy eco-mod-source: script path invalid: %w", err)
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

// validateEcoModName constrains <name> to the shape a real Eco mod
// directory and GitHub release asset can have: ASCII letters, digits,
// underscore, hyphen, dot. No leading dash (would parse as a flag), no
// shell metacharacters, no path separators, length cap. The script is
// hardcoded; only this token reaches argv, so the bound here is the
// effective bound on what the parameterized leaves accept.
func validateEcoModName(name string) error {
	if name == "" {
		return fmt.Errorf("eco mod name is empty")
	}
	if len(name) > 128 {
		return fmt.Errorf("eco mod name too long")
	}
	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("eco mod name must not start with '-'")
	}
	for _, r := range name {
		if !ecoModNameByte(r) {
			return fmt.Errorf("eco mod name contains invalid character %q", r)
		}
	}
	return nil
}

func ecoModNameByte(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z',
		r >= 'A' && r <= 'Z',
		r >= '0' && r <= '9',
		r == '_', r == '-', r == '.':
		return true
	}
	return false
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

	// Phase 2 (no-sudo path): just run the script as the ssh user.
	if t.NoSudo {
		runArgv := append([]string{"bash", t.Script}, t.ScriptArgs...)
		fmt.Fprintf(os.Stderr, "deploy %s: %s\n", t.Name, strings.Join(runArgv, " "))
		if err := r.SSH.Stream(ctx, host, user, strings.Join(runArgv, " "), os.Stdout, os.Stderr); err != nil {
			return fmt.Errorf("ssh deploy %s: install: %w", t.Name, err)
		}
		return nil
	}

	// Phase 2a: try sudo -n. If NOPASSWD covers the script, this
	// streams installer output and returns the script's exit status.
	nArgv := append([]string{"sudo", "-n", "bash", t.Script}, t.ScriptArgs...)
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
	sArgv := append([]string{"sudo", "-S", "-p", "''", "bash", t.Script}, t.ScriptArgs...)
	fmt.Fprintf(os.Stderr, "deploy %s: %s\n", t.Name, strings.Join(sArgv, " "))
	pwLine := append(append([]byte{}, pw...), '\n')
	defer sudo.Zero(pwLine)
	if err := r.SSH.StreamStdin(ctx, host, user, strings.Join(sArgv, " "), bytes.NewReader(pwLine), os.Stdout, os.Stderr); err != nil {
		return fmt.Errorf("ssh deploy %s: install (sudo -S): %w", t.Name, err)
	}
	return nil
}

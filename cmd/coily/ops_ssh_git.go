package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

// gitVerbs is the closed set of git operations exposed through coily ssh git.
// Each leaf takes exactly one validated <repo-path> and runs as the configured
// ssh user (no sudo). Mirrors the systemctl pattern: fixed argv shapes, no
// free-form pass-through. New verbs land in this table.
var gitVerbs = []struct {
	Name  string
	Usage string
	// Argv builds the remote argv given the validated repo path.
	Argv func(path string) []string
}{
	{"pull", "Run git pull --ff-only in <repo-path>.", func(p string) []string {
		return []string{"git", "-C", p, "pull", "--ff-only"}
	}},
	{"fetch", "Run git fetch --all --prune in <repo-path>.", func(p string) []string {
		return []string{"git", "-C", p, "fetch", "--all", "--prune"}
	}},
	{"status", "Run git status --short --branch in <repo-path>.", func(p string) []string {
		return []string{"git", "-C", p, "status", "--short", "--branch"}
	}},
	{"log", "Run git log --oneline -n 20 in <repo-path>.", func(p string) []string {
		return []string{"git", "-C", p, "log", "--oneline", "-n", "20"}
	}},
	{"branch", "Run git branch --show-current in <repo-path>.", func(p string) []string {
		return []string{"git", "-C", p, "branch", "--show-current"}
	}},
	{"rev-parse", "Run git rev-parse HEAD in <repo-path>.", func(p string) []string {
		return []string{"git", "-C", p, "rev-parse", "HEAD"}
	}},
	{"diff", "Run git diff in <repo-path> (read-only, unstaged changes).", func(p string) []string {
		return []string{"git", "-C", p, "diff"}
	}},
	{"update-index", "Run git update-index --really-refresh in <repo-path> (re-hashes file contents to clear stat-stale phantom-dirty entries; mutates only the index's cached-stat column).", func(p string) []string {
		return []string{"git", "-C", p, "update-index", "--really-refresh"}
	}},
}

func (r *Runner) sshGitCommand() *cli.Command {
	cmds := make([]*cli.Command, 0, len(gitVerbs))
	for _, v := range gitVerbs {
		cmds = append(cmds, r.sshGitVerb(v.Name, v.Usage, v.Argv))
	}
	return &cli.Command{
		Name:        "git",
		Usage:       "Run a fixed-shape git verb on the remote.",
		Description: "Each leaf maps to one git call against a validated <repo-path>, run as the ssh user (no sudo). Mirrors common git verb names; no free-form passthrough.",
		Commands:    cmds,
	}
}

func (r *Runner) sshGitVerb(name, usage string, build func(string) []string) *cli.Command {
	return &cli.Command{
		Name:      name,
		Usage:     usage,
		ArgsUsage: "<repo-path>",
		Flags:     r.sshHostUserFlags(),
		Action: verb.Wrap(
			verb.Spec{
				Name: "ssh.git." + name,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--host": c.String("host"),
						"--user": c.String("user"),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 1 {
						return fmt.Errorf("ssh git %s: need exactly one <repo-path> arg, got %d", name, c.Args().Len())
					}
					path := c.Args().First()
					if err := validateRepoPath(path); err != nil {
						return err
					}
					host, user, err := sshTarget(c)
					if err != nil {
						return err
					}
					argv := build(path)
					return r.SSH.Stream(ctx, host, user, strings.Join(argv, " "), os.Stdout, os.Stderr)
				},
			},
			r.Audit,
		),
	}
}

// validateRepoPath rejects anything that isn't a sane absolute filesystem
// path safe to interpolate after `git -C`. verb.Wrap already rejects shell
// metacharacters in argv; this adds the path-shape constraints (absolute,
// no `..`, no embedded whitespace, no leading `-`, length cap).
func validateRepoPath(path string) error {
	if path == "" {
		return fmt.Errorf("ssh git: repo path is empty")
	}
	if len(path) > 4096 {
		return fmt.Errorf("ssh git: repo path too long")
	}
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("ssh git: repo path must be absolute (start with '/')")
	}
	if strings.HasPrefix(path, "-") {
		return fmt.Errorf("ssh git: repo path must not start with '-'")
	}
	for _, r := range path {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return fmt.Errorf("ssh git: repo path must not contain whitespace")
		}
	}
	for _, seg := range strings.Split(path, "/") {
		if seg == ".." {
			return fmt.Errorf("ssh git: repo path must not contain '..' segments")
		}
	}
	return nil
}

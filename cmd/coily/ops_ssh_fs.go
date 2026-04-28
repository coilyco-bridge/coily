package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

// fsVerbs is the closed set of read-only filesystem inspection commands
// exposed flat under `coily ssh`. Each leaf takes exactly one validated
// <path> and runs as the configured ssh user (no sudo). Mirrors the
// systemctl / git pattern: fixed argv shapes, no free-form pass-through.
// Mutating shell ops (rm, mv, cp, sed -i, etc.) intentionally do not
// land here. Filesystem changes go through `coily ssh deploy` or
// install scripts.
var fsVerbs = []struct {
	Name  string
	Usage string
	Argv  func(path string) []string
}{
	{"ls", "Run ls -la <path>.", func(p string) []string {
		return []string{"ls", "-la", "--", p}
	}},
	{"tree", "Run tree -L 2 <path> (depth-limited).", func(p string) []string {
		return []string{"tree", "-L", "2", "--", p}
	}},
	{"cat", "Run cat <path>.", func(p string) []string {
		return []string{"cat", "--", p}
	}},
	{"head", "Run head <path> (first 10 lines).", func(p string) []string {
		return []string{"head", "--", p}
	}},
	{"tail", "Run tail <path> (last 10 lines).", func(p string) []string {
		return []string{"tail", "--", p}
	}},
	{"wc", "Run wc <path>.", func(p string) []string {
		return []string{"wc", "--", p}
	}},
	{"file", "Run file <path>.", func(p string) []string {
		return []string{"file", "--", p}
	}},
}

// sshFsCommands returns the flat list of fs leaves to splice into
// sshCommand's Commands slice.
func (r *Runner) sshFsCommands() []*cli.Command {
	cmds := make([]*cli.Command, 0, len(fsVerbs)+1)
	for _, v := range fsVerbs {
		cmds = append(cmds, r.sshFsVerb(v.Name, v.Usage, v.Argv))
	}
	cmds = append(cmds, r.sshGrepCommand())
	return cmds
}

func (r *Runner) sshFsVerb(name, usage string, build func(string) []string) *cli.Command {
	return &cli.Command{
		Name:      name,
		Usage:     usage,
		ArgsUsage: "<path>",
		Flags:     r.sshHostUserFlags(),
		Action: verb.Wrap(
			verb.Spec{
				Name: "ssh." + name,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
							"--host": c.String("host"),
							"--user": c.String("user"),
						},
						c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 1 {
						return fmt.Errorf("ssh %s: need exactly one <path> arg, got %d", name, c.Args().Len())
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

// sshGrepCommand exposes a constrained `grep -F -- '<pattern>' <path>`
// readonly verb. Fixed-string only (no regex) so policy's
// metacharacter-rejection on the pattern does not turn into "grep
// silently doesn't match anything". The pattern is wrapped in single
// quotes on the remote, so single quotes in the pattern are rejected
// up front; leading dashes are rejected to avoid flag confusion (the
// `--` belt-and-suspenders the same property).
func (r *Runner) sshGrepCommand() *cli.Command {
	return &cli.Command{
		Name:      "grep",
		Usage:     "Run grep -F -- '<pattern>' <path> (fixed-string match).",
		ArgsUsage: "<pattern> <path>",
		Flags:     r.sshHostUserFlags(),
		Action: verb.Wrap(
			verb.Spec{
				Name: "ssh.grep",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
							"--host": c.String("host"),
							"--user": c.String("user"),
						},
						c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 2 {
						return fmt.Errorf("ssh grep: need <pattern> <path>, got %d arg(s)", c.Args().Len())
					}
					pattern := c.Args().Get(0)
					path := c.Args().Get(1)
					if err := validateGrepPattern(pattern); err != nil {
						return err
					}
					if err := validateRepoPath(path); err != nil {
						return err
					}
					host, user, err := sshTarget(c)
					if err != nil {
						return err
					}
					argv := []string{"grep", "-F", "--", "'" + pattern + "'", path}
					return r.SSH.Stream(ctx, host, user, strings.Join(argv, " "), os.Stdout, os.Stderr)
				},
			},
			r.Audit,
		),
	}
}

// validateGrepPattern is the pattern-side counterpart to validateRepoPath.
// verb.Wrap already rejects shell metacharacters in argv; this layer adds
// the "safe to wrap in single quotes" constraint and a length cap.
func validateGrepPattern(p string) error {
	if p == "" {
		return fmt.Errorf("ssh grep: pattern is empty")
	}
	if len(p) > 1024 {
		return fmt.Errorf("ssh grep: pattern too long")
	}
	if strings.Contains(p, "'") {
		return fmt.Errorf("ssh grep: pattern must not contain a single quote")
	}
	if strings.HasPrefix(p, "-") {
		return fmt.Errorf("ssh grep: pattern must not start with '-'")
	}
	return nil
}

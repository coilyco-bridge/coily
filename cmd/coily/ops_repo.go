package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/coilysiren/coily/pkg/repocfg"
	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

// loadRepoExecCommand discovers coily.yaml relative to cwd and returns the
// loaded *repocfg.Config (or nil when no config was found) along with an
// `exec` cli.Command whose subcommands are the entries from coily.yaml. A
// missing config returns (nil, nil); the caller skips wiring `exec` in that
// case so it does not appear as a no-op verb in --help.
func (r *Runner) loadRepoExecCommand() (*repocfg.Config, *cli.Command) {
	cfg, err := repocfg.LoadDefault()
	if err != nil {
		if errors.Is(err, repocfg.ErrNoConfig) {
			return nil, nil
		}
		fmt.Fprintf(os.Stderr, "coily: repo config error: %v\n", err)
		return nil, nil
	}
	subs := make([]*cli.Command, 0, len(cfg.Commands))
	for _, rc := range cfg.Commands {
		subs = append(subs, r.buildRepoCommand(cfg, rc))
	}
	exec := &cli.Command{
		Name:     "exec",
		Usage:    "Run a named command from .coily/coily.yaml",
		Category: "repo",
		Description: fmt.Sprintf(
			"Run a per-repo command declared in %s. Subcommand names come from "+
				"the commands: map. Extra positional args are appended and validated "+
				"against the same shell-metacharacter rules as privileged verbs.",
			cfg.Path,
		),
		Commands: subs,
	}
	return cfg, exec
}

// buildRepoCommand turns one repocfg.Command into a cli.Command whose Action
// exec's the declared argv plus any user-supplied positional args. Everything
// runs through verb.Wrap so policy validation and audit logging apply.
func (r *Runner) buildRepoCommand(cfg *repocfg.Config, rc repocfg.Command) *cli.Command {
	usage := rc.Description
	if usage == "" {
		usage = "Repo command: " + strings.Join(rc.Argv, " ")
	}
	return &cli.Command{
		Name:      rc.Name,
		Usage:     usage,
		ArgsUsage: "[-- extra args]",
		Description: fmt.Sprintf(
			"Per-repo command loaded from %s.\nExpands to: %s\n\nExtra positional args are appended and validated against the same "+
				"shell-metacharacter rules as privileged verbs.",
			cfg.Path, strings.Join(rc.Argv, " "),
		),
		Action: verb.Wrap(
			verb.Spec{
				Name: "repo." + rc.Name,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					positional := append([]string{}, rc.Argv...)
					positional = append(positional, c.Args().Slice()...)
					return nil, positional
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					argv := append([]string{}, rc.Argv[1:]...)
					argv = append(argv, c.Args().Slice()...)
					return r.Runner.Exec(ctx, rc.Argv[0], argv...)
				},
			},
			r.Audit,
		),
	}
}

// listCommand renders the built-in and repo command inventory in one shot.
// Same output for --list on the root command; see main.go.
func listCommand(builtIns []*cli.Command, exec *cli.Command, repoCfg *repocfg.Config) {
	fmt.Println("Built-in commands:")
	printCmdGroup(builtIns)
	fmt.Println()
	if repoCfg == nil {
		fmt.Println("Repo commands (coily exec <name>):")
		fmt.Println("  (no coily.yaml found in the current directory or any parent)")
		return
	}
	fmt.Printf("Repo commands from %s (coily exec <name>):\n", repoCfg.Path)
	if exec == nil || len(exec.Commands) == 0 {
		fmt.Println("  (none declared)")
		return
	}
	printCmdGroup(exec.Commands)
}

// treeCommand renders every coily command and subcommand recursively.
// Same surfaces as --list, but walks the full subcommand tree instead of
// stopping at the top level.
func treeCommand(builtIns []*cli.Command, exec *cli.Command, repoCfg *repocfg.Config) {
	fmt.Println("Built-in commands:")
	printCmdTree(builtIns, "  ")
	fmt.Println()
	if repoCfg == nil {
		fmt.Println("Repo commands (coily exec <name>):")
		fmt.Println("  (no coily.yaml found in the current directory or any parent)")
		return
	}
	fmt.Printf("Repo commands from %s (coily exec <name>):\n", repoCfg.Path)
	if exec == nil || len(exec.Commands) == 0 {
		fmt.Println("  (none declared)")
		return
	}
	printCmdTree(exec.Commands, "  ")
}

func printCmdTree(cmds []*cli.Command, indent string) {
	for _, c := range cmds {
		if c.Hidden || c.Name == "help" {
			continue
		}
		if c.Usage != "" {
			fmt.Printf("%s%s  %s\n", indent, c.Name, c.Usage)
		} else {
			fmt.Printf("%s%s\n", indent, c.Name)
		}
		if len(c.Commands) > 0 {
			printCmdTree(c.Commands, indent+"  ")
		}
	}
}

func printCmdGroup(cmds []*cli.Command) {
	width := 0
	for _, c := range cmds {
		if len(c.Name) > width {
			width = len(c.Name)
		}
	}
	for _, c := range cmds {
		fmt.Printf("  %-*s  %s\n", width, c.Name, c.Usage)
	}
}

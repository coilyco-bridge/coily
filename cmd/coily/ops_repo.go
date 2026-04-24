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

// loadRepoCommands discovers coily.yaml relative to cwd and returns a
// *cli.Command for each non-shadowing entry along with the loaded
// *repocfg.Config (or nil when no config was found). Reserved names (those
// already registered as built-in verbs) are skipped with a stderr warning so
// an accidentally-named repo command cannot silently replace a privileged
// op. A missing config is not an error; returns (nil, nil).
func (r *Runner) loadRepoCommands(reserved map[string]bool) (*repocfg.Config, []*cli.Command) {
	cfg, err := repocfg.LoadDefault()
	if err != nil {
		if errors.Is(err, repocfg.ErrNoConfig) {
			return nil, nil
		}
		fmt.Fprintf(os.Stderr, "coily: repo config error: %v\n", err)
		return nil, nil
	}
	out := make([]*cli.Command, 0, len(cfg.Commands))
	for _, rc := range cfg.Commands {
		if reserved[rc.Name] {
			fmt.Fprintf(os.Stderr,
				"coily: %s: repo command %q shadows a built-in; skipping\n",
				cfg.Path, rc.Name)
			continue
		}
		out = append(out, r.buildRepoCommand(cfg, rc))
	}
	return cfg, out
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
		Category:  "repo",
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
					// Every token, declared and appended, is validated. argv
					// tokens were already checked at load time but re-checking
					// is cheap and keeps the security boundary uniform.
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
func listCommand(builtIns, repo []*cli.Command, repoCfg *repocfg.Config) {
	fmt.Println("Built-in commands:")
	printCmdGroup(builtIns)
	fmt.Println()
	if repoCfg == nil {
		fmt.Println("Repo commands:")
		fmt.Println("  (no coily.yaml found in the current directory or any parent)")
		return
	}
	fmt.Printf("Repo commands (from %s):\n", repoCfg.Path)
	if len(repo) == 0 {
		fmt.Println("  (none; every entry shadowed a built-in)")
		return
	}
	printCmdGroup(repo)
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

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
)

// Version is injected at build time via -ldflags "-X main.Version=<sha>".
var Version = "dev"

// devCommandBuilders is populated by init() in files with `//go:build dev`.
// Empty in prod builds. Each builder receives the Runner and returns a
// cli.Command. Kept separate from prod commands so the split is visible to
// readers and auditors.
var devCommandBuilders []func(*Runner) *cli.Command

func registerDevCommandBuilder(b func(*Runner) *cli.Command) {
	devCommandBuilders = append(devCommandBuilders, b)
}

func main() {
	r := NewRunner()
	if err := run(r, os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run wires a Runner into the urfave/cli v3 root command and executes it
// against argv. Split out from main() so tests can drive a Runner with fake
// dependencies through a real cli.Command tree.
func run(r *Runner, argv []string) error {
	builtIns := r.builtInCommands()
	for _, b := range devCommandBuilders {
		builtIns = append(builtIns, b(r))
	}

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
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "list",
				Usage: "print every command coily can run (built-in + repo) and exit",
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
// builder here.
func (r *Runner) builtInCommands() []*cli.Command {
	return []*cli.Command{
		r.versionCommand(),
		r.whoamiCommand(),
		r.authCommand(),
		r.lockdownCommand(),
		r.installCompletionCommand(),
		r.ecoCommand(),
		r.awsCommand(),
		r.ghCommand(),
		r.kubectlCommand(),
	}
}

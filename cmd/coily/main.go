package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
)

// Version is injected at build time via -ldflags "-X main.Version=<sha>".
var Version = "dev"

// registeredCommands is populated by init() functions in sibling files. Each
// verb (or verb tree) lives in its own file and self-registers here. This
// keeps main.go free of a central registration list, which is the thing
// parallel feature branches would otherwise conflict on.
//
// Add a command by writing a new file in this package that calls
// `registerCommand(myCmd)` from init().
var registeredCommands []*cli.Command

// devOnlyCommands is populated by init() in files with `//go:build dev`.
// Empty in prod builds. Kept separate so the split is visible to readers and
// auditors.
var devOnlyCommands []*cli.Command

func registerCommand(c *cli.Command)        { registeredCommands = append(registeredCommands, c) }
func registerDevOnlyCommand(c *cli.Command) { devOnlyCommands = append(devOnlyCommands, c) }

func main() {
	cmd := &cli.Command{
		Name:                  "coily",
		Usage:                 "Operator CLI for Kai's homelab.",
		Version:               Version,
		Commands:              append(append([]*cli.Command{}, registeredCommands...), devOnlyCommands...),
		EnableShellCompletion: true,
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

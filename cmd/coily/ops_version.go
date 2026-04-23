package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

// versionCommand prints the build version. ReadOnly, no audit, no token. The
// receiver is unused but kept for symmetry with the other Runner methods so
// main.go can wire every command the same way.
func (r *Runner) versionCommand() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Print the build version and exit.",
		Action: func(_ context.Context, _ *cli.Command) error {
			fmt.Println(Version)
			return nil
		},
	}
}

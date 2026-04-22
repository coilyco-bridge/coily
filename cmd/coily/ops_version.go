package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func init() {
	registerCommand(&cli.Command{
		Name:  "version",
		Usage: "Print the build version and exit.",
		Action: func(_ context.Context, _ *cli.Command) error {
			fmt.Println(Version)
			return nil
		},
	})
}

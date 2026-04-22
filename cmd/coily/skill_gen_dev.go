//go:build dev

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/coilysiren/coily/pkg/skillgen"
	"github.com/urfave/cli/v3"
)

// skill-gen is only registered in dev builds. Production coily cannot write
// skill files. Per docs/threat-model.md, dev-only conveniences are compiled
// out of the prod binary so an agent that lands on /usr/local/bin/coily
// cannot call them.
func init() { registerDevOnlyCommand(skillGenCmd) }

var skillGenCmd = &cli.Command{
	Name:  "skill-gen",
	Usage: "Regenerate skill/SKILL.md and skill/reference/*.md from configs/commands/*.yaml. (dev build only)",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "commands-dir",
			Usage: "directory of manifest yaml files produced by subcli-scope",
			Value: "configs/commands",
		},
		&cli.StringFlag{
			Name:  "out",
			Usage: "skill output directory",
			Value: "skill",
		},
	},
	Action: func(_ context.Context, c *cli.Command) error {
		opt := skillgen.Options{
			CommandsDir: c.String("commands-dir"),
			OutDir:      c.String("out"),
			Verbs:       handWrittenVerbs(),
		}
		if err := skillgen.Generate(opt); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "wrote %s/SKILL.md + reference/*.md\n", opt.OutDir)
		return nil
	},
}

// handWrittenVerbs returns metadata for coily's own (non-mirror) top-level
// subcommands. Hand-maintained here because the urfave/cli tree doesn't
// expose flag-name lists cleanly in v3, and this is short enough to list
// explicitly. Add an entry whenever a new coily-native verb lands.
func handWrittenVerbs() []skillgen.Verb {
	return []skillgen.Verb{
		{
			Name:    "lockdown",
			Usage:   "Write per-repo Claude Code permissions that force all ops through coily.",
			Flags:   []string{"--path", "--local", "--apply", "--replace"},
			Example: "coily lockdown --path . --apply",
		},
		{
			Name:    "whoami",
			Usage:   "Print the authenticated identity coily sees across aws, kubectl, and gh.",
			Example: "coily whoami",
		},
		{
			Name:    "version",
			Usage:   "Print the build version and exit.",
			Example: "coily version",
		},
	}
}

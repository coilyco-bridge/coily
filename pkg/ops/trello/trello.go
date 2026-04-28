// Package trello wraps the message-ops Trello CLI scripts (status, update,
// create) so coily can drive Kai's recruiter pipeline through the audit log.
//
// The underlying tooling is a Node project at ~/projects/coilysiren/message-ops
// invoked via `npm run trello:<verb>`. The checkout path is resolved from the
// --dir flag, then $COILY_MESSAGE_OPS_DIR, then the workspace default
// ~/projects/coilysiren/message-ops. npm is resolved via $PATH like every
// other binary coily shells out to.
//
// Argv is forwarded verbatim through `npm --prefix <dir> run trello:<verb> --`
// so the underlying scripts see the same flags they would from a direct shell
// invocation. Every flag value still passes through coily's policy gate, so
// shell metacharacters in (e.g.) a --comment will be rejected up-front.
package trello

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/coilysiren/coily/pkg/audit"
	"github.com/coilysiren/coily/pkg/shell"
	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

const (
	// EnvDir overrides the message-ops checkout location. Set when the
	// repo lives outside the workspace default (Windows host, atypical
	// layout, CI).
	EnvDir = "COILY_MESSAGE_OPS_DIR"

	// DefaultRelPath is the workspace-relative location of message-ops on
	// macOS / Linux. Joined with $HOME at resolve time. Windows users set
	// EnvDir; there is no second hard-coded path.
	DefaultRelPath = "projects/coilysiren/message-ops"
)

// Command returns the cli.Command tree for `coily trello`.
func Command(r *shell.Runner, w *audit.Writer) *cli.Command {
	return &cli.Command{
		Name:  "trello",
		Usage: "Trello recruiter-pipeline ops via message-ops scripts.",
		Commands: []*cli.Command{
			statusCmd(r, w),
			updateCmd(r, w),
			createCmd(r, w),
			sortCmd(r, w),
		},
	}
}

func sortCmd(r *shell.Runner, w *audit.Writer) *cli.Command {
	return &cli.Command{
		Name:  "sort",
		Usage: "Sort each list so cards with the given label rise to the top.",
		Flags: []cli.Flag{
			dirFlag(),
			&cli.StringFlag{Name: "label", Usage: "label that should rise to the top of every list", Value: "Waiting on Me"},
			&cli.BoolFlag{Name: "dry", Usage: "print the planned moves without writing"},
		},
		Action: verb.Wrap(
			verb.Spec{
				Name: "trello.sort",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					args := map[string]string{
						"--dir":   c.String("dir"),
						"--label": c.String("label"),
					}
					if c.Bool("dry") {
						args["--dry"] = "true"
					}
					return args, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					scriptArgs := []string{"--label", c.String("label")}
					if c.Bool("dry") {
						scriptArgs = append(scriptArgs, "--dry")
					}
					return runScript(ctx, r, c, "trello:sort", scriptArgs)
				},
			},
			w,
		),
	}
}

func statusCmd(r *shell.Runner, w *audit.Writer) *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "List Trello cards with their list, labels, and last activity.",
		Flags: []cli.Flag{
			dirFlag(),
		},
		Action: verb.Wrap(
			verb.Spec{
				Name: "trello.status",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return statusArgs(c), c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return runScript(ctx, r, c, "trello:status", nil)
				},
			},
			w,
		),
	}
}

func updateCmd(r *shell.Runner, w *audit.Writer) *cli.Command {
	return &cli.Command{
		Name:      "update",
		Usage:     "Mutate one Trello card (move list, toggle labels, append a comment, rename, close/reopen).",
		ArgsUsage: "<cardId>",
		Flags: []cli.Flag{
			dirFlag(),
			&cli.StringFlag{Name: "list", Usage: "move card to this list (exact name match, including any trailing space)"},
			&cli.StringFlag{Name: "label-on", Usage: "add this label to the card"},
			&cli.StringFlag{Name: "label-off", Usage: "remove this label from the card"},
			&cli.StringFlag{Name: "comment", Usage: "append a comment to the card"},
			&cli.StringFlag{Name: "name", Usage: "rename the card"},
			&cli.StringFlag{Name: "desc", Usage: "rewrite the card description"},
			&cli.BoolFlag{Name: "close", Usage: "archive the card"},
			&cli.BoolFlag{Name: "reopen", Usage: "unarchive the card"},
		},
		Action: verb.Wrap(
			verb.Spec{
				Name: "trello.update",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return updateArgs(c), c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() < 1 {
						return fmt.Errorf("trello update: need <cardId> as first positional arg")
					}
					return runScript(ctx, r, c, "trello:update", buildUpdateScriptArgs(c))
				},
			},
			w,
		),
	}
}

func createCmd(r *shell.Runner, w *audit.Writer) *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a new Trello card.",
		Flags: []cli.Flag{
			dirFlag(),
			&cli.StringFlag{Name: "list", Usage: "list to create the card in (exact name match, including any trailing space)"},
			&cli.StringFlag{Name: "name", Usage: "card name (typically a short company handle)"},
			&cli.StringFlag{Name: "desc", Usage: "card description"},
			&cli.StringSliceFlag{Name: "label", Usage: "labels to apply (repeat for multiple)"},
		},
		Action: verb.Wrap(
			verb.Spec{
				Name: "trello.create",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return createArgs(c), c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if !c.IsSet("list") || !c.IsSet("name") {
						return fmt.Errorf("trello create: --list and --name are required")
					}
					return runScript(ctx, r, c, "trello:create", buildCreateScriptArgs(c))
				},
			},
			w,
		),
	}
}

func createArgs(c *cli.Command) map[string]string {
	args := map[string]string{
		"--dir":  c.String("dir"),
		"--list": c.String("list"),
		"--name": c.String("name"),
		"--desc": c.String("desc"),
	}
	for i, v := range c.StringSlice("label") {
		args[fmt.Sprintf("--label[%d]", i)] = v
	}
	return args
}

func buildCreateScriptArgs(c *cli.Command) []string {
	out := []string{"--list", c.String("list"), "--name", c.String("name")}
	if c.IsSet("desc") {
		out = append(out, "--desc", c.String("desc"))
	}
	for _, label := range c.StringSlice("label") {
		out = append(out, "--label", label)
	}
	return out
}

func dirFlag() cli.Flag {
	return &cli.StringFlag{
		Name:  "dir",
		Usage: "path to the message-ops checkout. Defaults to $" + EnvDir + " or ~/" + DefaultRelPath,
	}
}

func statusArgs(c *cli.Command) map[string]string {
	return map[string]string{"--dir": c.String("dir")}
}

func updateArgs(c *cli.Command) map[string]string {
	args := map[string]string{
		"--dir":       c.String("dir"),
		"--list":      c.String("list"),
		"--label-on":  c.String("label-on"),
		"--label-off": c.String("label-off"),
		"--comment":   c.String("comment"),
		"--name":      c.String("name"),
		"--desc":      c.String("desc"),
	}
	if c.Bool("close") {
		args["--close"] = "true"
	}
	if c.Bool("reopen") {
		args["--reopen"] = "true"
	}
	return args
}

// buildUpdateScriptArgs assembles the argv tail forwarded to update.js. The
// cardId comes in as the first positional argument; remaining flags map 1:1.
// IsSet checks are used so a zero-value flag doesn't get forwarded as
// `--list ""`, which update.js would treat as an explicit empty rename.
func buildUpdateScriptArgs(c *cli.Command) []string {
	out := []string{c.Args().First()}
	for _, name := range []string{"list", "label-on", "label-off", "comment", "name", "desc"} {
		if c.IsSet(name) {
			out = append(out, "--"+name, c.String(name))
		}
	}
	if c.Bool("close") {
		out = append(out, "--close")
	}
	if c.Bool("reopen") {
		out = append(out, "--reopen")
	}
	// Forward any further positionals (rare; update.js takes one cardId,
	// but keep the door open for future additions).
	if rest := c.Args().Slice(); len(rest) > 1 {
		out = append(out, rest[1:]...)
	}
	_ = strconv.Itoa // mirror the codegen template's import-keepalive pattern
	return out
}

// runScript invokes `npm --prefix <dir> run <npmScript> -- <scriptArgs>`. When
// scriptArgs is nil, any positional args from the cli.Command are forwarded
// verbatim instead. The double-dash is what tells npm to hand the rest of
// argv to the underlying node script unchanged.
func runScript(ctx context.Context, r *shell.Runner, c *cli.Command, npmScript string, scriptArgs []string) error {
	dir, err := resolveDir(c)
	if err != nil {
		return err
	}
	argv := []string{"--prefix", dir, "run", npmScript}
	if scriptArgs == nil {
		scriptArgs = c.Args().Slice()
	}
	if len(scriptArgs) > 0 {
		argv = append(argv, "--")
		argv = append(argv, scriptArgs...)
	}
	return r.Exec(ctx, "npm", argv...)
}

// resolveDir picks the message-ops checkout in --dir > $COILY_MESSAGE_OPS_DIR
// > ~/projects/coilysiren/message-ops priority. Verifies the directory exists
// and contains a package.json so a typo fails up-front instead of producing a
// confusing npm error.
func resolveDir(c *cli.Command) (string, error) {
	candidate := c.String("dir")
	if candidate == "" {
		candidate = os.Getenv(EnvDir)
	}
	if candidate == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("trello: cannot resolve home dir: %w", err)
		}
		candidate = filepath.Join(home, DefaultRelPath)
	}
	candidate = filepath.Clean(candidate)
	pkg := filepath.Join(candidate, "package.json")
	if _, err := os.Stat(pkg); err != nil {
		return "", fmt.Errorf("trello: %s not found (resolved checkout: %s). pass --dir or set %s",
			pkg, candidate, EnvDir)
	}
	return candidate, nil
}

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coilysiren/coily/pkg/audit"
	"github.com/coilysiren/coily/pkg/exitcode"
	"github.com/coilysiren/coily/pkg/gittree"
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
//
// Repo verbs are gated on a clean+synced working tree. The gate refuses the
// invocation when uncommitted changes, untracked files, a detached HEAD, a
// branch with no upstream, or a behind-upstream state would prevent the
// audit row from being reconstructed from git history alone. The
// --audit-override-dirty flag bypasses the gate but tags the audit row with
// audit_override=true and captures the porcelain status snapshot. Built-in
// verbs are unaffected: their behavior is baked into the homebrew-released
// binary and reproducible from the version trailer in the audit row.
func (r *Runner) buildRepoCommand(cfg *repocfg.Config, rc repocfg.Command) *cli.Command {
	usage := rc.Description
	if usage == "" {
		usage = "Repo command: " + strings.Join(rc.Argv, " ")
	}
	// repoRoot is the parent of .coily/, derived from the discovered config
	// path. cfg.Path looks like <repoRoot>/.coily/coily.yaml.
	repoRoot := filepath.Dir(filepath.Dir(cfg.Path))
	verbName := "repo." + rc.Name
	var dirtyState *gittree.State
	return &cli.Command{
		Name:      rc.Name,
		Usage:     usage,
		ArgsUsage: "[-- extra args]",
		Description: fmt.Sprintf(
			"Per-repo command loaded from %s.\nExpands to: %s\n\nExtra positional args are appended and validated against the same "+
				"shell-metacharacter rules as privileged verbs.\n\nRepo verbs require a clean working tree and a synced upstream branch "+
				"so the audit log can be reconstructed from git history. Use "+
				"--audit-override-dirty for genuine emergencies; the override is "+
				"recorded in the audit row.",
			cfg.Path, strings.Join(rc.Argv, " "),
		),
		Action: verb.Wrap(
			verb.Spec{
				Name: verbName,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					positional := append([]string{}, rc.Argv...)
					positional = append(positional, c.Args().Slice()...)
					return nil, positional
				},
				OnComplete: func(rec *audit.Record) {
					if dirtyState == nil {
						return
					}
					rec.AuditOverride = true
					rec.WorkingTreeStatus = dirtyState.Status
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					override := false
					if root := c.Root(); root != nil {
						override = root.Bool("audit-override-dirty")
					}
					state, err := gittree.CheckClean(repoRoot)
					if err != nil {
						return exitcode.New(exitcode.Internal, "gittree_error", err,
							"coily could not evaluate the repo verb gate; run `git status` "+
								"to confirm the repo is in a sane state, then retry")
					}
					if !state.Clean {
						if !override {
							return exitcode.New(exitcode.PolicyDenied, "repo_verb_dirty",
								errors.New(state.FormatRefusal(verbName)),
								"commit/push the outstanding work and retry, or pass "+
									"--audit-override-dirty for a genuine emergency")
						}
						dirtyState = state
					}
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

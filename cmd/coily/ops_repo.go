package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/coilysiren/coily/pkg/audit"
	"github.com/coilysiren/coily/pkg/exitcode"
	"github.com/coilysiren/coily/pkg/gittree"
	"github.com/coilysiren/coily/pkg/repocfg"
	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

// repoExecResult bundles what loadRepoExecCommand discovered. Either
// Ancestor is set (cwd ancestry has a coily.yaml; the legacy happy path)
// or Children is set (cwd itself has no ancestor config but direct
// children declare commands). Both can be empty when no config is found
// anywhere; callers render that as the "no config" stub. Mutually
// exclusive: an ancestor match wins over child discovery so the operator
// who has consciously cd'd into a configured repo is never overridden by
// a sibling that happens to declare the same name.
type repoExecResult struct {
	Ancestor *repocfg.Config
	Children []*repocfg.Config
}

// childMatch records one (config, command) pair from the child-discovery
// pass. Multiple matches with the same command name are the ambiguous
// case; exactly one match is the auto-execute case.
type childMatch struct {
	cfg *repocfg.Config
	cmd repocfg.Command
}

// loadRepoExecCommand resolves the `exec` verb against cwd. The verb is
// always returned (non-nil) so it stays visible in --help and --tree
// regardless of where coily was invoked from. Three resolution modes:
//
//  1. Ancestor coily.yaml found: subcommands come from that file. Same
//     behavior as the original repo-verb path.
//  2. No ancestor, but direct children declare commands: subcommands
//     aggregate from those children. Names declared by exactly one child
//     auto-execute against that child (cwd-set, commit-scope-bound to it).
//     Names declared by multiple children become error subcommands that
//     list the matches so the operator can disambiguate.
//  3. Neither: a single Action returns a UserError naming the recovery.
//
// Mode 2 is the headline case for running `coily exec daily-social` from
// one directory above coilyco-ai. The communication contract is loud:
// the auto-executing branch prints a stderr line naming the matched
// child before exec, so the operator never silently runs against the
// wrong repo.
func (r *Runner) loadRepoExecCommand() (repoExecResult, *cli.Command) {
	cfg, err := repocfg.LoadDefault()
	if err != nil && !errors.Is(err, repocfg.ErrNoConfig) {
		fmt.Fprintf(os.Stderr, "coily: repo config error: %v\n", err)
		cfg = nil
	}
	if cfg != nil {
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
		return repoExecResult{Ancestor: cfg}, exec
	}

	cwd, _ := os.Getwd()
	children, _ := repocfg.DiscoverChildren(cwd)
	if len(children) > 0 {
		return repoExecResult{Children: children}, r.buildExecFromChildren(children)
	}

	return repoExecResult{}, &cli.Command{
		Name:     "exec",
		Usage:    "Run a named command from .coily/coily.yaml (no config found in cwd)",
		Category: "repo",
		Description: "Run a per-repo command declared in .coily/coily.yaml. " +
			"Discovery walks from cwd up to the filesystem root looking for " +
			".coily/coily.yaml, then falls back to scanning direct children. " +
			"Neither found anything from the current cwd. Create one in the " +
			"target repo, or cd into a repo (or a directory above one) that " +
			"has one, then retry.",
		Action: func(_ context.Context, _ *cli.Command) error {
			where, _ := os.Getwd()
			return exitcode.New(exitcode.UserError, "repo_no_config",
				fmt.Errorf("no .coily/coily.yaml found from cwd (%s) up to filesystem root, "+
					"and no direct child declares one either", where),
				"create .coily/coily.yaml in the target repo (or cd into a repo, "+
					"or one directory above a repo, that has one) and retry")
		},
	}
}

// buildExecFromChildren constructs the `exec` cli.Command from configs
// discovered in direct children of cwd. Each command name unique across
// children becomes an auto-executing subcommand bound to that child;
// names declared by multiple children become error subcommands that list
// the matches so the operator can disambiguate by cd'ing into the target.
func (r *Runner) buildExecFromChildren(children []*repocfg.Config) *cli.Command {
	matches := map[string][]childMatch{}
	for _, cfg := range children {
		for _, c := range cfg.Commands {
			matches[c.Name] = append(matches[c.Name], childMatch{cfg: cfg, cmd: c})
		}
	}
	names := make([]string, 0, len(matches))
	for n := range matches {
		names = append(names, n)
	}
	sort.Strings(names)
	subs := make([]*cli.Command, 0, len(names))
	for _, n := range names {
		ms := matches[n]
		if len(ms) == 1 {
			subs = append(subs, r.buildChildRepoCommand(ms[0].cfg, ms[0].cmd))
			continue
		}
		subs = append(subs, buildAmbiguousChildCommand(n, ms))
	}
	paths := make([]string, 0, len(children))
	for _, c := range children {
		paths = append(paths, filepath.Dir(filepath.Dir(c.Path)))
	}
	return &cli.Command{
		Name:     "exec",
		Usage:    "Run a command from a direct child's .coily/coily.yaml (cwd has no ancestor config)",
		Category: "repo",
		Description: fmt.Sprintf(
			"cwd has no .coily/coily.yaml in its ancestry, so coily searched its "+
				"direct children. %d declare commands. The subcommands below "+
				"aggregate them: names declared by exactly one child auto-execute "+
				"against that child (cwd-set, audit row bound to its repo). Names "+
				"declared by multiple children require explicit disambiguation by "+
				"cd'ing into the target.\n\nChildren scanned:\n  %s",
			len(children), strings.Join(paths, "\n  "),
		),
		Commands: subs,
	}
}

// buildChildRepoCommand wraps a single (cfg, command) pair from the
// child-discovery pass into a cli.Command whose Action runs the declared
// argv inside the child repo (cmd.Dir = repoRoot) and binds the audit
// row to that child's commit-scope, not cwd's. Mirrors buildRepoCommand
// but with two differences: the working directory is forced to the
// child, and the commit-scope is preset rather than read from --commit-
// scope (which would default to cwd's git toplevel and very likely fail
// to resolve when the operator is one level above the repo).
func (r *Runner) buildChildRepoCommand(cfg *repocfg.Config, rc repocfg.Command) *cli.Command {
	repoRoot := filepath.Dir(filepath.Dir(cfg.Path))
	verbName := "repo." + rc.Name
	usage := rc.Description
	if usage == "" {
		usage = "Repo command: " + strings.Join(rc.Argv, " ")
	}
	usage = fmt.Sprintf("%s [from %s]", usage, repoRoot)
	var dirtyState *gittree.State
	return &cli.Command{
		Name:      rc.Name,
		Usage:     usage,
		ArgsUsage: "[-- extra args]",
		Description: fmt.Sprintf(
			"Per-repo command discovered in a direct child of cwd: %s.\nExpands to: %s\n\n"+
				"cwd has no .coily/coily.yaml in its ancestry, so coily searched its direct "+
				"children and matched exactly one declaring %q. The command will run with "+
				"working directory %s and the audit row will bind to that repo's commit-scope.\n\n"+
				"Extra positional args are appended and validated against the same "+
				"shell-metacharacter rules as privileged verbs. Repo verbs require a clean "+
				"working tree and a synced upstream branch.",
			cfg.Path, strings.Join(rc.Argv, " "), rc.Name, repoRoot,
		),
		Action: verb.Wrap(
			verb.Spec{
				Name:                verbName,
				CommitScopeOverride: repoRoot,
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
								"in the matched child repo to confirm it is in a sane state, then retry")
					}
					if !state.Clean {
						if !override {
							return exitcode.New(exitcode.PolicyDenied, "repo_verb_dirty",
								errors.New(state.FormatRefusal(verbName)),
								"commit/push the outstanding work in the matched child repo and retry, "+
									"or pass --audit-override-dirty for a genuine emergency")
						}
						dirtyState = state
					}
					fmt.Fprintf(os.Stderr,
						"coily: exec %s in %s (cwd has no .coily/coily.yaml; "+
							"matched a single direct child)\n",
						rc.Name, repoRoot)
					argv := append([]string{}, rc.Argv[1:]...)
					argv = append(argv, c.Args().Slice()...)
					return r.Runner.ExecIn(ctx, repoRoot, rc.Argv[0], argv...)
				},
			},
			r.Audit,
		),
	}
}

// buildAmbiguousChildCommand returns a cli.Command for a name declared
// by more than one direct child. Running it errors with the list of
// matches so the operator can disambiguate; the matches also appear in
// the command's --help so disambiguation is possible without invoking.
// No verb.Wrap here: the command is informational and never reaches the
// audit layer.
func buildAmbiguousChildCommand(name string, matches []childMatch) *cli.Command {
	paths := make([]string, 0, len(matches))
	for _, m := range matches {
		paths = append(paths, filepath.Dir(filepath.Dir(m.cfg.Path)))
	}
	return &cli.Command{
		Name:  name,
		Usage: fmt.Sprintf("Ambiguous: declared by %d direct children", len(matches)),
		Description: fmt.Sprintf(
			"Multiple direct children of cwd declare %q in their .coily/coily.yaml. "+
				"coily refuses to pick one; cd into the target repo (or a parent "+
				"that has a .coily/coily.yaml) and run again.\n\nMatches:\n  %s",
			name, strings.Join(paths, "\n  "),
		),
		Action: func(_ context.Context, _ *cli.Command) error {
			return exitcode.New(exitcode.UserError, "exec_ambiguous_children",
				fmt.Errorf("%q is declared by %d direct children: %s",
					name, len(matches), strings.Join(paths, ", ")),
				"cd into the target repo (or a parent that has a .coily/coily.yaml) "+
					"and retry")
		},
	}
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
// Same output for --list on the root command; see main.go. The `exec` verb
// is always present in the built-in list (see loadRepoExecCommand); the repo
// section underneath enumerates whatever subcommands the discovered
// .coily/coily.yaml or direct-child configs declare.
func listCommand(builtIns []*cli.Command, exec *cli.Command, result repoExecResult) {
	fmt.Println("Built-in commands:")
	printCmdGroup(builtIns)
	fmt.Println()
	if result.Ancestor != nil {
		fmt.Printf("Repo commands from %s (coily exec <name>):\n", result.Ancestor.Path)
		if exec == nil || len(exec.Commands) == 0 {
			fmt.Println("  (none declared)")
			return
		}
		printCmdGroup(exec.Commands)
		return
	}
	if len(result.Children) > 0 {
		fmt.Printf("Repo commands from %d direct child config(s) (coily exec <name>):\n", len(result.Children))
		for _, c := range result.Children {
			fmt.Printf("  %s:\n", c.Path)
			for _, rc := range c.Commands {
				if rc.Description != "" {
					fmt.Printf("    %s  %s\n", rc.Name, rc.Description)
				} else {
					fmt.Printf("    %s\n", rc.Name)
				}
			}
		}
		return
	}
	fmt.Println("Repo commands (coily exec <name>):")
	fmt.Println("  (no .coily/coily.yaml found from cwd or its direct children; coily exec is wired but has no subcommands)")
}

// treeCommand renders every coily command and subcommand recursively.
// Same surfaces as --list, but walks the full subcommand tree instead of
// stopping at the top level.
func treeCommand(builtIns []*cli.Command, exec *cli.Command, result repoExecResult) {
	fmt.Println("Built-in commands:")
	printCmdTree(builtIns, "  ")
	fmt.Println()
	if result.Ancestor != nil {
		fmt.Printf("Repo commands from %s (coily exec <name>):\n", result.Ancestor.Path)
		if exec == nil || len(exec.Commands) == 0 {
			fmt.Println("  (none declared)")
			return
		}
		printCmdTree(exec.Commands, "  ")
		return
	}
	if len(result.Children) > 0 {
		fmt.Printf("Repo commands from %d direct child config(s) (coily exec <name>):\n", len(result.Children))
		if exec != nil && len(exec.Commands) > 0 {
			printCmdTree(exec.Commands, "  ")
		}
		return
	}
	fmt.Println("Repo commands (coily exec <name>):")
	fmt.Println("  (no .coily/coily.yaml found from cwd or its direct children; coily exec is wired but has no subcommands)")
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

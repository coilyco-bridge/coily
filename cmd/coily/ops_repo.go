package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/audit"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/egress"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/exitcode"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/gittree"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/repocfg"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/shell"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// repoExecResult bundles what loadRepoExecCommand discovered. The
// discovery surface is a single unified pool: every coily.yaml reachable
// from cwd via the ancestor walk plus every coily.yaml in a direct child.
// `Configs` is that pool, sorted by Path. Empty means no config was
// found anywhere reachable. There is no ancestor-vs-children branching
// at this layer: the verb groups commands by name and disambiguates one
// per-name (single declarant auto-runs, multiple declarants prompt).
type repoExecResult struct {
	Configs []*repocfg.Config
}

// childMatch records one (config, command) pair from the unified pool.
// Multiple matches with the same command name trigger the interactive
// pick prompt; exactly one match is the auto-execute case.
type childMatch struct {
	cfg *repocfg.Config
	cmd repocfg.Command
}

// loadRepoExecCommand resolves the `exec` verb against cwd. The verb is
// always returned (non-nil) so it stays visible in --help and --tree
// regardless of where coily was invoked from.
//
// Discovery is a single pool: every coily.yaml reachable from cwd via
// repocfg.DiscoverAll (ancestor walk + direct children of cwd). Commands
// are aggregated across that pool by name (a simple alphanumeric match).
//
//   - 0 declarants for a name: name does not exist as a subcommand.
//   - 1 declarant: subcommand auto-runs against that repo, cwd set to the
//     repo root.
//   - 2+ declarants: subcommand prompts on stderr for a numeric pick,
//     reads stdin, then runs the picked repo's command.
//
// This replaces the older "ancestor wins, else fall back to direct
// children" branching, which produced "where did my config come from"
// non-determinism when cwd shifted one level up or down.
func (r *Runner) loadRepoExecCommand() (repoExecResult, *cli.Command) {
	cwd, _ := os.Getwd()
	configs, err := repocfg.DiscoverAll(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "coily: repo config discovery error: %v\n", err)
	}
	if len(configs) == 0 {
		return repoExecResult{}, &cli.Command{
			Name:     "exec",
			Usage:    "Run a named command from .coily/coily.yaml (no config reachable from cwd)",
			Category: "repo",
			Description: "Run a per-repo command declared in .coily/coily.yaml. " +
				"Discovery is a single pool: every coily.yaml from cwd's ancestor " +
				"walk plus every coily.yaml in a direct child of cwd. Nothing " +
				"was reachable. Create one in the target repo, or cd into a " +
				"directory that has one (or sits above one), then retry.",
			Action: func(_ context.Context, _ *cli.Command) error {
				where, _ := os.Getwd()
				return exitcode.New(exitcode.UserError, "repo_no_config",
					fmt.Errorf("no .coily/coily.yaml reachable from cwd (%s) "+
						"via ancestor walk or direct-child scan", where),
					"create .coily/coily.yaml in the target repo (or cd into a repo, "+
						"or one directory above a repo, that has one) and retry").
					WithReason("repo verbs require a .coily/coily.yaml so the verb surface is declarative and reviewable")
			},
		}
	}
	return repoExecResult{Configs: configs}, r.buildExecFromConfigs(configs)
}

// buildExecFromConfigs constructs the `exec` cli.Command from the unified
// candidate pool. Each command name unique across the pool becomes an
// auto-executing subcommand bound to that repo; names declared by more
// than one repo become prompting subcommands that ask the operator (or
// agent on stdin) to pick before exec.
func (r *Runner) buildExecFromConfigs(configs []*repocfg.Config) *cli.Command {
	matches := map[string][]childMatch{}
	for _, cfg := range configs {
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
		subs = append(subs, r.buildPromptingChildCommand(n, ms))
	}
	paths := make([]string, 0, len(configs))
	for _, c := range configs {
		paths = append(paths, filepath.Dir(filepath.Dir(c.Path)))
	}
	return &cli.Command{
		Name:     "exec",
		Usage:    "Run a command from a .coily/coily.yaml reachable from cwd",
		Category: "repo",
		Description: fmt.Sprintf(
			"Discovery pool: every coily.yaml from cwd's ancestor walk plus "+
				"every coily.yaml in a direct child of cwd. %d configs in scope. "+
				"Subcommand names are the union of declared commands across the "+
				"pool: declared by exactly one repo means auto-execute against "+
				"that repo (cwd-set); "+
				"declared by multiple repos means prompt on stderr for a numeric "+
				"pick before exec.\n\nConfigs in scope:\n  %s",
			len(configs), strings.Join(paths, "\n  "),
		),
		Commands: subs,
	}
}

// buildChildRepoCommand wraps a single (cfg, command) pair into a
// cli.Command whose Action runs the declared argv inside the matched
// repo (cmd.Dir = repoRoot). Used for command names with exactly one declarant in the
// unified discovery pool. Repo verbs require a synced upstream branch and
// that the coily.yaml that declares them be committed, so the audit row's
// argv can be reconstructed from git history; --audit-override-dirty
// bypasses with audit_override=true. Working-tree dirt outside coily.yaml
// is recorded in the audit row but does not refuse.
func (r *Runner) buildChildRepoCommand(cfg *repocfg.Config, rc repocfg.Command) *cli.Command {
	repoRoot := filepath.Dir(filepath.Dir(cfg.Path))
	verbName := "repo." + rc.Name
	usage := rc.Description
	if usage == "" {
		usage = "Repo command: " + strings.Join(rc.Argv, " ")
	}
	usage = fmt.Sprintf("%s [from %s]", usage, repoRoot)
	var (
		capturedState *gittree.State
		overrideUsed  bool
		egressRows    []audit.EgressRow
	)
	return &cli.Command{
		Name:      rc.Name,
		Usage:     usage,
		ArgsUsage: "[-- extra args]",
		Description: fmt.Sprintf(
			"Per-repo command discovered in %s.\nExpands to: %s\n\n"+
				"Exactly one repo in the discovery pool declares %q, so coily runs "+
				"it without prompting. Working directory is set to %s.\n\nExtra positional args "+
				"are appended and validated against the same shell-metacharacter "+
				"rules as privileged verbs unless the declaring command opts in "+
				"with allow_metacharacters: true (audit row stamps policy_skipped "+
				"so forensics still see the relaxed policy). Repo verbs require "+
				"the declaring coily.yaml to be committed and a synced upstream "+
				"branch; unrelated working-tree dirt is captured in the audit row "+
				"but does not refuse.",
			cfg.Path, strings.Join(rc.Argv, " "), rc.Name, repoRoot,
		),
		Action: r.WrapVerb(
			verb.Spec{
				Name:       verbName,
				SkipPolicy: rc.AllowMetacharacters,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					positional := append([]string{}, rc.Argv...)
					positional = append(positional, c.Args().Slice()...)
					return nil, positional
				},
				OnComplete: func(rec *audit.Record) {
					if rc.AllowMetacharacters {
						rec.PolicySkipped = true
					}
					if len(egressRows) > 0 {
						rec.Egress = egressRows
					}
					if capturedState == nil {
						return
					}
					rec.WorkingTreeStatus = capturedState.Status
					rec.AuditOverride = overrideUsed
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					state, used, err := runRepoGate(c, repoRoot, cfg.Path, verbName)
					if err != nil {
						return err
					}
					if state.Status != "" {
						capturedState = state
						overrideUsed = used
					}
					fmt.Fprintf(os.Stderr,
						"coily: exec %s in %s (only repo in pool declaring this name)\n",
						rc.Name, repoRoot)
					argv := append([]string{}, rc.Argv[1:]...)
					argv = append(argv, c.Args().Slice()...)
					rows, execErr := execRepoCommand(ctx, r.Runner, rc.Egress, repoRoot, rc.Argv[0], argv...)
					egressRows = rows
					return execErr
				},
			},
			r.Audit,
		),
	}
}

// buildPromptingChildCommand returns a cli.Command for a name declared
// by more than one repo in the unified discovery pool. On invocation it
// prints the matches on stderr, reads a numeric pick from stdin, then
// runs the chosen repo's command with cwd set to that repo. The pick
// happens inside the wrapped Action so verb.Wrap's argv validation and
// audit logging cover the whole flow.
func (r *Runner) buildPromptingChildCommand(name string, matches []childMatch) *cli.Command {
	verbName := "repo." + name
	repoPaths := make([]string, 0, len(matches))
	allAllowMetacharacters := true
	for _, m := range matches {
		repoPaths = append(repoPaths, filepath.Dir(filepath.Dir(m.cfg.Path)))
		if !m.cmd.AllowMetacharacters {
			allAllowMetacharacters = false
		}
	}
	var (
		chosen        *childMatch
		capturedState *gittree.State
		overrideUsed  bool
		egressRows    []audit.EgressRow
	)
	return &cli.Command{
		Name:      name,
		Usage:     fmt.Sprintf("Declared by %d repos; prompts for a pick on stderr/stdin", len(matches)),
		ArgsUsage: "[-- extra args]",
		Description: fmt.Sprintf(
			"%d repos in the discovery pool declare %q. Invoking this "+
				"subcommand prints the matches on stderr and reads a numeric "+
				"choice from stdin. The chosen repo's command runs with cwd "+
				"set to that repo.\n\nMatches:\n  %s",
			len(matches), name, strings.Join(repoPaths, "\n  ")),
		Action: r.WrapVerb(
			verb.Spec{
				Name:       verbName,
				SkipPolicy: allAllowMetacharacters,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return nil, c.Args().Slice()
				},
				OnComplete: func(rec *audit.Record) {
					if chosen != nil {
						if chosen.cmd.AllowMetacharacters {
							rec.PolicySkipped = true
						}
					}
					if len(egressRows) > 0 {
						rec.Egress = egressRows
					}
					if capturedState != nil {
						rec.WorkingTreeStatus = capturedState.Status
						rec.AuditOverride = overrideUsed
					}
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					pick, err := promptChildChoice(name, matches, r.Runner.Stdin, r.Runner.Stderr)
					if err != nil {
						return err
					}
					chosen = &pick
					repoRoot := filepath.Dir(filepath.Dir(pick.cfg.Path))
					state, used, gerr := runRepoGate(c, repoRoot, pick.cfg.Path, verbName)
					if gerr != nil {
						return gerr
					}
					if state.Status != "" {
						capturedState = state
						overrideUsed = used
					}
					fmt.Fprintf(os.Stderr,
						"coily: exec %s in %s (picked from %d declarants)\n",
						pick.cmd.Name, repoRoot, len(matches))
					argv := append([]string{}, pick.cmd.Argv[1:]...)
					argv = append(argv, c.Args().Slice()...)
					rows, execErr := execRepoCommand(ctx, r.Runner, pick.cmd.Egress, repoRoot, pick.cmd.Argv[0], argv...)
					egressRows = rows
					return execErr
				},
			},
			r.Audit,
		),
	}
}

// execRepoCommand runs a repo-declared command, optionally through a
// per-invocation HTTP CONNECT proxy so the audit row's egress array
// matches what passthrough verbs (ops.aws, ops.gh, pkg.uv, ...) already
// emit. When egressOn is false this is a thin wrapper around
// base.ExecIn. When true, a proxy starts in observe mode (every CONNECT
// forwarded and logged; no allowlist), a shadow Runner injects
// HTTPS_PROXY/HTTP_PROXY for this invocation, and the collected rows
// return to the caller for the OnComplete hook to stamp onto the audit
// record. Mirrors passthrough.withEgressAction for the cli-guard side
// of the asymmetry; coilysiren/coily#281.
func execRepoCommand(ctx context.Context, base *shell.Runner, egressOn bool, dir, bin string, argv ...string) ([]audit.EgressRow, error) {
	if !egressOn {
		return nil, base.ExecIn(ctx, dir, bin, argv...)
	}
	p := egress.New(nil, egress.ModeObserve)
	proxyURL, err := p.Start(ctx)
	if err != nil {
		return nil, fmt.Errorf("egress: start proxy: %w", err)
	}
	shadow := *base
	shadow.Env = append([]string(nil), base.Env...)
	shadow.Env = append(shadow.Env,
		"HTTPS_PROXY="+proxyURL,
		"HTTP_PROXY="+proxyURL,
		"https_proxy="+proxyURL,
		"http_proxy="+proxyURL,
	)
	execErr := shadow.ExecIn(ctx, dir, bin, argv...)
	return p.Stop(), execErr
}

// runRepoGate runs the clean-tree gate for a repo verb. Returns the
// gittree state, an overrideUsed boolean (true when --audit-override-dirty
// was actually needed to bypass the gate, false when the gate passed on
// its own), or an error on refusal / gittree failure.
//
// The gate refuses when the repo's audit row could not be reconstructed
// from git history: an upstream/HEAD problem, or the declaring coily.yaml
// itself is among the dirty paths (so the verb argv at HEAD is not what
// just ran). Working-tree dirt outside coily.yaml does not refuse - the
// committed coily.yaml plus HEAD still reconstruct the verb invocation -
// but the porcelain status is returned so the caller can stamp it on the
// audit row for forensics. See coilysiren/coily#211.
func runRepoGate(c *cli.Command, repoRoot, cfgPath, verbName string) (*gittree.State, bool, error) {
	override := false
	if root := c.Root(); root != nil {
		override = root.Bool("audit-override-dirty")
	}
	state, err := gittree.CheckClean(repoRoot)
	if err != nil {
		return nil, false, exitcode.New(exitcode.Internal, "gittree_error", err,
			"coily could not evaluate the repo verb gate; run `git status` "+
				"in the matched repo to confirm it is in a sane state, then retry")
	}
	if state.Clean {
		return state, false, nil
	}
	if dirtIsOutsideCoilyConfig(state, repoRoot, cfgPath) {
		return state, false, nil
	}
	if override {
		return state, true, nil
	}
	return nil, false, exitcode.New(exitcode.PolicyDenied, "repo_verb_dirty",
		errors.New(state.FormatRefusal(verbName)),
		"commit/push the outstanding coily.yaml change in the matched repo "+
			"and retry, or pass --audit-override-dirty for a genuine emergency").
		WithReason("audit rows must bind to a committed coily.yaml so the verb argv can be reconstructed from git history")
}

// dirtIsOutsideCoilyConfig reports whether state is a "working tree is
// dirty" refusal whose dirty path set does not touch the declaring
// coily.yaml. Returns false for upstream/HEAD refusals (those still gate
// even if coily.yaml is committed) and for any dirty set that includes
// coily.yaml itself.
func dirtIsOutsideCoilyConfig(state *gittree.State, repoRoot, cfgPath string) bool {
	if state.Reason != "working tree is dirty" {
		return false
	}
	rel, err := filepath.Rel(repoRoot, cfgPath)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	for _, p := range state.DirtyPaths {
		if filepath.ToSlash(p) == rel {
			return false
		}
	}
	return true
}

// promptChildChoice presents the matches on out (stderr) as a numbered
// menu and reads a single line from in (stdin) containing the 1-indexed
// choice. Used by the prompting subcommand when a command name is
// declared by more than one repo. Returns a UserError when input is
// missing or out of range so the agent gets a clean retry hint instead
// of a panic.
func promptChildChoice(name string, matches []childMatch, in io.Reader, out io.Writer) (childMatch, error) {
	if in == nil {
		return childMatch{}, exitcode.New(exitcode.UserError, "exec_prompt_no_stdin",
			fmt.Errorf("%q is declared by %d repos and no stdin is attached to read a pick from",
				name, len(matches)),
			"re-invoke with stdin attached and pipe the 1-indexed choice number, "+
				"or cd into the target repo to disambiguate without a prompt").
			WithReason("the choice prompt for ambiguous repo names needs an interactive stdin; pipe the 1-indexed pick instead")
	}
	_, _ = fmt.Fprintf(out, "coily: %q is declared by %d repos. pick one:\n", name, len(matches))
	for i, m := range matches {
		repoRoot := filepath.Dir(filepath.Dir(m.cfg.Path))
		_, _ = fmt.Fprintf(out, "  %d) %s (%s)\n", i+1, filepath.Base(repoRoot), repoRoot)
	}
	_, _ = fmt.Fprintf(out, "choice [1-%d]: ", len(matches))
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && (line == "" || !errors.Is(err, io.EOF)) {
		return childMatch{}, exitcode.New(exitcode.UserError, "exec_prompt_read",
			fmt.Errorf("read pick for %q: %w", name, err),
			"re-invoke and pipe the 1-indexed choice number on stdin").
			WithReason("the choice prompt for ambiguous repo names needs an interactive stdin; pipe the 1-indexed pick instead")
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return childMatch{}, exitcode.New(exitcode.UserError, "exec_prompt_empty",
			fmt.Errorf("no choice provided for %q (got %d declarants)", name, len(matches)),
			"re-invoke and pipe the 1-indexed choice number on stdin").
			WithReason("the choice prompt requires a 1-indexed integer; an empty pick is ambiguous")
	}
	idx, err := strconv.Atoi(line)
	if err != nil || idx < 1 || idx > len(matches) {
		return childMatch{}, exitcode.New(exitcode.UserError, "exec_prompt_invalid",
			fmt.Errorf("pick %q is not in [1-%d]", line, len(matches)),
			fmt.Sprintf("re-invoke and pipe an integer in [1-%d] on stdin", len(matches))).
			WithReason("the choice prompt requires a 1-indexed integer in range")
	}
	return matches[idx-1], nil
}

// listCommand renders the built-in and repo command inventory in one shot.
// Same output for --list on the root command; see main.go. The `exec` verb
// is always present in the built-in list (see loadRepoExecCommand); the repo
// section underneath enumerates whatever subcommands the unified discovery
// pool produces.
func listCommand(builtIns []*cli.Command, _ *cli.Command, result repoExecResult) {
	fmt.Println("Built-in commands:")
	printCmdGroup(builtIns)
	fmt.Println()
	if len(result.Configs) > 0 {
		fmt.Printf("Repo commands from %d config(s) reachable from cwd (coily exec <name>):\n", len(result.Configs))
		for _, c := range result.Configs {
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
	fmt.Println("  (no .coily/coily.yaml reachable from cwd; coily exec is wired but has no subcommands)")
}

// treeCommand renders every coily command and subcommand recursively.
// Same surfaces as --list, but walks the full subcommand tree instead of
// stopping at the top level.
func treeCommand(builtIns []*cli.Command, exec *cli.Command, result repoExecResult) {
	fmt.Println("Built-in commands:")
	printCmdTree(builtIns, "  ")
	fmt.Println()
	if len(result.Configs) > 0 {
		fmt.Printf("Repo commands from %d config(s) reachable from cwd (coily exec <name>):\n", len(result.Configs))
		if exec != nil && len(exec.Commands) > 0 {
			printCmdTree(exec.Commands, "  ")
		}
		return
	}
	fmt.Println("Repo commands (coily exec <name>):")
	fmt.Println("  (no .coily/coily.yaml reachable from cwd; coily exec is wired but has no subcommands)")
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

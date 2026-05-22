package main

import (
	"github.com/coilysiren/cli-guard/passthrough"
	"github.com/urfave/cli/v3"
)

// gitPassthroughVerbs is the heavy-rotation git command set fronted by
// `coily git <verb>`. Each becomes a thin passthrough to `git <verb> ...`
// with an audit row, mounted alongside the audit-trailer subcommands.
//
// `coily git` cannot be a single whole-binary passthrough (the way
// `coily docker` is) because the name is already the audit-trailer group.
// Per-verb subcommands sidestep that and keep the audit verb granular.
var gitPassthroughVerbs = []struct{ name, usage string }{
	{"status", "git status - show the working tree state."},
	{"log", "git log - show commit history."},
	{"diff", "git diff - show changes."},
	{"show", "git show - show a commit or object."},
	{"add", "git add - stage changes."},
	{"commit", "git commit - record staged changes."},
	{"fetch", "git fetch - download objects and refs from a remote."},
	{"pull", "git pull - fetch and integrate."},
	{"push", "git push - update remote refs."},
	{"branch", "git branch - list or manage branches."},
	{"checkout", "git checkout - switch branches or restore files."},
	{"stash", "git stash - shelve working-tree changes."},
	{"restore", "git restore - restore working-tree files."},
}

// gitVerbRewriter prepends the git subcommand to the passthrough argv, and
// hoists a leading `-C <path>` ahead of the verb so `coily git <verb> -C
// <path>` resolves to `git -C <path> <verb> ...` (git requires -C before
// the subcommand). The -C form lets a `coily ssh` session operate on a repo
// other than the ssh target's fixed working directory.
func gitVerbRewriter(verb string) func([]string) []string {
	return func(argv []string) []string {
		if len(argv) >= 2 && argv[0] == "-C" {
			out := []string{"-C", argv[1], verb}
			return append(out, argv[2:]...)
		}
		return append([]string{verb}, argv...)
	}
}

// gitPassthroughCommand builds one `coily git <verb>` passthrough command.
func (r *Runner) gitPassthroughCommand(verb, usage string) *cli.Command {
	cmd := passthrough.Command("git", r.Runner, r.Audit,
		passthrough.WithVerbName("git."+verb),
		passthrough.WithArgvRewriter(gitVerbRewriter(verb)),
	)
	cmd.Name = verb
	cmd.Usage = usage
	return cmd
}

// gitPassthroughCommands builds the full passthrough set for mounting under
// the `coily git` group.
func (r *Runner) gitPassthroughCommands() []*cli.Command {
	out := make([]*cli.Command, 0, len(gitPassthroughVerbs))
	for _, v := range gitPassthroughVerbs {
		out = append(out, r.gitPassthroughCommand(v.name, v.usage))
	}
	return out
}

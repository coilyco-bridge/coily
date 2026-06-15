package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/cli/hook"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/cli/verb"
	"github.com/urfave/cli/v3"
)

// hookCommand groups Claude Code hook entry points. coily exposes
// pre-tool-use (recovery: route a denied bare binary through its wrapper)
// and session-start (awareness: inject the capability index). The shape is
// extensible to other hook events (PostToolUse, UserPromptSubmit) when
// there is a reason to gate on them.
//
// coilysiren/coily#248 + cli-guard#74: the per-repo lockdown-deny.sh
// coily renders exec's `coily hook pre-tool-use`, which calls the
// shared cli-guard/hook engine with coily's own integrity rules and
// routing-hint Route table. ward runs its own equivalent hook
// in its own repos. The two peers do not reference each other in
// source.
func (r *Runner) hookCommand() *cli.Command {
	return &cli.Command{
		Name:  "hook",
		Usage: "Claude Code hook entry points.",
		Commands: []*cli.Command{
			r.hookPreToolUseCommand(),
			r.hookSessionStartCommand(),
		},
	}
}

func (r *Runner) hookPreToolUseCommand() *cli.Command {
	return &cli.Command{
		Name:  "pre-tool-use",
		Usage: "PreToolUse hook for the Bash tool. Routes bare-binary invocations through coily wrappers with a recovery hint; rejects coily-binary invocations resolving outside the canonical install paths.",
		Action: r.WrapVerb(verb.Spec{
			Name:       "hook.pre-tool-use",
			SkipPolicy: true,
			Action: func(_ context.Context, _ *cli.Command) error {
				return runCoilyPreToolUse(os.Stdin, os.Stderr, exec.LookPath)
			},
		}, r.Audit),
	}
}

// coilyHookSource is the prefix string passed to the cli-guard hook
// engine. Appears in block messages so the user knows which guard
// process emitted the block ("coily hook: blocked bare `gh` ...").
const coilyHookSource = "coily"

// coilyBinaryAllowedPaths is the canonical install-path allow-list
// for the coily binary. A bare `coily ...` invocation resolving
// outside these paths is a PATH-hijack candidate and gets blocked
// with a sharp diagnostic, ahead of any route lookup.
var coilyBinaryAllowedPaths = []string{
	"/opt/homebrew/bin/coily",
	"/usr/local/bin/coily",
	"/home/linuxbrew/.linuxbrew/bin/coily",
}

// runCoilyPreToolUse is the testable core. Reads a PreToolUse payload
// from in, emits any block message to errOut, returns nil on pass-
// through and a cli.Exit(2) on block (PreToolUse contract).
func runCoilyPreToolUse(in *os.File, errOut *os.File, lookup hook.LookPath) error {
	payload := hook.ReadPayload(in)
	decision := hook.PreToolUse(payload, coilyHookSource, coilyIntegrityRules(), coilyHookRoutes(), lookup)
	if !decision.Block {
		return nil
	}
	_, _ = fmt.Fprintln(errOut, decision.Message)
	return cli.Exit("", 2)
}

// coilyIntegrityRules returns the integrity-rule set the hook engine
// applies before route lookup. Coily owns its own binary path
// expectations; ward owns its own (if it cares).
func coilyIntegrityRules() []hook.IntegrityRule {
	return []hook.IntegrityRule{
		{Binary: "coily", AllowedPaths: coilyBinaryAllowedPaths},
	}
}

// coilyHookRoutes returns the routing-hint Route table coily wants
// the hook engine to surface. Built from wrapperRecovery (the same
// table coily lockdown uses to compose its allow rules), so the two
// surfaces stay in sync. The hint format mirrors the prior agent-
// guard-side strings to keep operator muscle memory.
//
// Special case: the `gh` route appends a GraphQL-trap reminder when
// the segment is one of the gh subcommands that hit the GraphQL API
// by default. coily-specific knowledge; lives in this route's Extra.
func coilyHookRoutes() []hook.Route {
	tokens := make([]string, 0, len(wrapperRecovery))
	for tok := range wrapperRecovery {
		tokens = append(tokens, tok)
	}
	sort.Strings(tokens)
	routes := make([]hook.Route, 0, len(tokens))
	for _, tok := range tokens {
		target := wrapperRecovery[tok]
		hint := fmt.Sprintf("use `%s ...` (audited wrapper).", target)
		r := hook.Route{Token: tok, Hint: hint}
		if tok == "gh" {
			r.Extra = ghGraphQLTrapSuffix
		}
		routes = append(routes, r)
	}
	return routes
}

// ghGraphQLTrapSuffix returns the GraphQL-trap reminder when seg is a
// gh invocation whose subcommand routes through GraphQL by default.
// The list is the frequent offenders; signaling, not policing.
func ghGraphQLTrapSuffix(seg string) string {
	rest := strings.TrimPrefix(seg, "gh ")
	if rest == seg {
		return ""
	}
	rest = strings.TrimLeft(rest, " ")
	parts := strings.SplitN(rest, " ", 3)
	if len(parts) < 2 {
		return ""
	}
	if !isGhGraphQLSub(parts[0], parts[1]) {
		return ""
	}
	return " (and note: `gh issue view` / `gh pr view` / `gh repo view` / `gh search` use the GraphQL API by default - prefer `coily ops gh api /repos/OWNER/REPO/...` to avoid the GraphQL rate-limit budget)"
}

// isGhGraphQLSub returns true when the (sub, action) pair hits GitHub's
// GraphQL API by default. Frequent offenders only; not exhaustive.
func isGhGraphQLSub(sub, action string) bool {
	switch sub {
	case "issue":
		return action == "view" || action == "list" || action == "status"
	case "pr":
		return action == "view" || action == "list" || action == "status" || action == "checks"
	case "repo":
		return action == "view" || action == "list"
	case "search", "project":
		return true
	}
	return false
}

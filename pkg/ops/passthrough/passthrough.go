// Package passthrough is the thin pass-through used to wrap any sub-CLI
// (aws, gh, kubectl, docker, tailscale, plus every package manager) as a
// single `coily <bin> ...` verb. One Command(name, runner, audit) builder
// per binary, no subcommand modeling.
//
// What the wrapper buys:
//
//   - Audit log. Every invocation lands in ~/.coily/audit/*.jsonl with
//     timestamp + argv + cwd + exit code.
//   - Argv validation. Refusal of shell metacharacters in argv blocks the
//     dumbest injection paths. Doesn't stop a malicious tool; does stop
//     the agent from being tricked into "kubectl get pod 'foo; curl bad'".
//   - Symmetric deny coverage. The lockdown deny list blocks raw `aws` /
//     `kubectl` / `pnpm` / `cargo` / etc., so the agent can only reach
//     them via coily.
//
// What it doesn't buy: protection from a malicious binary or package
// installed on the host. The wrapper is an audit + gate, not a sandbox;
// sandboxing belongs in agentcontainers, not here.
//
// Why the per-CLI subcommand trees were ripped (issue #27): the
// generator-driven trees that previously wrapped aws/gh/kubectl earned
// their cost only via per-leaf readonly-vs-mutator gating, which is
// already redundant with the lockdown deny list (Bash(kubectl get:*) allow
// vs Bash(kubectl apply:*) deny fires before coily ever runs). The
// remaining benefits (help mirroring, tab completion below `coily aws`)
// were convenience without security value, and the cost was ~80k lines of
// generated Go plus a weekly refresh cadence per upstream. SkipFlagParsing
// + verb.Wrap is the same security boundary in 60 lines.
package passthrough

import (
	"context"
	"fmt"

	"github.com/coilysiren/coily/pkg/audit"
	"github.com/coilysiren/coily/pkg/shell"
	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

// Command returns the *cli.Command for `coily <bin>`. Every argument
// after the binary name is forwarded verbatim through the pass-through
// pipeline (argv validation -> audit -> exec). SkipFlagParsing keeps
// urfave/cli from interpreting flags meant for the underlying tool.
func Command(bin string, r *shell.Runner, w *audit.Writer) *cli.Command {
	return &cli.Command{
		Name:            bin,
		Usage:           fmt.Sprintf("Pass-through to %s with argv validation + audit log.", bin),
		SkipFlagParsing: true,
		Action: verb.Wrap(
			verb.Spec{
				Name: bin,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return nil, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return r.Exec(ctx, bin, c.Args().Slice()...)
				},
			},
			w,
		),
	}
}

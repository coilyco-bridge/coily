// Package pkgmgr is the thin pass-through used to wrap each language /
// system package manager (npm, pnpm, yarn, bun, uv, pip, pipx, poetry,
// cargo, gem, bundle, brew). One small Command(name, ...) builder per
// binary; no subcommand modeling. The aws/gh/kubectl pattern of
// generating a full subcommand tree pays off when there is a real
// readonly-vs-mutator split worth gating per leaf. Every package-manager
// invocation that matters is a mutation, so the classification gives
// nothing back, and the help-mirroring + tab-completion cost is not
// worth a hundred-leaf generated file per binary.
//
// What the wrapper buys:
//
//   - Audit log. Every "pnpm add" / "uv pip install" / "cargo add" lands
//     in ~/.coily/audit/*.jsonl with timestamp + argv + cwd + exit code.
//   - Argv validation. Refusal of shell metacharacters in argv blocks the
//     dumbest injection paths. Doesn't stop a malicious package; does
//     stop the agent from being tricked into "pnpm add 'foo; curl bad'".
//   - Symmetric deny coverage. The lockdown deny list now blocks raw
//     `pnpm` / `uv` / `cargo` / `brew` / etc. so the agent can only
//     reach them via coily, the same shape as `aws` / `gh` / `kubectl`.
//
// What it doesn't buy: protection from a malicious package that makes it
// past `coily <pkgmgr> add evil`. Postinstall scripts run with full
// process privileges. The wrapper is an audit + gate, not a sandbox;
// sandboxing belongs in agentcontainers, not here.
package pkgmgr

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

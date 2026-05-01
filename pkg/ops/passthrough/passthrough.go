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
	"github.com/coilysiren/coily/pkg/egress"
	"github.com/coilysiren/coily/pkg/shell"
	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

// Option configures a pass-through Command. Use the With* helpers below
// rather than setting fields directly.
type Option func(*config)

type config struct {
	skipPolicy bool
	egressOn   bool
	egressList []string
	egressMode egress.Mode
}

// WithSkipPolicy disables the shell-metacharacter check for this binary.
// Use only for tools whose argv goes through execve straight to the
// underlying CLI without ever being handed to a shell. Safe set: gh, aws,
// tailscale, package managers (npm/pnpm/yarn/cargo/uv/pip/...). Unsafe
// set: kubectl, docker, ssh - these have exec-into-shell paths where
// argv content can reach a remote bash -c. The audit log and lockdown
// deny list still cover the safe set; the metacharacter check was only
// ever defending the shell-bound path, and rejecting '>' / backticks /
// '$' in argv breaks legitimate markdown bodies in (e.g.)
// `gh issue create --body`.
func WithSkipPolicy() Option {
	return func(c *config) { c.skipPolicy = true }
}

// WithEgress wires the per-invocation HTTP CONNECT proxy. The child runs
// with HTTPS_PROXY/HTTP_PROXY pointed at 127.0.0.1:NNNNN for the lifetime
// of this invocation; the proxy collects one row per host contacted and
// attaches them to the audit record. In ModeEnforce, hosts not on the
// allowlist are denied with 403 (the underlying tool sees a connection
// failure). In ModeObserve, every host is forwarded and logged; the
// allowlist is ignored. See pkg/egress and issue #35.
func WithEgress(allowlist []string, mode egress.Mode) Option {
	return func(c *config) {
		c.egressOn = true
		c.egressList = allowlist
		c.egressMode = mode
	}
}

// Command returns the *cli.Command for `coily <bin>`. Every argument
// after the binary name is forwarded verbatim through the pass-through
// pipeline (argv validation -> audit -> exec). SkipFlagParsing keeps
// urfave/cli from interpreting flags meant for the underlying tool.
func Command(bin string, r *shell.Runner, w *audit.Writer, opts ...Option) *cli.Command {
	cfg := config{}
	for _, opt := range opts {
		opt(&cfg)
	}
	spec := verb.Spec{
		Name: bin,
		ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
			return nil, c.Args().Slice()
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			return r.Exec(ctx, bin, c.Args().Slice()...)
		},
		SkipPolicy: cfg.skipPolicy,
	}
	if cfg.egressOn {
		spec.Action, spec.OnComplete = withEgressAction(bin, r, cfg.egressList, cfg.egressMode)
	}
	return &cli.Command{
		Name:            bin,
		Usage:           fmt.Sprintf("Pass-through to %s with argv validation + audit log.", bin),
		SkipFlagParsing: true,
		Action:          verb.Wrap(spec, w),
	}
}

// withEgressAction wraps the standard exec action to start a per-invocation
// proxy, inject HTTPS_PROXY/HTTP_PROXY into a per-call shadow Runner so
// concurrent commands stay independent, and surface the collected rows
// through OnComplete.
func withEgressAction(bin string, base *shell.Runner, allowlist []string, mode egress.Mode) (cli.ActionFunc, func(*audit.Record)) {
	var rows []audit.EgressRow
	action := func(ctx context.Context, c *cli.Command) error {
		p := egress.New(allowlist, mode)
		proxyURL, err := p.Start(ctx)
		if err != nil {
			return fmt.Errorf("egress: start proxy: %w", err)
		}
		// Shadow the Runner so we don't mutate the shared base state.
		shadow := *base
		shadow.Env = append([]string(nil), base.Env...)
		shadow.Env = append(shadow.Env,
			"HTTPS_PROXY="+proxyURL,
			"HTTP_PROXY="+proxyURL,
			"https_proxy="+proxyURL,
			"http_proxy="+proxyURL,
		)
		execErr := shadow.Exec(ctx, bin, c.Args().Slice()...)
		rows = p.Stop()
		return execErr
	}
	hook := func(rec *audit.Record) {
		if len(rows) > 0 {
			rec.Egress = rows
		}
	}
	return action, hook
}

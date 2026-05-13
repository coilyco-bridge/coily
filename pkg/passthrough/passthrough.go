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
// remaining benefits (help mirroring, tab completion below `coily ops aws`)
// were convenience without security value, and the cost was ~80k lines of
// generated Go plus a weekly refresh cadence per upstream. SkipFlagParsing
// + verb.Wrap is the same security boundary in 60 lines.
package passthrough

import (
	"context"
	"fmt"
	"io"
	"strings"

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
	skipPolicy    bool
	egressOn      bool
	egressList    []string
	egressMode    egress.Mode
	verbName      string
	scopeArgvHint func(argv []string) string
	argvRewriter  func(argv []string) []string
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

// WithVerbName overrides the dotted verb name used for audit logging.
// The user-visible cli command name (and the binary actually executed)
// stays as bin; only the audit verb path changes. Used to namespace
// pass-throughs that are mounted under a parent group, e.g. `coily ops
// aws` logs as verb "ops.aws" while still exec'ing the aws binary.
func WithVerbName(name string) Option {
	return func(c *config) {
		c.verbName = name
	}
}

// WithArgvRewriter installs a hook that may rewrite the argv passed to the
// underlying binary before exec. The original argv is still recorded in the
// audit log (verb.Wrap captures argv before Action runs); only what reaches
// the child process changes. Used by the gh wrapper to rewrite GraphQL-backed
// subcommands like `gh issue create` into REST equivalents via `gh api`, so
// the GraphQL secondary rate limit is dodged on the common write paths.
// Return the same slice unchanged to decline the rewrite.
func WithArgvRewriter(fn func(argv []string) []string) Option {
	return func(c *config) {
		c.argvRewriter = fn
	}
}

// WithScopeArgvHint installs a fallback --commit-scope resolver that runs
// only when the operator did not set --commit-scope (or COILY_COMMIT_SCOPE)
// explicitly. The hook receives the verb's argv and returns an absolute
// path or "" to decline. Used by `coily ops gh` to default the audit
// scope to ~/projects/coilysiren/<name> when --repo coilysiren/<name>
// is on the command line, so the agent does not have to also pass
// --commit-scope from outside a git repo.
func WithScopeArgvHint(fn func(argv []string) string) Option {
	return func(c *config) {
		c.scopeArgvHint = fn
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
	verbName := bin
	if cfg.verbName != "" {
		verbName = cfg.verbName
	}
	spec := verb.Spec{
		Name: verbName,
		ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
			return nil, c.Args().Slice()
		},
		SkipPolicy:          cfg.skipPolicy,
		CommitScopeArgvHint: cfg.scopeArgvHint,
	}
	if cfg.egressOn {
		spec.Action, spec.OnComplete = withEgressAction(bin, r, cfg.egressList, cfg.egressMode, cfg.argvRewriter)
	} else {
		spec.Action, spec.OnComplete = withStderrTail(bin, r, cfg.argvRewriter)
	}
	return &cli.Command{
		Name:            bin,
		Usage:           fmt.Sprintf("Pass-through to %s with argv validation + audit log.", bin),
		SkipFlagParsing: true,
		Action:          verb.Wrap(spec, w),
	}
}

// withStderrTail wraps the standard exec action so the wrapped tool's
// stderr is tee'd into a bounded ring buffer. On non-zero exit the captured
// tail is attached to the audit record via OnComplete. Issue #63: without
// this, kubectl/aws/gh failures in the audit log collapse to bare
// "exit status 1" and the operator/forensic reader can't tell what
// actually went wrong.
func withStderrTail(bin string, base *shell.Runner, rewriter func([]string) []string) (cli.ActionFunc, func(*audit.Record)) {
	tail := newTailBuffer(audit.MaxStderrTailBytes)
	action := func(ctx context.Context, c *cli.Command) error {
		shadow := *base
		// Tee stderr through the tail buffer while still streaming to the
		// caller's terminal. Falls back to just the tail buffer when the
		// runner has no stderr writer (test contexts).
		if base.Stderr != nil {
			shadow.Stderr = io.MultiWriter(base.Stderr, tail)
		} else {
			shadow.Stderr = tail
		}
		argv := c.Args().Slice()
		if rewriter != nil {
			argv = rewriter(argv)
		}
		return shadow.Exec(ctx, bin, argv...)
	}
	hook := func(rec *audit.Record) {
		if rec.ExitCode == 0 {
			return
		}
		if s := strings.TrimSpace(tail.String()); s != "" {
			rec.StderrTail = s
		}
	}
	return action, hook
}

// withEgressAction wraps the standard exec action to start a per-invocation
// proxy, inject HTTPS_PROXY/HTTP_PROXY into a per-call shadow Runner so
// concurrent commands stay independent, and surface the collected rows
// through OnComplete.
func withEgressAction(bin string, base *shell.Runner, allowlist []string, mode egress.Mode, rewriter func([]string) []string) (cli.ActionFunc, func(*audit.Record)) {
	var rows []audit.EgressRow
	tail := newTailBuffer(audit.MaxStderrTailBytes)
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
		if base.Stderr != nil {
			shadow.Stderr = io.MultiWriter(base.Stderr, tail)
		} else {
			shadow.Stderr = tail
		}
		argv := c.Args().Slice()
		if rewriter != nil {
			argv = rewriter(argv)
		}
		execErr := shadow.Exec(ctx, bin, argv...)
		rows = p.Stop()
		return execErr
	}
	hook := func(rec *audit.Record) {
		if len(rows) > 0 {
			rec.Egress = rows
		}
		if rec.ExitCode != 0 {
			if s := strings.TrimSpace(tail.String()); s != "" {
				rec.StderrTail = s
			}
		}
	}
	return action, hook
}

// tailBuffer is a fixed-size last-N-bytes ring. Writes never block and never
// error. After N total bytes, older bytes are dropped to keep the buffer
// bounded. String() returns the current contents in write order.
type tailBuffer struct {
	cap int
	buf []byte
}

func newTailBuffer(capBytes int) *tailBuffer {
	return &tailBuffer{cap: capBytes}
}

func (t *tailBuffer) Write(p []byte) (int, error) {
	n := len(p)
	if t.cap <= 0 {
		return n, nil
	}
	if len(p) >= t.cap {
		t.buf = append(t.buf[:0], p[len(p)-t.cap:]...)
		return n, nil
	}
	t.buf = append(t.buf, p...)
	if over := len(t.buf) - t.cap; over > 0 {
		t.buf = t.buf[over:]
	}
	return n, nil
}

func (t *tailBuffer) String() string {
	return string(t.buf)
}

package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/coilysiren/cli-guard/audit"
	"github.com/coilysiren/cli-guard/egress"
	"github.com/coilysiren/cli-guard/exitcode"
	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// brewCommand is the scoped wrapper around the four mutating brew verbs:
// install, uninstall, upgrade, reinstall. Default-allow is constrained
// to the coilysiren/homebrew-tap namespace; everything outside requires
// the --allow-untapped flag, which is the second confirmation on top of
// having typed the verb. This is the audited recovery path for the
// "brew upgrade of coily half-fails, agent needs to run more brew" case
// where the bare brew binary is denied at the harness (issue #90).
//
// Read-only verbs (search, info, list, ...) belong on `coily pkg brew`,
// the thin pass-through. They are unscoped by design: read-only brew
// queries are not a privileged op.
func (r *Runner) brewCommand() *cli.Command {
	return &cli.Command{
		Name:  "brew",
		Usage: "Scoped brew install/uninstall/upgrade/reinstall (coilysiren/tap/* default-allow).",
		Description: `brew is the audited wrapper around the four mutating brew verbs.
Formulae in coilysiren/tap/* (plus the bare formula names that resolve
to that tap: coily, repo-recall, arize-phoenix) are default-allowed.
Anything else requires --allow-untapped, which is the explicit
double-confirm gate on top of having typed the subcommand.

Use ` + "`coily pkg brew`" + ` for read-only verbs (search, info, list).
That pass-through is unscoped by design.

Subcommands forward verbatim to brew after the scope check. The
--allow-untapped flag is consumed by coily and never reaches brew.`,
		Commands: []*cli.Command{
			r.brewSubcommand("install"),
			r.brewSubcommand("uninstall"),
			r.brewSubcommand("upgrade"),
			r.brewSubcommand("reinstall"),
		},
	}
}

// scopedTapFormulae mirrors the bare formula names in
// coilysiren/homebrew-tap/Formula/. They are default-allowed without
// --allow-untapped because brew installs them under the user-controlled
// tap regardless of whether the operator typed the qualified name.
//
// Update this map in lockstep with the Formula directory in
// coilysiren/homebrew-tap. Drift means false-positive denials (bare
// name in the tap rejected) or false-negative allows (a same-named
// formula in homebrew-core slipping past the gate).
var scopedTapFormulae = map[string]bool{
	"coily":         true,
	"repo-recall":   true,
	"arize-phoenix": true,
}

const brewAllowFlag = "--allow-untapped"

// brewSubcommand wires one of install/uninstall/upgrade/reinstall. The
// audit verb is "brew.<sub>" so the row distinguishes it from the
// `coily pkg brew` pass-through (which audits as "brew").
func (r *Runner) brewSubcommand(sub string) *cli.Command {
	action, hook := r.brewAction(sub)
	return &cli.Command{
		Name:            sub,
		Usage:           fmt.Sprintf("brew %s, scoped to coilysiren/tap/* unless --allow-untapped is set.", sub),
		SkipFlagParsing: true,
		Action: r.WrapVerb(verb.Spec{
			Name:       "brew." + sub,
			SkipPolicy: true,
			ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
				return nil, c.Args().Slice()
			},
			Action:     action,
			OnComplete: hook,
		}, r.Audit),
	}
}

// brewAction returns the (action, onComplete) pair for a brew subcommand.
// The action splits raw argv into forwarded args and formula positionals,
// gates on scope, then forwards to brew with egress + stderr-tail wired
// the same way pkg/passthrough does. The hook attaches the egress rows
// and any captured stderr tail to the audit record.
func (r *Runner) brewAction(sub string) (cli.ActionFunc, func(*audit.Record)) {
	var rows []audit.EgressRow
	tail := newBrewTail(audit.MaxStderrTailBytes)
	action := func(ctx context.Context, c *cli.Command) error {
		raw := c.Args().Slice()
		allow, forward, formulae := splitBrewArgs(raw)

		if !allow {
			if sub == "upgrade" && len(formulae) == 0 {
				return exitcode.New(exitcode.PolicyDenied, "policy_denied",
					fmt.Errorf("brew upgrade with no formula upgrades every keg on the system; pass --allow-untapped to confirm"),
					"add --allow-untapped to confirm an upgrade-everything run, or name specific formulae")
			}
			for _, f := range formulae {
				if !brewInTapScope(f) {
					return exitcode.New(exitcode.PolicyDenied, "policy_denied",
						fmt.Errorf("brew %s %q is outside coilysiren/tap/*; pass --allow-untapped to confirm", sub, f),
						"prefix the formula with coilysiren/tap/ if it lives in that tap, or add --allow-untapped to confirm an off-tap formula")
				}
			}
		}

		captured, execErr := r.execBrew(ctx, sub, forward, tail)
		rows = captured
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

// splitBrewArgs walks the raw argv once. Pops --allow-untapped out of
// the forwarded args (it is a coily-side flag, brew has never heard of
// it). Treats anything not starting with '-' as a formula name for the
// scope check. Flag values (e.g. `--cask <name>`) are still scope-
// checked here, which is conservative but harmless: a flag value that
// happens to look like an off-tap formula trips the gate, and the fix
// is the same --allow-untapped opt-out.
func splitBrewArgs(raw []string) (allow bool, forward, formulae []string) {
	forward = make([]string, 0, len(raw))
	formulae = make([]string, 0, len(raw))
	for _, a := range raw {
		if a == brewAllowFlag {
			allow = true
			continue
		}
		forward = append(forward, a)
		if !strings.HasPrefix(a, "-") {
			formulae = append(formulae, a)
		}
	}
	return allow, forward, formulae
}

// brewInTapScope is true when f is either a coilysiren/tap/* qualified
// name or one of the bare formula names this tap publishes.
func brewInTapScope(f string) bool {
	if strings.HasPrefix(f, "coilysiren/tap/") {
		return true
	}
	return scopedTapFormulae[f]
}

// execBrew forwards to brew with the egress proxy + stderr tail wired in,
// same shape as pkg/passthrough's withEgressAction. Inlined here so the
// brew wrapper is self-contained and the passthrough package stays
// closed over its own Command builder. The caller-owned tail buffer
// receives a tee of brew's stderr; the egress rows are returned so
// the audit hook can attach them.
func (r *Runner) execBrew(ctx context.Context, sub string, args []string, tail *brewTail) ([]audit.EgressRow, error) {
	allow := egress.Allowlists["brew"]
	p := egress.New(allow, egress.ModeEnforce)
	proxyURL, err := p.Start(ctx)
	if err != nil {
		return nil, fmt.Errorf("egress: start proxy: %w", err)
	}
	shadow := *r.Runner
	shadow.Env = append([]string(nil), r.Runner.Env...)
	shadow.Env = append(shadow.Env,
		"HTTPS_PROXY="+proxyURL,
		"HTTP_PROXY="+proxyURL,
		"https_proxy="+proxyURL,
		"http_proxy="+proxyURL,
	)
	if r.Runner.Stderr != nil {
		shadow.Stderr = io.MultiWriter(r.Runner.Stderr, tail)
	} else {
		shadow.Stderr = tail
	}
	full := append([]string{sub}, args...)
	execErr := shadow.Exec(ctx, "brew", full...)
	rows := p.Stop()
	return rows, execErr
}

// brewTail is a fixed-size last-N-bytes ring, structurally identical to
// pkg/passthrough's tailBuffer. Duplicated rather than exported because
// the buffer is an implementation detail of the egress wrap, and the
// brew wrapper's egress wrap is itself a duplicate.
type brewTail struct {
	cap int
	buf []byte
}

func newBrewTail(capBytes int) *brewTail { return &brewTail{cap: capBytes} }

func (t *brewTail) String() string { return string(t.buf) }

func (t *brewTail) Write(p []byte) (int, error) {
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

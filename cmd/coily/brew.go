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

// pkgBrewCommand is the single entry point for every brew verb. Mirrors
// brew's argv shape exactly (coily-meta §4.5): one `coily pkg brew <verb>
// [args...]` surface for the whole brew CLI, with the mutating-vs-readonly
// distinction applied internally by classifyBrewInvocation rather than
// projected onto the verb path. Replaces the prior split between top-level
// `coily brew` (scoped install/uninstall/upgrade/reinstall + services
// start/stop/restart) and `coily pkg brew` (unscoped passthrough).
// coilysiren/coily#253.
//
// Categories:
//
//   - formula-scoped (default-allow coilysiren/tap/* + bare names in the
//     tap; --allow-untapped otherwise): install, uninstall, upgrade,
//     reinstall, link, unlink, pin, unpin.
//   - tap-scoped (default-allow coilysiren/* taps; --allow-untapped
//     otherwise): tap, untap.
//   - touch-everything (require --allow-untapped, no formula check):
//     cleanup, autoremove, services cleanup.
//   - services formula-scoped (same rule as the top-level formula set):
//     services start, services stop, services restart, services run,
//     services kill.
//   - passthrough (no scope, audit-logged): update, search, info, list,
//     deps, leaves, outdated, doctor, config, commands, services list,
//     services info, anything else.
//
// Audit verb names follow the ops.<area>.<sub> pattern: pkg.brew,
// pkg.brew.install, pkg.brew.services.restart, etc.
func (r *Runner) pkgBrewCommand() *cli.Command {
	return &cli.Command{
		Name:  "brew",
		Usage: "Scoped wrapper around brew. Mirrors brew's argv shape.",
		Description: `brew is the single entry point for every brew verb. Mutating verbs
that operate on a named formula default to coilysiren/tap/* scope and
require --allow-untapped otherwise. Tap mutations default to
coilysiren/* taps. Touch-everything verbs (cleanup, autoremove,
services cleanup) require --allow-untapped. Read-only verbs and
brew update pass through verbatim.

  formula-scoped:    install, uninstall, upgrade, reinstall,
                     link, unlink, pin, unpin
  tap-scoped:        tap, untap
  touch-everything:  cleanup, autoremove, services cleanup
  services-scoped:   services start, stop, restart, run, kill
  passthrough:       update, search, info, list, deps, leaves,
                     outdated, doctor, config, commands,
                     services list, services info, any other`,
		SkipFlagParsing: true,
		Action:          r.pkgBrewDispatch,
	}
}

// pkgBrewDispatch is the parent Action that routes by argv to one of the
// scoped action templates or the passthrough fallback. Constructs a
// verb.Spec dynamically with the right audit name so per-subverb rows
// stay distinguishable (pkg.brew.install vs pkg.brew.services.restart vs
// pkg.brew.search).
func (r *Runner) pkgBrewDispatch(ctx context.Context, c *cli.Command) error {
	raw := c.Args().Slice()
	auditName, action, hook := r.classifyBrewInvocation(raw)
	spec := verb.Spec{
		Name:       auditName,
		SkipPolicy: true,
		ArgsFunc: func(_ *cli.Command) (map[string]string, []string) {
			return nil, raw
		},
		Action:     action,
		OnComplete: hook,
	}
	return r.WrapVerb(spec, r.Audit)(ctx, c)
}

// brewFormulaScopedVerbs are the mutating brew verbs whose positionals
// are formula names and which should default-allow coilysiren/tap/*.
var brewFormulaScopedVerbs = map[string]bool{
	"install":   true,
	"uninstall": true,
	"upgrade":   true,
	"reinstall": true,
	"link":      true,
	"unlink":    true,
	"pin":       true,
	"unpin":     true,
}

// brewTapScopedVerbs are the verbs that mutate the registered-tap set.
// Their positionals are <user>/<repo> tap names; the gate is "starts
// with coilysiren/".
var brewTapScopedVerbs = map[string]bool{
	"tap":   true,
	"untap": true,
}

// brewTouchEverythingVerbs operate on every keg or every service. No
// formula check; the gate is "did the operator type --allow-untapped".
// Same shape as the existing bare `brew upgrade` gate.
var brewTouchEverythingVerbs = map[string]bool{
	"cleanup":    true,
	"autoremove": true,
}

// brewServicesFormulaScopedSubs are the services subverbs that take a
// formula positional and follow the formula-scoped rule.
var brewServicesFormulaScopedSubs = map[string]bool{
	"start":   true,
	"stop":    true,
	"restart": true,
	"run":     true,
	"kill":    true,
}

// brewServicesTouchEverythingSubs are the services subverbs that touch
// every service. --allow-untapped required.
var brewServicesTouchEverythingSubs = map[string]bool{
	"cleanup": true,
}

// classifyBrewInvocation picks the audit name + action template for one
// invocation. The action closure captures the verb and rest args; the
// hook closure attaches egress + stderr-tail to the audit row after exec.
func (r *Runner) classifyBrewInvocation(raw []string) (string, cli.ActionFunc, func(*audit.Record)) {
	if len(raw) == 0 {
		// Bare `coily pkg brew` → passthrough (brew shows its top-level help).
		return "pkg.brew", r.brewPassthroughAction(nil), nil
	}
	verbName := raw[0]
	rest := raw[1:]
	switch {
	case verbName == "services":
		return r.classifyBrewServices(rest)
	case brewFormulaScopedVerbs[verbName]:
		action, hook := r.brewFormulaScopedAction([]string{verbName}, rest)
		return "pkg.brew." + verbName, action, hook
	case brewTapScopedVerbs[verbName]:
		action, hook := r.brewTapScopedAction([]string{verbName}, rest)
		return "pkg.brew." + verbName, action, hook
	case brewTouchEverythingVerbs[verbName]:
		action, hook := r.brewTouchEverythingAction([]string{verbName}, rest)
		return "pkg.brew." + verbName, action, hook
	default:
		return "pkg.brew." + verbName, r.brewPassthroughAction(raw), nil
	}
}

// classifyBrewServices is the `services <sub>` arm of the dispatcher.
// Bare `services` (no sub) and `services <unknown>` both fall through
// to passthrough so brew's own help can fire.
func (r *Runner) classifyBrewServices(rest []string) (string, cli.ActionFunc, func(*audit.Record)) {
	if len(rest) == 0 {
		return "pkg.brew.services", r.brewPassthroughAction([]string{"services"}), nil
	}
	sub := rest[0]
	tail := rest[1:]
	prefix := []string{"services", sub}
	switch {
	case brewServicesFormulaScopedSubs[sub]:
		action, hook := r.brewFormulaScopedAction(prefix, tail)
		return "pkg.brew.services." + sub, action, hook
	case brewServicesTouchEverythingSubs[sub]:
		action, hook := r.brewTouchEverythingAction(prefix, tail)
		return "pkg.brew.services." + sub, action, hook
	default:
		// services list, services info, anything else: passthrough.
		return "pkg.brew.services." + sub, r.brewPassthroughAction(append([]string{"services"}, rest...)), nil
	}
}

// brewFormulaScopedAction builds an action that scope-checks formula
// positionals against coilysiren/tap/*. prefix is the verb chain
// already consumed (e.g. ["install"] or ["services", "restart"]); rest
// is what came after.
func (r *Runner) brewFormulaScopedAction(prefix, rest []string) (cli.ActionFunc, func(*audit.Record)) {
	var rows []audit.EgressRow
	tail := newBrewTail()
	action := func(ctx context.Context, _ *cli.Command) error {
		allow, forward, formulae := splitBrewArgs(rest)
		verbLabel := strings.Join(prefix, " ")
		if !allow {
			// Bare `brew upgrade` (no formula) touches every keg; treat as
			// touch-everything for this one verb. The other formula-scoped
			// verbs without formulae are already invalid invocations brew
			// will reject downstream, but we surface a clearer error here.
			if len(formulae) == 0 {
				if len(prefix) == 1 && prefix[0] == "upgrade" {
					return exitcode.New(exitcode.PolicyDenied, "policy_denied",
						fmt.Errorf("brew upgrade with no formula upgrades every keg on the system; pass --allow-untapped to confirm"),
						"add --allow-untapped to confirm an upgrade-everything run, or name specific formulae").
						WithReason("bare `brew upgrade` touches every installed keg, turning a single-intent invocation into a global state shift on a long-lived host; --allow-untapped is the explicit opt-in")
				}
				// Fall through: brew will surface its own usage error.
			}
			for _, f := range formulae {
				if !brewInTapScope(f) {
					return exitcode.New(exitcode.PolicyDenied, "policy_denied",
						fmt.Errorf("brew %s %q is outside the known coilysiren taps; pass --allow-untapped to confirm", verbLabel, f),
						"qualify the formula with a coilysiren/<tap>/ prefix (e.g. coilysiren/tap/, coilysiren/coily/), or add --allow-untapped to confirm an off-tap formula").
						WithReason("brew state should live under a coilysiren-owned tap by default so the install graph is reviewable from coilysiren repos; --allow-untapped is the explicit opt-out for one-offs")
				}
			}
		}
		captured, execErr := r.execBrew(ctx, prefix, forward, tail)
		rows = captured
		return execErr
	}
	hook := makeBrewHook(&rows, tail)
	return action, hook
}

// brewTapScopedAction scopes `brew tap` / `brew untap` on the tap name,
// which is the first positional after the verb. Bare `brew tap` (no
// positional) lists configured taps and is a read-only operation; let
// it pass through to brew.
func (r *Runner) brewTapScopedAction(prefix, rest []string) (cli.ActionFunc, func(*audit.Record)) {
	var rows []audit.EgressRow
	tail := newBrewTail()
	action := func(ctx context.Context, _ *cli.Command) error {
		allow, forward, positionals := splitBrewArgs(rest)
		verbLabel := strings.Join(prefix, " ")
		if !allow && len(positionals) > 0 {
			for _, t := range positionals {
				if !brewTapPositionalAllowed(t) {
					return exitcode.New(exitcode.PolicyDenied, "policy_denied",
						fmt.Errorf("brew %s %q is outside coilysiren/*; pass --allow-untapped to confirm", verbLabel, t),
						"name a coilysiren/<repo> tap or a forgejo.coilysiren.me/coilysiren/<repo> URL, or add --allow-untapped to confirm an off-org tap").
						WithReason("brew-tap state should be coilysiren/* by default so the registered-tap graph is reviewable from one org; --allow-untapped is the explicit opt-out")
				}
			}
		}
		captured, execErr := r.execBrew(ctx, prefix, forward, tail)
		rows = captured
		return execErr
	}
	hook := makeBrewHook(&rows, tail)
	return action, hook
}

// brewTouchEverythingAction requires --allow-untapped before forwarding.
// Used for cleanup / autoremove / services cleanup whose effect is
// "touch every installed thing in this category."
func (r *Runner) brewTouchEverythingAction(prefix, rest []string) (cli.ActionFunc, func(*audit.Record)) {
	var rows []audit.EgressRow
	tail := newBrewTail()
	action := func(ctx context.Context, _ *cli.Command) error {
		allow, forward, _ := splitBrewArgs(rest)
		verbLabel := strings.Join(prefix, " ")
		if !allow {
			return exitcode.New(exitcode.PolicyDenied, "policy_denied",
				fmt.Errorf("brew %s touches every installed item in this category; pass --allow-untapped to confirm", verbLabel),
				"add --allow-untapped to confirm a touch-everything run").
				WithReason("touch-everything brew verbs turn a single invocation into a global state shift; --allow-untapped is the explicit opt-in")
		}
		captured, execErr := r.execBrew(ctx, prefix, forward, tail)
		rows = captured
		return execErr
	}
	hook := makeBrewHook(&rows, tail)
	return action, hook
}

// brewPassthroughAction forwards the original raw argv to brew with no
// scope check. Used for read-only verbs (search, info, list, ...),
// brew update (no formula at risk), and any verb we didn't recognize
// so brew can surface its own usage error. Argv is forwarded verbatim;
// --allow-untapped is consumed (a coily-side flag) so it doesn't leak
// to brew which has never heard of it.
func (r *Runner) brewPassthroughAction(raw []string) cli.ActionFunc {
	var rows []audit.EgressRow
	tail := newBrewTail()
	return func(ctx context.Context, _ *cli.Command) error {
		_, forward, _ := splitBrewArgs(raw)
		captured, execErr := r.execBrewRaw(ctx, forward, tail)
		rows = captured
		_ = rows // captured for the optional hook; current passthrough Spec passes nil
		return execErr
	}
}

// makeBrewHook returns an OnComplete closure that attaches captured
// egress rows and a stderr tail (when exit was non-zero) to the audit
// row. rows is a pointer because the action captures into a closure
// variable that gets populated mid-action.
func makeBrewHook(rows *[]audit.EgressRow, tail *brewTail) func(*audit.Record) {
	return func(rec *audit.Record) {
		if len(*rows) > 0 {
			rec.Egress = *rows
		}
		if rec.ExitCode != 0 {
			if s := strings.TrimSpace(tail.String()); s != "" {
				rec.StderrTail = s
			}
		}
	}
}

// scopedTapFormulae mirrors the bare formula names in
// coilysiren/homebrew-tap/Formula/. They are default-allowed without
// --allow-untapped because brew installs them under the user-controlled
// tap regardless of whether the operator typed the qualified name.
//
// Update this map in lockstep with the Formula directory in
// coilysiren/homebrew-tap.
var scopedTapFormulae = map[string]bool{
	"coily":         true,
	"repo-recall":   true,
	"arize-phoenix": true,
}

const brewAllowFlag = "--allow-untapped"

// splitBrewArgs walks the raw argv once. Pops --allow-untapped out of
// the forwarded args. Treats anything not starting with '-' as a
// positional (formula or tap name) for the scope check.
func splitBrewArgs(raw []string) (allow bool, forward, positionals []string) {
	forward = make([]string, 0, len(raw))
	positionals = make([]string, 0, len(raw))
	for _, a := range raw {
		if a == brewAllowFlag {
			allow = true
			continue
		}
		forward = append(forward, a)
		if !strings.HasPrefix(a, "-") {
			positionals = append(positionals, a)
		}
	}
	return allow, forward, positionals
}

// brewTapPositionalAllowed accepts the positionals to `brew tap` /
// `brew untap`: either a `coilysiren/<repo>` tap name or a tap source
// URL under forgejo.coilysiren.me/coilysiren/. Forgejo is the canonical
// source of truth for the brew taps (per-repo Formula clone URLs point
// at forgejo); the tap-name prefix still covers the bare `brew untap
// coilysiren/<repo>` form.
func brewTapPositionalAllowed(t string) bool {
	if strings.HasPrefix(t, "coilysiren/") {
		return true
	}
	return strings.HasPrefix(t, "https://forgejo.coilysiren.me/coilysiren/")
}

// brewInTapScope is true when f is either a coilysiren/<tap>/<formula>
// qualified name (umbrella `coilysiren/tap/*` or any per-repo tap like
// `coilysiren/coily/coily`) or one of the bare formula names the
// umbrella tap publishes. Widened from coilysiren/tap/* to all
// coilysiren/* taps in coilysiren/coily#271 to cover the per-repo tap
// release flow; the policy intent is "any coilysiren-owned tap".
func brewInTapScope(f string) bool {
	if strings.HasPrefix(f, "coilysiren/") {
		parts := strings.Split(f, "/")
		if len(parts) == 3 && parts[1] != "" && parts[2] != "" {
			return true
		}
	}
	return scopedTapFormulae[f]
}

// execBrew forwards `brew <prefix...> <forward...>` with the egress
// proxy + stderr tail wired in. prefix is the verb chain ("install",
// "services restart", etc.) and forward is the rest of the argv.
func (r *Runner) execBrew(ctx context.Context, prefix, forward []string, tail *brewTail) ([]audit.EgressRow, error) {
	full := append([]string{}, prefix...)
	full = append(full, forward...)
	return r.execBrewRaw(ctx, full, tail)
}

// execBrewRaw forwards `brew <argv...>` verbatim, same egress + tail
// wiring as the scoped variant. Used by the passthrough action where
// no prefix split is meaningful.
func (r *Runner) execBrewRaw(ctx context.Context, argv []string, tail *brewTail) ([]audit.EgressRow, error) {
	// ModeObserve: every CONNECT is forwarded and logged. The brew
	// install path for tap formulae spawns `go build`, which hits a
	// wide and unstable set of hosts; enforcing an allowlist here is
	// fragile. The audit row still captures egress rows in observe
	// mode. coilysiren/coily#195.
	p := egress.New(nil, egress.ModeObserve)
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
	execErr := shadow.Exec(ctx, "brew", argv...)
	rows := p.Stop()
	return rows, execErr
}

// brewTail is a fixed-size last-N-bytes ring.
type brewTail struct {
	cap int
	buf []byte
}

func newBrewTail() *brewTail { return &brewTail{cap: audit.MaxStderrTailBytes} }

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

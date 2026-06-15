package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/cli/verb"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/http/egress"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/pkg/audit"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/pkg/exitcode"
	"github.com/urfave/cli/v3"
)

// pkgScoopCommand is the single entry point for every scoop verb. Mirrors
// scoop's argv shape exactly: one `coily pkg scoop <verb> [args...]` surface
// for the whole scoop CLI, with the mutating-vs-readonly distinction applied
// internally by classifyScoopInvocation rather than projected onto the verb
// path. Structural twin of pkgBrewCommand (coily#329).
//
// Categories:
//
//   - app-scoped (default-allow coilysiren/* qualified names + bare names in
//     the coilysiren bucket; --allow-untapped otherwise): install, uninstall,
//     update <app>, reset, hold, unhold.
//   - bucket-scoped (default-allow coilysiren alias or
//     github.com/coilysiren/* URL; --allow-untapped otherwise): bucket add,
//     bucket rm.
//   - touch-everything (require --allow-untapped, no app check): cleanup,
//     bare `update` with no app, `update *`.
//   - passthrough (no scope, audit-logged): search, info, list, status,
//     which, prefix, depends, home, bucket list, bucket known, config,
//     anything else.
//
// Audit verb names follow the ops.<area>.<sub> pattern: pkg.scoop,
// pkg.scoop.install, pkg.scoop.bucket.add, etc.
func (r *Runner) pkgScoopCommand() *cli.Command {
	return &cli.Command{
		Name:  "scoop",
		Usage: "Scoped wrapper around scoop. Mirrors scoop's argv shape.",
		Description: `scoop is the single entry point for every scoop verb. Mutating verbs
that operate on a named app default to the coilysiren scoop bucket and
require --allow-untapped otherwise. Bucket mutations default to
coilysiren-owned buckets. Touch-everything verbs (cleanup, bare update)
require --allow-untapped. Read-only verbs pass through verbatim.

  app-scoped:        install, uninstall, update <app>, reset,
                     hold, unhold
  bucket-scoped:     bucket add, bucket rm
  touch-everything:  cleanup, bare update, update *
  passthrough:       search, info, list, status, which, prefix,
                     depends, home, bucket list, bucket known,
                     config, any other`,
		SkipFlagParsing: true,
		Action:          r.pkgScoopDispatch,
	}
}

// pkgScoopDispatch is the parent Action that routes by argv to one of the
// scoped action templates or the passthrough fallback. Constructs a
// verb.Spec dynamically with the right audit name so per-subverb rows
// stay distinguishable (pkg.scoop.install vs pkg.scoop.bucket.add vs
// pkg.scoop.search).
func (r *Runner) pkgScoopDispatch(ctx context.Context, c *cli.Command) error {
	raw := c.Args().Slice()
	auditName, action, hook := r.classifyScoopInvocation(raw)
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

// scoopAppScopedVerbs are the mutating scoop verbs whose positionals are
// app names and which should default-allow the coilysiren scoop bucket.
// `update` is handled separately because bare `scoop update` touches every
// installed app and needs the touch-everything gate.
var scoopAppScopedVerbs = map[string]bool{
	"install":   true,
	"uninstall": true,
	"reset":     true,
	"hold":      true,
	"unhold":    true,
}

// scoopBucketScopedSubs are the bucket subverbs that mutate the registered-
// bucket set. Positionals are an alias and optionally a URL; the gate is
// "alias is coilysiren OR URL is under github.com/coilysiren/".
var scoopBucketScopedSubs = map[string]bool{
	"add": true,
	"rm":  true,
}

// classifyScoopInvocation picks the audit name + action template for one
// invocation. The action closure captures the verb and rest args; the hook
// closure attaches egress + stderr-tail to the audit row after exec.
func (r *Runner) classifyScoopInvocation(raw []string) (string, cli.ActionFunc, func(*audit.Record)) {
	if len(raw) == 0 {
		return "pkg.scoop", r.scoopPassthroughAction(nil), nil
	}
	verbName := raw[0]
	rest := raw[1:]
	switch {
	case verbName == "bucket":
		return r.classifyScoopBucket(rest)
	case verbName == "update":
		// `update <app>` is app-scoped; bare `update` or `update *` is
		// touch-everything. splitScoopArgs decides which.
		return r.classifyScoopUpdate(rest)
	case verbName == "cleanup":
		action, hook := r.scoopTouchEverythingAction([]string{verbName}, rest)
		return "pkg.scoop." + verbName, action, hook
	case scoopAppScopedVerbs[verbName]:
		action, hook := r.scoopAppScopedAction([]string{verbName}, rest)
		return "pkg.scoop." + verbName, action, hook
	default:
		return "pkg.scoop." + verbName, r.scoopPassthroughAction(raw), nil
	}
}

// classifyScoopBucket is the `bucket <sub>` arm of the dispatcher. Bare
// `bucket` (no sub) and `bucket list` / `bucket known` / unknown subs all
// fall through to passthrough so scoop's own help can fire.
func (r *Runner) classifyScoopBucket(rest []string) (string, cli.ActionFunc, func(*audit.Record)) {
	if len(rest) == 0 {
		return "pkg.scoop.bucket", r.scoopPassthroughAction([]string{"bucket"}), nil
	}
	sub := rest[0]
	tail := rest[1:]
	prefix := []string{"bucket", sub}
	if scoopBucketScopedSubs[sub] {
		action, hook := r.scoopBucketScopedAction(prefix, tail)
		return "pkg.scoop.bucket." + sub, action, hook
	}
	// bucket list, bucket known, anything else: passthrough.
	return "pkg.scoop.bucket." + sub, r.scoopPassthroughAction(append([]string{"bucket"}, rest...)), nil
}

// classifyScoopUpdate splits `update` into the app-scoped and
// touch-everything arms. `scoop update` with no positionals (or `*`) updates
// every installed app; `scoop update <app>` updates just that app.
func (r *Runner) classifyScoopUpdate(rest []string) (string, cli.ActionFunc, func(*audit.Record)) {
	_, _, positionals := splitScoopArgs(rest)
	if scoopUpdateIsAll(positionals) {
		action, hook := r.scoopTouchEverythingAction([]string{"update"}, rest)
		return "pkg.scoop.update", action, hook
	}
	action, hook := r.scoopAppScopedAction([]string{"update"}, rest)
	return "pkg.scoop.update", action, hook
}

// scoopUpdateIsAll is true when the update target is "everything": no
// positionals, or a single `*` positional. Anything else names specific
// apps and is app-scoped.
func scoopUpdateIsAll(positionals []string) bool {
	if len(positionals) == 0 {
		return true
	}
	if len(positionals) == 1 && positionals[0] == "*" {
		return true
	}
	return false
}

// scoopAppScopedAction builds an action that scope-checks app positionals
// against the coilysiren bucket. prefix is the verb chain already consumed
// (e.g. ["install"] or ["update"]); rest is what came after.
func (r *Runner) scoopAppScopedAction(prefix, rest []string) (cli.ActionFunc, func(*audit.Record)) {
	var rows []audit.EgressRow
	tail := newBrewTail()
	action := func(ctx context.Context, _ *cli.Command) error {
		allow, forward, apps := splitScoopArgs(rest)
		verbLabel := strings.Join(prefix, " ")
		if !allow {
			for _, a := range apps {
				if !scoopInBucketScope(a) {
					return exitcode.New(exitcode.PolicyDenied, "policy_denied",
						fmt.Errorf("scoop %s %q is outside the coilysiren scoop bucket; pass --allow-untapped to confirm", verbLabel, a),
						"qualify the app with a coilysiren/ prefix (e.g. coilysiren/coily), or add --allow-untapped to confirm an off-bucket app").
						WithReason("scoop state should live under a coilysiren-owned bucket by default so the install graph is reviewable from coilysiren repos; --allow-untapped is the explicit opt-out for one-offs")
				}
			}
		}
		captured, execErr := r.execScoop(ctx, prefix, forward, tail)
		rows = captured
		return execErr
	}
	hook := makeBrewHook(&rows, tail)
	return action, hook
}

// scoopBucketScopedAction scopes `scoop bucket add` / `scoop bucket rm` on
// the bucket alias + optional URL. The gate is "alias is coilysiren OR URL
// is under github.com/coilysiren/". Bare `bucket add` (no positional) falls
// through to scoop, which will surface its own usage error.
func (r *Runner) scoopBucketScopedAction(prefix, rest []string) (cli.ActionFunc, func(*audit.Record)) {
	var rows []audit.EgressRow
	tail := newBrewTail()
	action := func(ctx context.Context, _ *cli.Command) error {
		allow, forward, positionals := splitScoopArgs(rest)
		verbLabel := strings.Join(prefix, " ")
		if !allow && len(positionals) > 0 && !scoopBucketPositionalsInScope(positionals) {
			return exitcode.New(exitcode.PolicyDenied, "policy_denied",
				fmt.Errorf("scoop %s %v is outside coilysiren/*; pass --allow-untapped to confirm", verbLabel, positionals),
				"name the bucket `coilysiren` (with optional https://github.com/coilysiren/<repo> URL), or add --allow-untapped to confirm an off-org bucket").
				WithReason("scoop-bucket state should be coilysiren-owned by default so the registered-bucket graph is reviewable from one org; --allow-untapped is the explicit opt-out")
		}
		captured, execErr := r.execScoop(ctx, prefix, forward, tail)
		rows = captured
		return execErr
	}
	hook := makeBrewHook(&rows, tail)
	return action, hook
}

// scoopTouchEverythingAction requires --allow-untapped before forwarding.
// Used for cleanup and bare `scoop update` whose effect is "touch every
// installed app."
func (r *Runner) scoopTouchEverythingAction(prefix, rest []string) (cli.ActionFunc, func(*audit.Record)) {
	var rows []audit.EgressRow
	tail := newBrewTail()
	action := func(ctx context.Context, _ *cli.Command) error {
		allow, forward, _ := splitScoopArgs(rest)
		verbLabel := strings.Join(prefix, " ")
		if !allow {
			return exitcode.New(exitcode.PolicyDenied, "policy_denied",
				fmt.Errorf("scoop %s touches every installed app; pass --allow-untapped to confirm", verbLabel),
				"add --allow-untapped to confirm a touch-everything run, or name specific apps").
				WithReason("touch-everything scoop verbs turn a single invocation into a global state shift; --allow-untapped is the explicit opt-in")
		}
		captured, execErr := r.execScoop(ctx, prefix, forward, tail)
		rows = captured
		return execErr
	}
	hook := makeBrewHook(&rows, tail)
	return action, hook
}

// scoopPassthroughAction forwards the original raw argv to scoop with no
// scope check. Used for read-only verbs (search, info, list, status, ...)
// and any verb we did not recognize so scoop can surface its own usage
// error. --allow-untapped is consumed (a coily-side flag) so it does not
// leak to scoop which has never heard of it.
func (r *Runner) scoopPassthroughAction(raw []string) cli.ActionFunc {
	var rows []audit.EgressRow
	tail := newBrewTail()
	return func(ctx context.Context, _ *cli.Command) error {
		_, forward, _ := splitScoopArgs(raw)
		captured, execErr := r.execScoopRaw(ctx, forward, tail)
		rows = captured
		_ = rows
		return execErr
	}
}

// scopedBucketApps mirrors the bare app names in coilysiren/scoop-bucket
// bucket/. They are default-allowed without --allow-untapped because scoop
// installs them under the user-controlled bucket regardless of whether the
// operator typed the qualified name.
//
// Update this map in lockstep with the bucket/ directory in
// coilysiren/scoop-bucket.
var scopedBucketApps = map[string]bool{
	"coily": true,
}

// splitScoopArgs walks the raw argv once. Pops --allow-untapped out of the
// forwarded args. Treats anything not starting with '-' as a positional
// (app name, bucket alias, or URL) for the scope check.
func splitScoopArgs(raw []string) (allow bool, forward, positionals []string) {
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

// scoopInBucketScope is true when a is either a coilysiren/<app> qualified
// name or one of the bare app names the coilysiren bucket publishes.
func scoopInBucketScope(a string) bool {
	if strings.HasPrefix(a, "coilysiren/") {
		rest := strings.TrimPrefix(a, "coilysiren/")
		if rest != "" && !strings.Contains(rest, "/") {
			return true
		}
	}
	return scopedBucketApps[a]
}

// scoopBucketPositionalsInScope is true when the bucket-add / bucket-rm
// positionals name a coilysiren-owned bucket. Accepts either the literal
// alias "coilysiren" or any positional that looks like a coilysiren-owned
// git URL (https or git+ssh).
func scoopBucketPositionalsInScope(positionals []string) bool {
	for _, p := range positionals {
		if p == "coilysiren" {
			return true
		}
		if strings.HasPrefix(p, "https://github.com/coilysiren/") {
			return true
		}
		if strings.HasPrefix(p, "git@github.com:coilysiren/") {
			return true
		}
	}
	return false
}

// execScoop forwards `scoop <prefix...> <forward...>` with the egress proxy
// + stderr tail wired in. prefix is the verb chain ("install", "bucket add",
// etc.) and forward is the rest of the argv.
func (r *Runner) execScoop(ctx context.Context, prefix, forward []string, tail *brewTail) ([]audit.EgressRow, error) {
	full := append([]string{}, prefix...)
	full = append(full, forward...)
	return r.execScoopRaw(ctx, full, tail)
}

// execScoopRaw forwards `scoop <argv...>` verbatim, same egress + tail
// wiring as the scoped variant. Used by the passthrough action where no
// prefix split is meaningful.
func (r *Runner) execScoopRaw(ctx context.Context, argv []string, tail *brewTail) ([]audit.EgressRow, error) {
	// ModeObserve: every CONNECT is forwarded and logged. scoop install
	// hits a wide set of hosts (the bucket repo, the app's release host,
	// shim CDNs); enforcing an allowlist here is fragile. Same reasoning
	// as execBrewRaw.
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
	execErr := shadow.Exec(ctx, "scoop", argv...)
	rows := p.Stop()
	return rows, execErr
}

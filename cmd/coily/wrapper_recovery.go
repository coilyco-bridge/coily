package main

import "fmt"

// recoveryWrapper pairs a bare binary an agent might reach for with the
// audited coily invocation that should replace it. It is the atom both
// wrapperRecovery (the PreToolUse recovery-hint + deny-handoff map) and
// wrapperAllows (the explicit settings.json allow map) are projected from,
// so the two surfaces - and `coily --tree --json` - cannot drift from what
// coily actually fronts. coilyco-bridge/coily#197.
//
// History: wrapperRecovery and wrapperAllows used to be hand-maintained map
// literals in two files, and had already diverged - recovery was missing
// flyctl/gcloud/mcporter/netlify, allows were missing mcporter/netlify/nix/
// scoop. A binary that is deny-trimmed + hook-routed but lacks its explicit
// allow re-opens the #43 classifier-asymmetry bug, so the two must move
// together. Deriving both from one source closes that whole class.
type recoveryWrapper struct {
	Bin     string // the bare binary (and the deny-rule token)
	Wrapper string // the audited coily invocation that replaces it
}

// passthroughWrappers derives the recovery mappings for every thin
// passthrough coily fronts, straight from the passthrough registries
// (ptOps/ptTopLevel/ptPkg in passthroughs.go) - the exact set
// `coily --tree --json` enumerates as leaves carrying a bin. The wrapper
// path mirrors where each registry mounts: ops under `coily ops <bin>`,
// top-level under `coily <bin>`, package managers under `coily pkg <bin>`.
func passthroughWrappers() []recoveryWrapper {
	out := make([]recoveryWrapper, 0, len(ptOps)+len(ptTopLevel)+len(ptPkg))
	for _, e := range ptOps {
		out = append(out, recoveryWrapper{e.Bin, "coily ops " + e.Bin})
	}
	for _, e := range ptTopLevel {
		out = append(out, recoveryWrapper{e.Bin, "coily " + e.Bin})
	}
	for _, e := range ptPkg {
		out = append(out, recoveryWrapper{e.Bin, "coily pkg " + e.Bin})
	}
	return out
}

// scopedWrappers are bare binaries whose audited replacement is a coily
// command that is NOT a thin passthrough leaf, so it cannot come from the
// passthrough registries. brew has its own formula/tap-scoped wrapper
// (pkgBrewCommand, coily#253) and scoop likewise, so neither sits in ptPkg -
// but bare `brew`/`scoop` still must route to the audited form.
var scopedWrappers = []recoveryWrapper{
	{"brew", "coily pkg brew"},
	{"scoop", "coily pkg scoop"},
}

// buildRunnerWrappers route bare build runners to the named-verb dispatch.
// Their replacement is repo-specific (`coily exec <verb>`), so they are a
// recovery HINT only - never an allow, since there is no fixed allowable
// `Bash(coily exec <verb>:*)` prefix to sanction.
var buildRunnerWrappers = []recoveryWrapper{
	{"make", "coily exec <verb>"},
	{"just", "coily exec <verb>"},
	{"task", "coily exec <verb>"},
	{"invoke", "coily exec <verb>"},
}

// recoveryWrappers is the full bin->wrapper set for the PreToolUse recovery
// surface: passthroughs (generated) + scoped pkg wrappers + build runners.
func recoveryWrappers() []recoveryWrapper {
	out := passthroughWrappers()
	out = append(out, scopedWrappers...)
	out = append(out, buildRunnerWrappers...)
	return out
}

// allowableWrappers is the subset that earns an explicit settings.json
// allow: passthroughs + scoped pkg wrappers. Build runners are excluded
// (no fixed allow prefix).
func allowableWrappers() []recoveryWrapper {
	out := passthroughWrappers()
	out = append(out, scopedWrappers...)
	return out
}

// wrapperRecovery maps a denied bare-binary leading-token to the coily
// wrapper that should be used instead. Source of truth for cross-repo
// recovery hints (issue #122): every coily wrapper that shadows a denied
// bare binary lands here so `coily lockdown` renders the hint into each
// repo's lockdown-deny.sh and the PreToolUse hook routes off it. Generated
// from recoveryWrappers() so it tracks the passthrough registries. Issues
// #61, #122, #197.
var wrapperRecovery = func() map[string]string {
	m := make(map[string]string)
	for _, w := range recoveryWrappers() {
		m[w.Bin] = w.Wrapper
	}
	return m
}()

// wrapperAllows maps a bare-binary deny entry (the deny-list shape baked
// into cli-guard's defaults.yaml) to the explicit `Bash(coily ...:*)` allow
// that names the audited route. The bare-deny is what we reject; the
// coily-prefixed allow is what we sanction. The Claude Code auto-mode
// classifier strips `coily tailscale status` back to `tailscale status` and
// re-applies the bare-binary deny (#159, #43), so the explicit allow tells
// the classifier "this exact form is the audited path the deny was carved
// around." Generated from allowableWrappers() so it can never fall out of
// sync with wrapperRecovery again. Issues #115, #43, #197.
var wrapperAllows = func() map[string]string {
	m := make(map[string]string)
	for _, w := range allowableWrappers() {
		m[fmt.Sprintf("Bash(%s:*)", w.Bin)] = fmt.Sprintf("Bash(%s:*)", w.Wrapper)
	}
	return m
}()

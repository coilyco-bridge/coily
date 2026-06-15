package main

import (
	"fmt"
	"strings"
	"testing"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/cli/lockdown"
)

// TestApplyHookHandoffTrim_DropsWrappedBareDenies pins the contract
// from coilysiren/coily#183 and coilysiren/coily#185: for every token
// in wrapperRecovery (the bare binaries coily's PreToolUse hook
// gates), the bare `Bash(<token>:*)` deny entry must not survive
// the trim pass. Otherwise Claude Code CLI's built-in deny matcher
// fires first and clobbers the hook's recovery hint.
func TestApplyHookHandoffTrim_DropsWrappedBareDenies(t *testing.T) {
	d, err := lockdown.LoadDefaults()
	if err != nil {
		t.Fatalf("LoadDefaults: %v", err)
	}
	got := applyHookHandoffTrim(d)
	for token := range wrapperRecovery {
		bareDeny := fmt.Sprintf("Bash(%s:*)", token)
		if containsString(got.Deny, bareDeny) {
			t.Errorf("expected %q to be trimmed from deny list, but it survived", bareDeny)
		}
	}
}

// TestApplyHookHandoffTrim_PreservesExplicitWrapperAllows pins the
// reversal from coilyco-bridge/coily#43: the explicit `Bash(coily X:*)`
// allow MUST survive the hook-handoff trim. The auto-mode classifier
// reasons off the user-level deny set (which keeps the full, untrimmed
// `Bash(gh:*)` deny via the `--user`/ancestor merge), so it flags
// `coily ops gh` as deny circumvention even after the per-repo bare deny
// is trimmed. The positive allow is the explicit sanction that disarms
// the classifier - trimming it (the old behavior) reopened #43.
func TestApplyHookHandoffTrim_PreservesExplicitWrapperAllows(t *testing.T) {
	d, err := lockdown.LoadDefaults()
	if err != nil {
		t.Fatalf("LoadDefaults: %v", err)
	}
	// Same pipeline order lockdownOne uses: trim runs first, then the
	// explicit allows are added back. The end state must carry them.
	got := applyWrapperAllows(applyHookHandoffTrim(d))
	for token := range wrapperRecovery {
		bareDeny := fmt.Sprintf("Bash(%s:*)", token)
		wantedAllow, ok := wrapperAllows[bareDeny]
		if !ok {
			continue
		}
		if !containsString(got.Allow, wantedAllow) {
			t.Errorf("expected %q to survive the hook handoff (it is the classifier's sanction signal), but it was dropped", wantedAllow)
		}
	}
}

// TestLockdownPipeline_GhWrapperSanctioned is the end-to-end regression
// for coilyco-bridge/coily#43: after the full stampDefaults pipeline
// (`applyWrapperAllows(applyHookHandoffTrim(...))`, the order lockdownAction
// uses), the per-repo settings must carry the positive
// `Bash(coily ops gh:*)` allow while the bare `Bash(gh:*)` deny is gone.
// That combination is what stops the auto-mode classifier from reading
// `coily ops gh` as circumvention of the user-level `gh:*` deny.
func TestLockdownPipeline_GhWrapperSanctioned(t *testing.T) {
	base, err := lockdown.LoadDefaults()
	if err != nil {
		t.Fatalf("LoadDefaults: %v", err)
	}
	got := applyWrapperAllows(applyHookHandoffTrim(base))

	if !containsString(got.Allow, "Bash(coily ops gh:*)") {
		t.Errorf("expected explicit Bash(coily ops gh:*) allow in stamped defaults, got allows: %v", got.Allow)
	}
	if containsString(got.Deny, "Bash(gh:*)") {
		t.Errorf("expected bare Bash(gh:*) deny to be trimmed by the hook handoff, but it survived")
	}
}

// TestApplyHookHandoffTrim_PreservesUnwrappedDenies asserts that a bare
// deny for a binary with no coily wrapper (so no hook recovery route) is
// preserved: the deny is the only protection. The unwrapped token is
// chosen dynamically from defaults.yaml minus wrapperRecovery, so the test
// stays valid as the wrapped set grows (every passthrough is now wrapped +
// routed, coily#197, so the prior flyctl example no longer qualifies).
func TestApplyHookHandoffTrim_PreservesUnwrappedDenies(t *testing.T) {
	d, err := lockdown.LoadDefaults()
	if err != nil {
		t.Fatalf("LoadDefaults: %v", err)
	}
	got := applyHookHandoffTrim(d)

	// Find a bare single-token Bash deny whose token is not in
	// wrapperRecovery. That deny has no recovery route, so the trim must
	// leave it in place.
	var unwrapped string
	for _, dn := range d.Deny {
		tok := strings.TrimSuffix(strings.TrimPrefix(dn, "Bash("), ":*)")
		if tok == dn || strings.ContainsAny(tok, " (:") {
			continue // not a simple Bash(<tok>:*) deny
		}
		if _, wrapped := wrapperRecovery[tok]; !wrapped {
			unwrapped = dn
			break
		}
	}
	if unwrapped == "" {
		t.Skip("no unwrapped bare-binary deny in defaults.yaml to exercise")
	}
	if !containsString(got.Deny, unwrapped) {
		t.Errorf("expected unwrapped deny %q to survive trim, but it was dropped", unwrapped)
	}
}

// TestApplyHookHandoffTrim_PreservesShellInterpreterDenies asserts
// that the no-recovery family of denies (bash, sh, dash, fish, ksh,
// zsh, env, exec, the powershell family, force-push variants) is
// untouched. These are not in wrapperRecovery because no `coily X`
// wrapper exists for them - the deny is the only protection.
func TestApplyHookHandoffTrim_PreservesShellInterpreterDenies(t *testing.T) {
	d, err := lockdown.LoadDefaults()
	if err != nil {
		t.Fatalf("LoadDefaults: %v", err)
	}
	got := applyHookHandoffTrim(d)
	shellInterps := []string{
		"Bash(bash:*)", "Bash(sh:*)", "Bash(dash:*)", "Bash(fish:*)",
		"Bash(ksh:*)", "Bash(zsh:*)", "Bash(env:*)", "Bash(exec:*)",
	}
	for _, want := range shellInterps {
		if !containsString(d.Deny, want) {
			// Skip if cli-guard's defaults.yaml doesn't ship this one.
			continue
		}
		if !containsString(got.Deny, want) {
			t.Errorf("expected %q to survive trim (no coily wrapper exists), but it was dropped", want)
		}
	}
}

// TestCoilyRenderHookScript_IsCoilyHookShim asserts the rendered hook
// body is a one-line exec into `coily hook pre-tool-use`, NOT a cross-
// consumer reference like the legacy `exec <peer> hook ...` shim.
// Per coilysiren/coily#248 + cli-guard#74, coily and ward are
// peer cli-guard consumers; neither names the other in source.
func TestCoilyRenderHookScript_IsCoilyHookShim(t *testing.T) {
	body, err := coilyRenderHookScript(nil, nil)
	if err != nil {
		t.Fatalf("coilyRenderHookScript: %v", err)
	}
	if !strings.Contains(body, "exec coily hook pre-tool-use") {
		t.Errorf("expected hook body to exec into coily hook, got:\n%s", body)
	}
	if strings.Contains(body, "ward hook") {
		t.Errorf("hook body must not shim into the ward peer consumer (boundary violation, coily#248); got:\n%s", body)
	}
	if !strings.HasPrefix(body, "#!/bin/sh\n") {
		t.Errorf("expected POSIX-sh shebang, got first line: %q", strings.SplitN(body, "\n", 2)[0])
	}
	if strings.Contains(body, "case ") {
		t.Errorf("hook body still contains a case statement; the 159-line legacy script should be gone:\n%s", body)
	}
	const maxLines = 15
	if got := strings.Count(body, "\n"); got > maxLines {
		t.Errorf("expected the hook shim to be tiny (<= %d lines), got %d:\n%s", maxLines, got, body)
	}
}

// TestCoilyLockdownDriver_WiresHookHandoff asserts the driver factory
// has the coily-hook renderer wired, not a cross-consumer reference.
func TestCoilyLockdownDriver_WiresHookHandoff(t *testing.T) {
	drv := coilyLockdownDriver()
	if drv.RenderHookScript == nil {
		t.Fatalf("coilyLockdownDriver returned a driver with no RenderHookScript")
	}
	body, err := drv.RenderHookScript(nil, drv)
	if err != nil {
		t.Fatalf("RenderHookScript: %v", err)
	}
	if !strings.Contains(body, "exec coily hook pre-tool-use") {
		t.Errorf("RenderHookScript should exec into coily hook (coily#248); got:\n%s", body)
	}
	if strings.Contains(body, "ward hook") {
		t.Errorf("RenderHookScript must not shim into the ward peer consumer; got:\n%s", body)
	}
}

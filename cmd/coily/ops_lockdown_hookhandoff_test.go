package main

import (
	"fmt"
	"strings"
	"testing"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/lockdown"
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

// TestApplyHookHandoffTrim_DropsExplicitWrapperAllows asserts the
// counterweight allows added by applyWrapperAllows are also removed
// for trimmed denies. Once the bare deny is gone, the auto-mode
// classifier has nothing to flag, so the explicit `Bash(coily X:*)`
// allow loses its purpose (coilysiren/coily#159).
func TestApplyHookHandoffTrim_DropsExplicitWrapperAllows(t *testing.T) {
	d, err := lockdown.LoadDefaults()
	if err != nil {
		t.Fatalf("LoadDefaults: %v", err)
	}
	// Run through the same pipeline order lockdownOne uses, so the
	// explicit allows are added before trim has a chance to drop them.
	d = applyWrapperAllows(d)
	got := applyHookHandoffTrim(d)
	for token := range wrapperRecovery {
		bareDeny := fmt.Sprintf("Bash(%s:*)", token)
		wantedAllow, ok := wrapperAllows[bareDeny]
		if !ok {
			continue
		}
		if containsString(got.Allow, wantedAllow) {
			t.Errorf("expected %q to be trimmed from allow list (its deny was trimmed), but it survived", wantedAllow)
		}
	}
}

// TestApplyHookHandoffTrim_PreservesUnwrappedDenies asserts that bare
// denies for binaries with no coily hook route are preserved.
// flyctl is the canonical example today: coily wraps it via
// `coily ops flyctl`, but coily's hook route table does not yet
// route flyctl. The bare deny must stay so an unwrapped invocation
// still gets blocked.
func TestApplyHookHandoffTrim_PreservesUnwrappedDenies(t *testing.T) {
	d, err := lockdown.LoadDefaults()
	if err != nil {
		t.Fatalf("LoadDefaults: %v", err)
	}
	got := applyHookHandoffTrim(d)
	// flyctl is in wrapperAllows but not in wrapperRecovery, so the
	// trim should leave its bare deny alone.
	if _, isWrapped := wrapperRecovery["flyctl"]; isWrapped {
		t.Fatalf("test premise broken: wrapperRecovery now covers flyctl; pick another unwrapped binary")
	}
	if !containsString(got.Deny, "Bash(flyctl:*)") {
		// Only flag if flyctl was in the original deny list to begin
		// with. If cli-guard's defaults.yaml dropped it, that's a
		// different change.
		hadFlyctl := containsString(d.Deny, "Bash(flyctl:*)")
		if hadFlyctl {
			t.Errorf("expected Bash(flyctl:*) to survive trim, but it was dropped")
		}
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

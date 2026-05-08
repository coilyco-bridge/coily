package main

// Tests in this file walk SECURITY.md's load-bearing claims and assert each
// against runtime artifacts. The point is to catch prose-runtime drift: when
// a feature is trimmed and the prose forgets to follow, this test fails
// before the boundary description ships out of date.
//
// When SECURITY.md gains a testable claim, add a TestSecurityClaim_*
// function here. When a SECURITY.md claim gets deleted, delete the
// corresponding test. The two should move together.
//
// What "load-bearing" means here: a claim about runtime behavior whose
// failure would weaken the boundary or mislead an operator. Out of scope:
// rationale paragraphs, threat-model background, references to deny rules
// that live in another file.

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coilysiren/coily/pkg/config"
	"github.com/coilysiren/coily/pkg/lockdown"
	"github.com/coilysiren/coily/pkg/policy"
	"github.com/coilysiren/coily/pkg/scope"
	"github.com/urfave/cli/v3"
)

// newSecurityClaimRunner builds a Runner sufficient for command-tree
// walking. Loads the layered config (defaults + any host overlays) so verb
// builders that dereference r.Cfg do not panic. Audit and SSH stay nil;
// tests in this file do not invoke Actions.
func newSecurityClaimRunner(t *testing.T) *Runner {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	return &Runner{Cfg: cfg}
}

// TestSecurityClaim_PolicyRejectsAllShellMetacharacters covers the
// SECURITY.md claim that pkg/policy rejects shell metacharacters before they
// reach a subprocess. Walks every byte in the documented ShellMeta set so a
// loosened matcher fails here even if the matcher's own unit test missed it.
func TestSecurityClaim_PolicyRejectsAllShellMetacharacters(t *testing.T) {
	if policy.ShellMeta == "" {
		t.Fatal("policy.ShellMeta is empty; the metachar gate is wide open")
	}
	for _, b := range []byte(policy.ShellMeta) {
		probe := "x" + string(b) + "y"
		err := policy.ValidateArg("test", probe)
		if err == nil {
			t.Errorf("ValidateArg(%q) returned nil; ShellMeta byte %q (0x%02x) must be rejected",
				probe, b, b)
		}
		if !errors.Is(err, policy.ErrShellMeta) {
			t.Errorf("ValidateArg(%q) error = %v; want errors.Is(_, ErrShellMeta)", probe, err)
		}
	}
}

// TestSecurityClaim_NoEscapeHatchVerbs covers SECURITY.md's "No coily shell /
// coily run escape hatch, ever. Same rule applies to remote shells: no
// coily ssh exec, no coily ops kubectl exec pass-through."
//
// Walks the registered command tree built by the production Runner and fails
// if any forbidden name lands as a top-level verb or under ssh. (Kubectl is
// a passthrough that does not register subcommands in the tree; the deny
// list at pkg/lockdown/defaults.yaml covers `kubectl exec` separately and is
// asserted by TestSecurityClaim_LockdownDeniesKubectlExec.)
func TestSecurityClaim_NoEscapeHatchVerbs(t *testing.T) {
	r := newSecurityClaimRunner(t)
	cmds := r.builtInCommands()

	// "shell" and "run" must not appear as top-level coily verbs. They
	// would name an unrestricted-execution surface that defeats the
	// boundary.
	forbiddenTopLevel := map[string]bool{
		"shell": true,
		"run":   true,
		"exec":  true,
	}
	for _, c := range cmds {
		if forbiddenTopLevel[c.Name] {
			t.Errorf("forbidden top-level verb registered: coily %q (SECURITY.md: no escape hatch)", c.Name)
		}
	}

	// Under coily ssh, no "exec" subcommand. The named systemctl/git/copy/
	// deploy verbs are the supported ssh surface.
	for _, c := range cmds {
		if c.Name != "ssh" {
			continue
		}
		for _, sub := range c.Commands {
			if sub.Name == "exec" {
				t.Errorf("forbidden subcommand registered: coily ssh exec (SECURITY.md: named verbs only)")
			}
		}
	}
}

// TestSecurityClaim_NoConfirmationTokenVerb covers the SECURITY.md history
// note that the HMAC-token design was removed. A `coily auth issue` verb
// returning a fresh token would re-introduce the same self-authorization
// pattern that motivated the removal.
func TestSecurityClaim_NoConfirmationTokenVerb(t *testing.T) {
	r := newSecurityClaimRunner(t)
	for _, c := range r.builtInCommands() {
		if c.Name == "auth" {
			t.Errorf("coily auth verb registered; SECURITY.md says token ritual added no security")
			for _, sub := range c.Commands {
				t.Logf("  subcommand: %s", sub.Name)
			}
		}
	}
}

// TestSecurityClaim_CommitScopeOptOutRejected covers SECURITY.md's "every
// audit row binds to a real repo, no opt-out" guarantee. The rejected forms
// are "-", "none", "off"; an empty string is the unset signal handled
// elsewhere.
func TestSecurityClaim_CommitScopeOptOutRejected(t *testing.T) {
	// Resolve takes (flagValue, envFallback, cwd). Empty fallback + a real
	// cwd suffice; the opt-out check fires before any cwd lookup.
	cwd := t.TempDir()
	for _, val := range []string{"-", "none", "off", "NONE", "Off"} {
		_, err := scope.Resolve(val, "", cwd)
		if err == nil {
			t.Errorf("scope.Resolve(%q,...) returned nil err; SECURITY.md says no opt-out", val)
			continue
		}
		if !errors.Is(err, scope.ErrOptOutRejected) {
			t.Errorf("scope.Resolve(%q,...) err = %v; want errors.Is(_, ErrOptOutRejected)", val, err)
		}
	}
}

// TestSecurityClaim_VerbWrapIsTheChokepoint covers the architectural claim
// that every cli.Action goes through verb.Wrap. Today this is a convention
// rather than a structural enforcement (the test reads cli command Action
// pointers and counts how many distinct functions appear), so this test is
// best-effort: if it ever breaks, the fix is a code change to make every
// Action come from verb.Wrap, not a relaxation of the assertion.
//
// Implementation note: cli.ActionFunc values cannot be reliably compared by
// pointer because of closures, so we cannot assert "all Actions are the same
// function." Instead we walk the tree and verify that every leaf with an
// Action has a corresponding ArgsFunc-shaped wrap somewhere up the chain.
// This catches the most common bypass mode: a hand-built closure inlined at
// the verb registration site.
//
// For now, mark as t.Skip with a TODO so the test file documents the gap
// explicitly rather than silently lacking the check. When verb.Wrap exposes
// a marker (e.g. a wrapped *Spec attached to the command) this test can do
// real work.
func TestSecurityClaim_VerbWrapIsTheChokepoint(t *testing.T) {
	t.Skip("TODO: verb.Wrap does not yet stamp commands with a marker. " +
		"Until then, this test exists as a placeholder so the gap is visible.")
}

// TestSecurityClaim_NoDevModeBypassInProdBuilds is a build-tag check.
// SECURITY.md says production builds use -tags prod which compiles out
// any dev-mode conveniences. This test only runs the smoke path in unit
// tests; full coverage is the responsibility of the prod-tag build itself.
//
// Documented here as a placeholder so the prose claim is visible to anyone
// auditing what the test file covers.
func TestSecurityClaim_NoDevModeBypassInProdBuilds(t *testing.T) {
	t.Skip("Build-tag separation is enforced by the build system, not by a runtime test.")
}

// TestSecurityClaim_UserBinaryGateUnconditional covers the SECURITY.md
// claim that the user-level coily-binary-gate hook fires for every Bash
// PreToolUse event - including cron-spawned local-agent sessions.
//
// Two concrete checks back the claim:
//
//  1. The settings.json entry written by EnsureUserHook registers a
//     PreToolUse Bash matcher with no transcript_path / time-of-day /
//     other conditional skip. The companion UserPromptSubmit cron-bypass
//     fix (2026-05-08) must not have leaked into this entry.
//  2. The hook script body itself does not early-return on a
//     transcript_path or local-agent-mode marker. If a future change
//     adds such a skip, this test catches it.
//
// What this does NOT cover: whether Claude Code Desktop / cron-spawned
// agent sessions actually invoke the user-level hooks at all. That is
// runtime behavior of an external harness; the operational verification
// step is documented in issue #66 and lives outside the unit test.
// Issue #66.
func TestSecurityClaim_UserBinaryGateUnconditional(t *testing.T) {
	home := t.TempDir()
	hookPath, _, err := lockdown.EnsureUserHook(home)
	if err != nil {
		t.Fatalf("EnsureUserHook: %v", err)
	}

	// (1) settings.json entry: matcher = "Bash" with no conditional fields.
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatalf("parse settings: %v", err)
	}
	hooks, _ := root["hooks"].(map[string]any)
	preToolUse, _ := hooks["PreToolUse"].([]any)
	var ourEntry map[string]any
	for _, e := range preToolUse {
		m, ok := e.(map[string]any)
		if !ok {
			continue
		}
		inner, _ := m["hooks"].([]any)
		for _, h := range inner {
			hm, _ := h.(map[string]any)
			if marker, _ := hm["_coily"].(string); marker == lockdown.UserHookSettingsKey {
				ourEntry = m
				break
			}
		}
	}
	if ourEntry == nil {
		t.Fatalf("user-level coily-binary-gate hook not registered; got settings: %s", string(raw))
	}
	if matcher, _ := ourEntry["matcher"].(string); matcher != "Bash" {
		t.Errorf("hook matcher = %q, want %q (cron sessions need the same Bash gate)", matcher, "Bash")
	}
	// Any field beyond matcher/hooks would be a conditional that could
	// silently exclude cron-spawned sessions. The schema today is
	// {matcher, hooks} only; surface unexpected keys here so a future
	// schema-extension that adds (e.g.) a "when" clause has to be reviewed.
	for k := range ourEntry {
		if k != "matcher" && k != "hooks" {
			t.Errorf("user-level hook entry carries unexpected field %q; conditional fields could exclude cron sessions", k)
		}
	}

	// (2) Hook script body: no transcript_path / local-agent skip.
	body, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read hook: %v", err)
	}
	for _, marker := range []string{
		"transcript_path",
		"local-agent-mode-sessions",
		"late-ok",
	} {
		if strings.Contains(string(body), marker) {
			t.Errorf("hook script contains %q; the binary gate must fire unconditionally for cron sessions", marker)
		}
	}
}

// TestSecurityClaim_UserBinaryGateBlocksDevCoilyForCronStdin pins the
// runtime behavior of the user-level hook against synthetic stdin shaped
// like a cron-spawned session's PreToolUse event. The harness passes the
// transcript path in `transcript_path`; the hook ignores that field and
// still rejects a dev coily binary.
//
// Companion to TestSecurityClaim_UserBinaryGateUnconditional: the prior
// test asserts the hook is registered without a skip, this one asserts
// the hook actually rejects a forbidden invocation when the input shape
// looks cron-spawned. Issue #66.
func TestSecurityClaim_UserBinaryGateBlocksDevCoilyForCronStdin(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("/bin/sh not available")
	}
	home := t.TempDir()
	hookPath, _, err := lockdown.EnsureUserHook(home)
	if err != nil {
		t.Fatalf("EnsureUserHook: %v", err)
	}

	// Cron-style PreToolUse stdin: includes transcript_path under the
	// cron-spawned local-agent-mode-sessions tree. The hook must ignore
	// that field and reject the dev coily.
	stdin := `{"transcript_path":"/local-agent-mode-sessions/abc123/t.jsonl",` +
		`"tool_input":{"command":"/Users/someone/go/bin/coily version"}}`
	cmd := exec.Command("sh", hookPath) //nolint:gosec // hookPath is generated under t.TempDir
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("hook accepted dev coily for cron-style stdin; output: %s", out)
	}
	if !strings.Contains(string(out), "lockdown: blocked") {
		t.Errorf("hook output does not name lockdown block; got: %s", out)
	}
}

// recordedSubcommands is a debug helper used by t.Logf in failures so the
// failing test message includes what it actually saw.
//
//nolint:unused // kept so future TestSecurityClaim_* additions can use it
func recordedSubcommands(c *cli.Command) []string {
	out := []string{}
	for _, sub := range c.Commands {
		out = append(out, sub.Name)
		for _, leaf := range sub.Commands {
			out = append(out, sub.Name+" "+leaf.Name)
		}
	}
	return out
}

// TestSecurityClaim_LockdownDeniesBareKubectlAndAwsAndGh verifies the deny
// list shipped in pkg/lockdown/defaults.yaml covers bare invocation of
// kubectl, aws, and gh - the three privileged-op binaries that route
// through coily ops. The previous design enumerated read-verb allows and
// write-verb denies separately because Claude Code's Bash(prefix:*) syntax
// cannot pattern-match `aws * describe-*`; the current design inverts the
// allowlist and denies the bare binaries entirely so every call lands in
// the audit log. kubectl exec, kubectl run, and the rest are covered
// transitively by Bash(kubectl:*).
func TestSecurityClaim_LockdownDeniesBareKubectlAndAwsAndGh(t *testing.T) {
	// LoadDefaults parses pkg/lockdown/defaults.yaml (embedded). Asserting
	// against the parsed struct (rather than substring-matching the raw
	// file) means a typo in the file blows up here as a parse error rather
	// than as a silent miss.
	d, err := lockdown.LoadDefaults()
	if err != nil {
		t.Fatalf("lockdown.LoadDefaults: %v", err)
	}
	wantDenies := []string{
		"Bash(kubectl:*)",
		"Bash(aws:*)",
		"Bash(gh:*)",
	}
	denySet := map[string]bool{}
	for _, deny := range d.Deny {
		denySet[deny] = true
	}
	for _, want := range wantDenies {
		if !denySet[want] {
			t.Errorf("lockdown defaults missing %q; the inversion routes every call through coily ops", want)
		}
	}

	// Belt-and-suspenders: assert no enumerated read-verb allow leaked back
	// in. The whole point of the inversion is "no aws / kubectl / gh
	// allows," and a strayed `Bash(aws sts get-caller-identity:*)` would
	// silently re-open the ergonomics shortcut without re-opening the
	// design conversation.
	for _, allow := range d.Allow {
		for _, prefix := range []string{"Bash(aws ", "Bash(kubectl ", "Bash(gh "} {
			if strings.HasPrefix(allow, prefix) {
				t.Errorf("lockdown defaults allow %q; the inversion forbids enumerated %s reads", allow, prefix)
			}
		}
	}
}

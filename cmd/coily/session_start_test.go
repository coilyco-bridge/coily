package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestCapabilityIndex_CoversSurfaces proves the index, built from the live
// command tree, names the passthrough surface and the external-service
// wrappers, and leads with the presence!=auth doctrine.
func TestCapabilityIndex_CoversSurfaces(t *testing.T) {
	r := newTestRunner(t)
	idx := capabilityIndex(buildTree(r.builtInCommands(), nil))

	for _, want := range []string{
		capabilityDoctrine,
		"coily ops {aws gh kubectl",       // ops passthroughs
		"coily pkg {pnpm npm",             // pkg passthroughs
		"coily {docker tailscale}",        // top-level passthroughs
		"coily ops {modio discord sentry", // ops service wrappers
		"coily gaming {eco",               // gaming servers
		"coily --tree --json",             // depth pointer
	} {
		if !strings.Contains(idx, want) {
			t.Errorf("index missing %q\n--- index ---\n%s", want, idx)
		}
	}
}

// TestCapabilityIndex_ExcludesInternalGroups proves the service-wrapper line
// does not advertise coily-internal mechanics (dispatch registry, pkg
// directory wrappers) that would be session-open noise.
func TestCapabilityIndex_ExcludesInternalGroups(t *testing.T) {
	r := newTestRunner(t)
	idx := capabilityIndex(buildTree(r.builtInCommands(), nil))

	for _, unwanted := range []string{"registry", "glama", "skillsmp"} {
		// These appear only as non-passthrough subgroups outside serviceGroups;
		// the bin lines never carry them, so any occurrence is leakage.
		if strings.Contains(idx, unwanted) {
			t.Errorf("index leaked internal group %q\n%s", unwanted, idx)
		}
	}
}

// TestEmitCapabilityIndex_SessionStartContract proves the emitter writes a
// well-formed Claude Code SessionStart payload with the index as
// additionalContext and unescaped backticks.
func TestEmitCapabilityIndex_SessionStartContract(t *testing.T) {
	r := newTestRunner(t)
	root := buildTree(r.builtInCommands(), nil)

	var out sessionStartHookOutput
	out.HookSpecificOutput = sessionStartHookSpecific{
		HookEventName:     "SessionStart",
		AdditionalContext: capabilityIndex(root),
	}
	b, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var round sessionStartHookOutput
	if err := json.Unmarshal(b, &round); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if round.HookSpecificOutput.HookEventName != "SessionStart" {
		t.Errorf("hookEventName = %q, want SessionStart", round.HookSpecificOutput.HookEventName)
	}
	if !strings.Contains(round.HookSpecificOutput.AdditionalContext, capabilityDoctrine) {
		t.Errorf("additionalContext lost the doctrine line")
	}
}

// TestInjectSessionStartHook_AddsAndIsIdempotent proves the settings.json
// wrap adds the SessionStart hook once and never duplicates it on re-run,
// while leaving an existing PreToolUse block intact.
func TestInjectSessionStartHook_AddsAndIsIdempotent(t *testing.T) {
	base := []byte(`{
  "hooks": {
    "PreToolUse": [
      {"matcher": "Bash", "hooks": [{"type": "command", "command": ".claude/lockdown-deny.sh"}]}
    ]
  },
  "permissions": {"allow": [], "deny": []}
}`)

	once, err := injectSessionStartHook(base)
	if err != nil {
		t.Fatalf("inject: %v", err)
	}
	twice, err := injectSessionStartHook(once)
	if err != nil {
		t.Fatalf("inject again: %v", err)
	}
	if string(once) != string(twice) {
		t.Errorf("not idempotent:\nonce:\n%s\ntwice:\n%s", once, twice)
	}

	var got map[string]any
	if err := json.Unmarshal(twice, &got); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	hooks := got["hooks"].(map[string]any)
	ss, _ := hooks["SessionStart"].([]any)
	if len(ss) != 1 {
		t.Fatalf("SessionStart entries = %d, want 1", len(ss))
	}
	pre, _ := hooks["PreToolUse"].([]any)
	if len(pre) != 1 {
		t.Errorf("PreToolUse clobbered: %d entries", len(pre))
	}
	if !sessionStartHookPresent(ss) {
		t.Errorf("SessionStart entry does not carry the coily command")
	}
}

// TestInjectSessionStartHook_EmptyInput handles a nil/empty settings blob
// without panicking and produces a hooks block with the SessionStart entry.
func TestInjectSessionStartHook_EmptyInput(t *testing.T) {
	out, err := injectSessionStartHook(nil)
	if err != nil {
		t.Fatalf("inject nil: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("parse: %v", err)
	}
	hooks, ok := got["hooks"].(map[string]any)
	if !ok {
		t.Fatal("no hooks block")
	}
	if ss, _ := hooks["SessionStart"].([]any); !sessionStartHookPresent(ss) {
		t.Errorf("SessionStart hook not installed on empty input")
	}
}

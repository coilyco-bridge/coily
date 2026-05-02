package lockdown_test

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coilysiren/coily/pkg/lockdown"
)

func TestEnsureUserHook_FreshHome(t *testing.T) {
	home := t.TempDir()
	hookPath, changed, err := lockdown.EnsureUserHook(home)
	if err != nil {
		t.Fatalf("EnsureUserHook: %v", err)
	}
	if !changed {
		t.Error("expected settings change on fresh home, got changed=false")
	}
	wantHook := filepath.Join(home, ".claude", lockdown.UserHookFileName)
	if hookPath != wantHook {
		t.Errorf("hookPath = %q, want %q", hookPath, wantHook)
	}
	body, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read hook: %v", err)
	}
	if !strings.Contains(string(body), "Coily binary check") {
		t.Error("rendered hook missing coily binary check block")
	}
	settings := readJSON(t, filepath.Join(home, ".claude", "settings.json"))
	hooks := settings["hooks"].(map[string]any)
	pre := hooks["PreToolUse"].([]any)
	if len(pre) != 1 {
		t.Fatalf("PreToolUse len = %d, want 1", len(pre))
	}
}

func TestEnsureUserHook_Idempotent(t *testing.T) {
	home := t.TempDir()
	if _, _, err := lockdown.EnsureUserHook(home); err != nil {
		t.Fatalf("first run: %v", err)
	}
	_, changed, err := lockdown.EnsureUserHook(home)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if changed {
		t.Error("expected no settings change on second run, got changed=true")
	}
}

func TestEnsureUserHook_PreservesExistingFields(t *testing.T) {
	home := t.TempDir()
	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := map[string]any{
		"permissions": map[string]any{
			"allow": []any{"Bash(ls:*)"},
			"deny":  []any{"Bash(rm -rf:*)"},
		},
		"hooks": map[string]any{
			"UserPromptSubmit": []any{
				map[string]any{
					"hooks": []any{
						map[string]any{"type": "command", "command": "echo hi"},
					},
				},
			},
			"PreToolUse": []any{
				map[string]any{
					"matcher": "Bash",
					"hooks": []any{
						map[string]any{"type": "command", "command": "/path/to/user/hook.sh"},
					},
				},
			},
		},
		"enabledPlugins": map[string]any{"foo": true},
	}
	body, _ := json.MarshalIndent(existing, "", "  ")
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := lockdown.EnsureUserHook(home); err != nil {
		t.Fatalf("EnsureUserHook: %v", err)
	}
	got := readJSON(t, filepath.Join(claudeDir, "settings.json"))

	// permissions and enabledPlugins must survive verbatim.
	if got["permissions"] == nil || got["enabledPlugins"] == nil {
		t.Error("expected permissions and enabledPlugins preserved")
	}
	hooks := got["hooks"].(map[string]any)
	if hooks["UserPromptSubmit"] == nil {
		t.Error("UserPromptSubmit hook lost")
	}
	pre := hooks["PreToolUse"].([]any)
	if len(pre) != 1 {
		t.Fatalf("PreToolUse should still have 1 Bash matcher, got %d", len(pre))
	}
	inner := pre[0].(map[string]any)["hooks"].([]any)
	if len(inner) != 2 {
		t.Fatalf("inner hooks should have 2 entries (existing + coily), got %d", len(inner))
	}
}

func TestEnsureUserHook_MalformedJSON(t *testing.T) {
	home := t.TempDir()
	claudeDir := filepath.Join(home, ".claude")
	_ = os.MkdirAll(claudeDir, 0o755)
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := lockdown.EnsureUserHook(home); err == nil {
		t.Error("expected error on malformed settings.json, got nil")
	}
}

func TestEnsureUserHook_ScriptBlocksDevCoily(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh on PATH")
	}
	home := t.TempDir()
	hookPath, _, err := lockdown.EnsureUserHook(home)
	if err != nil {
		t.Fatalf("EnsureUserHook: %v", err)
	}
	cases := []struct {
		name   string
		stdin  string
		wantRC int
	}{
		{"~/go/bin/coily denied", `{"tool_input":{"command":"/Users/kai/go/bin/coily ssh"}}`, 2},
		{"/tmp/coily denied", `{"tool_input":{"command":"/tmp/coily ssh"}}`, 2},
		{"/opt/homebrew/bin/coily allowed", `{"tool_input":{"command":"/opt/homebrew/bin/coily ssh"}}`, 0},
		{"unrelated command allowed", `{"tool_input":{"command":"ls -la"}}`, 0},
		{"empty command allowed", `{"tool_input":{"command":""}}`, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command("sh", hookPath)
			cmd.Stdin = strings.NewReader(tc.stdin)
			err := cmd.Run()
			rc := 0
			if err != nil {
				var ee *exec.ExitError
				if errors.As(err, &ee) {
					rc = ee.ExitCode()
				} else {
					t.Fatalf("run hook: %v", err)
				}
			}
			if rc != tc.wantRC {
				t.Errorf("exit code = %d, want %d", rc, tc.wantRC)
			}
		})
	}
}

func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return m
}

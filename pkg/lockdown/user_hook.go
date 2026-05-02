package lockdown

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// UserHookSettingsKey is the unique identifier we stamp onto our
// PreToolUse Bash matcher entry in ~/.claude/settings.json. Used to
// recognize and update our own entry idempotently across runs without
// touching unrelated Bash hooks the user may have added themselves.
const UserHookSettingsKey = "coily-binary-gate"

// EnsureUserHook writes ~/.claude/coily-binary-gate.sh and patches
// ~/.claude/settings.json so a PreToolUse Bash matcher invokes it.
// Idempotent: re-runs overwrite the script and leave the settings entry
// alone if our marker is already present. Returns the resolved hook
// path and a flag for whether settings.json was changed.
//
// Preserves all other settings.json fields verbatim modulo JSON-Marshal
// key ordering (Go's encoding/json sorts map keys alphabetically). This
// is the only behavioral compromise; the file is hand-curated by the
// user but lives in a non-tracked location, so the cost is cosmetic.
func EnsureUserHook(homeDir string) (hookPath string, settingsChanged bool, err error) {
	if homeDir == "" {
		return "", false, errors.New("lockdown: EnsureUserHook: homeDir is empty")
	}
	claudeDir := filepath.Join(homeDir, ".claude")
	if mkErr := os.MkdirAll(claudeDir, 0o755); mkErr != nil {
		return "", false, fmt.Errorf("lockdown: mkdir %s: %w", claudeDir, mkErr)
	}
	hookPath = filepath.Join(claudeDir, UserHookFileName)
	if wErr := os.WriteFile(hookPath, []byte(RenderUserHookScript()), 0o755); wErr != nil {
		return "", false, fmt.Errorf("lockdown: write %s: %w", hookPath, wErr)
	}

	settingsPath := filepath.Join(claudeDir, "settings.json")
	changed, sErr := patchUserSettings(settingsPath, hookPath)
	if sErr != nil {
		return hookPath, false, sErr
	}
	return hookPath, changed, nil
}

// patchUserSettings reads ~/.claude/settings.json, ensures hooks.PreToolUse
// contains an entry whose hooks[].command matches hookPath and whose
// matcher is "Bash", and writes the file back if anything changed. A
// missing file is created with a minimal structure.
func patchUserSettings(settingsPath, hookPath string) (bool, error) {
	raw, readErr := os.ReadFile(settingsPath)
	root := map[string]any{}
	if readErr == nil {
		if uErr := json.Unmarshal(raw, &root); uErr != nil {
			return false, fmt.Errorf("lockdown: parse %s: %w", settingsPath, uErr)
		}
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return false, fmt.Errorf("lockdown: read %s: %w", settingsPath, readErr)
	}

	hooks, _ := root["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}
	preToolUse, _ := hooks["PreToolUse"].([]any)
	preToolUse = ensureCoilyHookEntry(preToolUse, hookPath)
	hooks["PreToolUse"] = preToolUse
	root["hooks"] = hooks

	after, mErr := json.MarshalIndent(root, "", "  ")
	if mErr != nil {
		return false, fmt.Errorf("lockdown: marshal settings: %w", mErr)
	}
	after = append(after, '\n')
	if len(raw) > 0 && string(raw) == string(after) {
		return false, nil
	}
	if wErr := os.WriteFile(settingsPath, after, 0o600); wErr != nil {
		return false, fmt.Errorf("lockdown: write %s: %w", settingsPath, wErr)
	}
	return true, nil
}

// ensureCoilyHookEntry returns preToolUse with a guaranteed entry whose
// inner hooks slice contains a {type: "command", command: hookPath,
// _coily: UserHookSettingsKey} record under matcher "Bash". An existing
// entry is identified by the _coily marker and updated in place; other
// entries (user-added Bash hooks) are preserved verbatim.
func ensureCoilyHookEntry(preToolUse []any, hookPath string) []any {
	wantHook := map[string]any{
		"type":    "command",
		"command": hookPath,
		"_coily":  UserHookSettingsKey,
	}
	for _, entry := range preToolUse {
		m, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		if matcher, _ := m["matcher"].(string); matcher != "Bash" {
			continue
		}
		inner, _ := m["hooks"].([]any)
		updated := false
		for i, h := range inner {
			hm, ok := h.(map[string]any)
			if !ok {
				continue
			}
			if marker, _ := hm["_coily"].(string); marker == UserHookSettingsKey {
				inner[i] = wantHook
				updated = true
				break
			}
		}
		if !updated {
			inner = append(inner, wantHook)
		}
		m["hooks"] = inner
		return preToolUse
	}
	// No Bash matcher entry exists; add one with our hook.
	return append(preToolUse, map[string]any{
		"matcher": "Bash",
		"hooks":   []any{wantHook},
	})
}

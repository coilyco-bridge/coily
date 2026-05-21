package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// claudeConfigPathOverride lets tests redirect ~/.claude.json to a
// tempfile so they don't mutate the operator's real config. Empty in
// production; read by claudeConfigPath.
var claudeConfigPathOverride = ""

// claudeConfigPath resolves the Claude Code config file that holds
// per-folder trust state (~/.claude.json).
func claudeConfigPath() (string, error) {
	if claudeConfigPathOverride != "" {
		return claudeConfigPathOverride, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude.json"), nil
}

// ensureClaudeFolderTrust marks dir as trusted in ~/.claude.json so the
// dispatched Claude session opens straight into work instead of stalling
// on the "Do you trust the files in this folder?" prompt.
//
// Claude Code keys folder trust per exact absolute path under
// projects.<path>.hasTrustDialogAccepted, and only flips that flag when a
// folder is opened interactively and the dialog accepted. A dispatch
// worktree is a brand-new path coily just created, and even the bare
// checkout may never have been opened interactively, so the operator hits
// the prompt on dispatch. coily set up the target, so coily can vouch for
// it (coilysiren/coily#290).
//
// Soft by contract: failures are returned for the caller to log, but the
// caller must not abort the dispatch over one - a stray trust prompt is a
// papercut, not a reason to drop the work. A missing config file is not a
// failure: Claude Code has not run yet and will write its own, so this
// returns nil and the caller stays quiet.
func ensureClaudeFolderTrust(dir string) error {
	path, err := claudeConfigPath()
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	if doc == nil {
		doc = map[string]any{}
	}
	projects, ok := doc["projects"].(map[string]any)
	if !ok {
		projects = map[string]any{}
		doc["projects"] = projects
	}
	entry, ok := projects[dir].(map[string]any)
	if !ok {
		entry = map[string]any{}
		projects[dir] = entry
	}
	if trusted, ok := entry["hasTrustDialogAccepted"].(bool); ok && trusted {
		return nil // already trusted - skip the rewrite, avoid a needless race
	}
	entry["hasTrustDialogAccepted"] = true
	return writeClaudeConfig(path, doc)
}

// writeClaudeConfig serializes doc back to path via a tmp-write + rename
// in the same directory, so a reader - Claude Code itself, or a concurrent
// dispatch - never observes a half-written file. Last writer wins on the
// rare concurrent-dispatch race; the cost of a dropped update is one trust
// prompt, never a corrupt config.
func writeClaudeConfig(path string, doc map[string]any) error {
	payload, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal claude config: %w", err)
	}
	tmp := path + ".coily-tmp"
	if err := os.WriteFile(tmp, payload, 0o600); err != nil {
		return fmt.Errorf("write tmp claude config: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename tmp claude config: %w", err)
	}
	return nil
}

// Package lockdown writes a per-repo Claude Code settings file that enforces
// the coily allowlist inversion. Defaults are embedded at build time.
//
// The shape of the output is compatible with Claude Code's settings.json:
//
//	{
//	  "permissions": { "allow": [...], "deny": [...] },
//	  "deniedMcpServers": [...]
//	}
//
// When writing to an existing file, allow/deny/deniedMcpServers entries are
// unioned with what's already there (duplicates removed). Other top-level
// keys in the existing file are preserved.
package lockdown

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"gopkg.in/yaml.v3"
)

//go:embed defaults.yaml
var defaultsYAML []byte

type Defaults struct {
	Allow            []string `yaml:"allow" json:"-"`
	Deny             []string `yaml:"deny" json:"-"`
	DeniedMcpServers []string `yaml:"deniedMcpServers" json:"-"`
}

// Settings is the subset of Claude Code settings we manipulate. Other keys
// in an existing settings file are preserved via rawSettings below.
type Settings struct {
	Permissions      Permissions `json:"permissions"`
	DeniedMcpServers []string    `json:"deniedMcpServers,omitempty"`
}

type Permissions struct {
	Allow []string `json:"allow,omitempty"`
	Deny  []string `json:"deny,omitempty"`
}

// LoadDefaults parses the embedded canonical allow/deny lists.
func LoadDefaults() (*Defaults, error) {
	var d Defaults
	if err := yaml.Unmarshal(defaultsYAML, &d); err != nil {
		return nil, fmt.Errorf("lockdown: parse embedded defaults: %w", err)
	}
	return &d, nil
}

// Plan describes what lockdown would (or did) write. Rendered as JSON for the
// caller to display or persist.
type Plan struct {
	TargetPath string          // the .claude/settings*.json path
	Existed    bool            // did TargetPath exist before?
	Before     json.RawMessage // original file contents, if any
	After      json.RawMessage // file contents that would be (or were) written
}

// BuildPlan computes what the target settings file should look like after
// applying the defaults. Does not touch disk.
func BuildPlan(targetPath string, d *Defaults, replace bool) (*Plan, error) {
	plan := &Plan{TargetPath: targetPath}

	// Load existing settings (if any).
	var existing map[string]any
	raw, err := os.ReadFile(targetPath)
	switch {
	case err == nil:
		plan.Existed = true
		plan.Before = append(json.RawMessage(nil), raw...)
		if err := json.Unmarshal(raw, &existing); err != nil {
			return nil, fmt.Errorf("lockdown: parse existing %s: %w", targetPath, err)
		}
	case os.IsNotExist(err):
		existing = map[string]any{}
	default:
		return nil, fmt.Errorf("lockdown: read %s: %w", targetPath, err)
	}

	if replace {
		existing = map[string]any{}
	}

	// Extract + merge permissions.
	allow, deny := extractPermissions(existing)
	allow = uniqueSorted(append(allow, d.Allow...))
	deny = uniqueSorted(append(deny, d.Deny...))
	existing["permissions"] = map[string]any{
		"allow": allow,
		"deny":  deny,
	}

	// Merge deniedMcpServers.
	mcp := extractStringSlice(existing, "deniedMcpServers")
	mcp = uniqueSorted(append(mcp, d.DeniedMcpServers...))
	if len(mcp) > 0 {
		existing["deniedMcpServers"] = mcp
	}

	out, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("lockdown: marshal: %w", err)
	}
	plan.After = out
	return plan, nil
}

// Write applies the plan to disk. Caller should have shown the plan first
// and confirmed.
func Write(plan *Plan) error {
	if err := os.MkdirAll(filepath.Dir(plan.TargetPath), 0o750); err != nil {
		return fmt.Errorf("lockdown: mkdir: %w", err)
	}
	return os.WriteFile(plan.TargetPath, plan.After, 0o600)
}

// TargetPath returns the settings file path under dir. If local is true,
// uses settings.local.json. Otherwise settings.json.
func TargetPath(dir string, local bool) string {
	name := "settings.json"
	if local {
		name = "settings.local.json"
	}
	return filepath.Join(dir, ".claude", name)
}

func extractPermissions(m map[string]any) (allow, deny []string) {
	p, ok := m["permissions"].(map[string]any)
	if !ok {
		return nil, nil
	}
	return extractStringSlice(p, "allow"), extractStringSlice(p, "deny")
}

func extractStringSlice(m map[string]any, key string) []string {
	v, ok := m[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(v))
	for _, x := range v {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func uniqueSorted(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	// Return nil for empty so JSON marshals as absent rather than [].
	if len(out) == 0 {
		return nil
	}
	return slices.Clip(out)
}

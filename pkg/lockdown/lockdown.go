// Package lockdown writes a per-repo Claude Code settings file that enforces
// the coily allowlist inversion. Defaults are embedded at build time.
//
// The shape of the output is compatible with Claude Code's settings.json:
//
//	{
//	  "permissions": { "allow": [...], "deny": [...] }
//	}
//
// MCP server allowlisting is intentionally out of scope. The Bash deny list
// gates shell-level blast radius (cluster mutations, secret reads, package
// installs). MCP-server gating is a different threat model - "is this MCP
// server trustworthy" - and the answer is per-user / per-machine, not
// per-repo. Baking it into a repo-scoped settings.json puts the decision in
// the wrong place. Drop it; let the user manage MCP allowlisting at the
// user-settings level.
//
// Behavior model (per docs/unresolved/13-lockdown-token.md, resolved):
//
//   - Bare `coily lockdown` prints the plan and exits. No write, no token.
//   - `coily lockdown --apply` writes a fresh file only if .claude/settings.json
//     does not already exist. It refuses an existing file. No token.
//   - `coily lockdown --apply --replace` overwrites an existing file. Mutating,
//     token required.
//
// The previous silent-merge behavior (union with existing allow/deny entries)
// is gone. The CLI either bootstraps a fresh file or replaces an existing one
// wholesale. BuildPlan always returns the canonical defaults, regardless of
// what is on disk. Any custom allow/deny entries the user added by hand are
// dropped on --replace.
package lockdown

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// validateShellSyntax pipes the script through `sh -n`, which parses
// without executing. Used to guard hook generation: a malformed script
// would silently neutralize the Desktop deny gate.
func validateShellSyntax(body string) error {
	cmd := exec.Command("sh", "-n")
	cmd.Stdin = strings.NewReader(body)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(out))
	}
	return nil
}

//go:embed defaults.yaml
var defaultsYAML []byte

type Defaults struct {
	Allow []string `yaml:"allow" json:"-"`
	Deny  []string `yaml:"deny" json:"-"`
}

// Settings is the subset of Claude Code settings we manipulate. Other keys
// in an existing settings file are preserved via rawSettings below.
type Settings struct {
	Permissions Permissions `json:"permissions"`
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
//
// The plan's After is always the canonical defaults rendered as JSON,
// independent of whatever is already on disk. The merge behavior that used
// to live here is gone: the CLI either bootstraps a fresh file (refusing if
// one exists) or overwrites with --replace. Existed and Before are still
// populated so callers can show a diff.
func BuildPlan(targetPath string, d *Defaults) (*Plan, error) {
	plan := &Plan{TargetPath: targetPath}

	raw, err := os.ReadFile(targetPath)
	switch {
	case err == nil:
		plan.Existed = true
		plan.Before = append(json.RawMessage(nil), raw...)
	case os.IsNotExist(err):
		// Nothing to load. Fresh bootstrap.
	default:
		return nil, fmt.Errorf("lockdown: read %s: %w", targetPath, err)
	}

	out := map[string]any{
		"permissions": map[string]any{
			"allow": uniqueSorted(append([]string(nil), d.Allow...)),
			"deny":  uniqueSorted(append([]string(nil), d.Deny...)),
		},
		// PreToolUse Bash hook is the Desktop-side enforcement path; the
		// built-in deny matcher is silently bypassed there. On the CLI the
		// matcher is the primary gate and this hook is defense-in-depth.
		// Two independent enforcement layers, neither depends on the other.
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					"matcher": "Bash",
					"hooks": []any{
						map[string]any{
							"type":    "command",
							"command": HookSettingsPath,
						},
					},
				},
			},
		},
	}

	encoded, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("lockdown: marshal: %w", err)
	}
	plan.After = encoded
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

// HookPath returns the absolute path of the generated PreToolUse hook
// script. It sits next to settings.json under .claude/.
func HookPath(settingsPath string) string {
	return filepath.Join(filepath.Dir(settingsPath), HookFileName)
}

// WriteHook renders and writes the PreToolUse hook script with 0755 perms.
// Validates the generated script with `sh -n` before writing - a syntax
// error would silently neutralize the deny gate on Desktop, so fail loud.
// Returns whether the file existed before the write so callers can report
// "created" vs "replaced".
func WriteHook(settingsPath string, d *Defaults) (string, bool, error) {
	body, err := RenderHookScript(d)
	if err != nil {
		return "", false, err
	}
	if err := validateShellSyntax(body); err != nil {
		return "", false, fmt.Errorf("lockdown: generated hook failed sh -n: %w", err)
	}
	hookPath := HookPath(settingsPath)
	existed := false
	if _, err := os.Stat(hookPath); err == nil {
		existed = true
	}
	if err := os.MkdirAll(filepath.Dir(hookPath), 0o750); err != nil {
		return "", false, fmt.Errorf("lockdown: mkdir hook: %w", err)
	}
	if err := os.WriteFile(hookPath, []byte(body), 0o755); err != nil {
		return "", false, fmt.Errorf("lockdown: write hook: %w", err)
	}
	return hookPath, existed, nil
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

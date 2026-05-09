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

// Settings is the subset of Claude Code settings we manipulate directly.
// In an existing settings file, BuildPlan replaces permissions wholesale
// and replaces the hooks.PreToolUse Bash matcher entry, but leaves every
// other top-level key (and every other PreToolUse matcher / hook event)
// untouched.
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
// Fresh-file path (no existing file): the After contains only the canonical
// permissions and hooks keys.
//
// Existing-file path: the After preserves every top-level key from the
// existing file, replaces permissions wholesale with the canonical allow +
// deny lists, and under hooks.PreToolUse swaps in (or appends) the
// canonical Bash matcher entry while leaving any other PreToolUse matchers
// and any other hook events (PostToolUse, SessionStart, ...) untouched.
//
// An existing file that does not parse as JSON is a hard error: BuildPlan
// cannot safely merge into an opaque blob. Operators recover by deleting
// the file (which becomes a fresh bootstrap).
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

	canonicalPerms := map[string]any{
		"allow": uniqueSorted(append([]string(nil), d.Allow...)),
		"deny":  uniqueSorted(append([]string(nil), d.Deny...)),
	}
	// PreToolUse Bash hook is the Desktop-side enforcement path; the
	// built-in deny matcher is silently bypassed there. On the CLI the
	// matcher is the primary gate and this hook is defense-in-depth.
	// Two independent enforcement layers, neither depends on the other.
	canonicalBashHook := map[string]any{
		"matcher": "Bash",
		"hooks": []any{
			map[string]any{
				"type":    "command",
				"command": HookSettingsPath,
			},
		},
	}

	var out map[string]any
	if plan.Existed && len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, fmt.Errorf("lockdown: parse existing %s: %w", targetPath, err)
		}
	}
	if out == nil {
		out = map[string]any{}
	}
	out["permissions"] = canonicalPerms
	out["hooks"] = mergeBashHook(out["hooks"], canonicalBashHook)

	encoded, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("lockdown: marshal: %w", err)
	}
	plan.After = encoded
	return plan, nil
}

// mergeBashHook returns a hooks-shaped map with the canonical Bash matcher
// entry installed under PreToolUse. Other PreToolUse matchers are
// preserved in place; other top-level hook events (PostToolUse,
// SessionStart, etc.) carry through untouched.
//
// If the existing PreToolUse already has an entry whose matcher is "Bash",
// that entry is replaced in place so the slice ordering of unrelated
// matchers is stable. Otherwise the canonical Bash entry is appended.
//
// If the existing hooks value is the wrong shape (not a map, or PreToolUse
// is not a slice), the malformed slot is replaced wholesale with a fresh
// PreToolUse containing only the canonical Bash entry. The rest of the
// hooks map (other top-level events) is preserved when possible.
func mergeBashHook(existing any, canonicalBash map[string]any) map[string]any {
	out := map[string]any{}
	if m, ok := existing.(map[string]any); ok {
		for k, v := range m {
			out[k] = v
		}
	}
	pre, ok := out["PreToolUse"].([]any)
	if !ok {
		out["PreToolUse"] = []any{canonicalBash}
		return out
	}
	for i, entry := range pre {
		em, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		if matcher, _ := em["matcher"].(string); matcher == "Bash" {
			pre[i] = canonicalBash
			out["PreToolUse"] = pre
			return out
		}
	}
	out["PreToolUse"] = append(pre, canonicalBash)
	return out
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

// MergeDenyInto reasserts the canonical deny list at an ancestor settings
// file - the recursion-root case where a parent directory's
// settings.local.json carries broad allows that would otherwise shadow
// per-repo deny rules below it. Claude Code applies deny ahead of allow
// within a single file, so injecting the canonical deny list into the
// ancestor file neutralizes the shadowing without touching the user's
// existing allow rules.
//
// If the file does not exist, it is created with just the deny list and
// no allow list. If it does exist, its existing top-level keys are
// preserved and `permissions.deny` becomes the union of existing deny
// entries and d.Deny. `permissions.allow` is left untouched. Returns
// (mutated, error) where mutated is true iff the file's effective
// content changed.
func MergeDenyInto(targetPath string, d *Defaults) (bool, error) {
	root := map[string]any{}
	existed := false
	if raw, err := os.ReadFile(targetPath); err == nil {
		existed = true
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &root); err != nil {
				return false, fmt.Errorf("lockdown: parse %s: %w", targetPath, err)
			}
		}
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("lockdown: read %s: %w", targetPath, err)
	}

	perms, _ := root["permissions"].(map[string]any)
	if perms == nil {
		perms = map[string]any{}
	}

	existingDeny := toStringSliceAny(perms["deny"])
	merged := uniqueSorted(append(append([]string(nil), existingDeny...), d.Deny...))

	// uniqueSorted returns nil for empty; never relevant here since d.Deny is
	// non-empty by construction, but keep the type stable as []string for the
	// equality check below.
	if merged == nil {
		merged = []string{}
	}

	if existed && stringSliceEqual(existingDeny, merged) {
		return false, nil
	}

	perms["deny"] = merged
	root["permissions"] = perms

	encoded, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return false, fmt.Errorf("lockdown: marshal %s: %w", targetPath, err)
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o750); err != nil {
		return false, fmt.Errorf("lockdown: mkdir %s: %w", filepath.Dir(targetPath), err)
	}
	if err := os.WriteFile(targetPath, encoded, 0o600); err != nil {
		return false, fmt.Errorf("lockdown: write %s: %w", targetPath, err)
	}
	return true, nil
}

func toStringSliceAny(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, x := range arr {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	as := append([]string(nil), a...)
	bs := append([]string(nil), b...)
	sort.Strings(as)
	sort.Strings(bs)
	for i := range as {
		if as[i] != bs[i] {
			return false
		}
	}
	return true
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

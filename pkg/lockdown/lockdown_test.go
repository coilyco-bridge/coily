package lockdown_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coilysiren/coily/pkg/lockdown"
)

func TestLoadDefaults_ReturnsNonEmpty(t *testing.T) {
	d, err := lockdown.LoadDefaults()
	if err != nil {
		t.Fatalf("LoadDefaults: %v", err)
	}
	if len(d.Allow) == 0 {
		t.Error("allow list is empty")
	}
	if len(d.Deny) == 0 {
		t.Error("deny list is empty")
	}
	if len(d.DeniedMcpServers) == 0 {
		t.Error("deniedMcpServers is empty")
	}
}

func TestLoadDefaults_AllowsCoilyBash(t *testing.T) {
	d, _ := lockdown.LoadDefaults()
	if !contains(d.Allow, "Bash(coily:*)") {
		t.Errorf("allow list missing Bash(coily:*). Got: %v", d.Allow)
	}
}

func TestLoadDefaults_DeniesDangerousBase(t *testing.T) {
	d, _ := lockdown.LoadDefaults()
	mustDeny := []string{
		"Bash(python:*)", "Bash(bash:*)", "Bash(aws:*)", "Bash(gh:*)",
		"Bash(kubectl apply:*)", "Bash(kubectl delete:*)",
	}
	for _, rule := range mustDeny {
		if !contains(d.Deny, rule) {
			t.Errorf("deny list missing required rule %q", rule)
		}
	}
}

func TestLoadDefaults_DeniesAwsEksMcp(t *testing.T) {
	d, _ := lockdown.LoadDefaults()
	if !contains(d.DeniedMcpServers, "aws-eks") {
		t.Errorf("deniedMcpServers missing aws-eks. Got: %v", d.DeniedMcpServers)
	}
}

func TestBuildPlan_NewFileGetsFullDefaults(t *testing.T) {
	d, _ := lockdown.LoadDefaults()
	target := filepath.Join(t.TempDir(), ".claude", "settings.json")
	plan, err := lockdown.BuildPlan(target, d)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if plan.Existed {
		t.Error("plan.Existed is true for a new target")
	}
	var after map[string]any
	if err := json.Unmarshal(plan.After, &after); err != nil {
		t.Fatalf("unmarshal After: %v", err)
	}
	perms := after["permissions"].(map[string]any)
	allow := toStringSlice(perms["allow"])
	if !contains(allow, "Bash(coily:*)") {
		t.Errorf("allow missing Bash(coily:*)")
	}
}

func TestBuildPlan_ExistingFileReportsExistedWithoutMerging(t *testing.T) {
	d, _ := lockdown.LoadDefaults()
	dir := t.TempDir()
	target := filepath.Join(dir, "settings.json")
	existing := map[string]any{
		"permissions": map[string]any{
			"allow": []any{"Bash(custom-tool:*)"},
			"deny":  []any{"Bash(npm run dangerous:*)"},
		},
		"someOtherKey": "preserved",
	}
	raw, _ := json.Marshal(existing)
	if err := os.WriteFile(target, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := lockdown.BuildPlan(target, d)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if !plan.Existed {
		t.Error("plan.Existed is false for an existing target")
	}
	var after map[string]any
	_ = json.Unmarshal(plan.After, &after)
	allow := toStringSlice(after["permissions"].(map[string]any)["allow"])
	if contains(allow, "Bash(custom-tool:*)") {
		t.Error("merge happened: custom allow entry leaked into After (silent merge is gone)")
	}
	if _, ok := after["someOtherKey"]; ok {
		t.Error("merge happened: unrelated top-level key leaked into After")
	}
	if !contains(allow, "Bash(coily:*)") {
		t.Error("default allow entry is missing")
	}
}

func TestBuildPlan_ExistingFileWithBadJSONIsAccepted(t *testing.T) {
	// BuildPlan no longer parses the existing file, only reads it for the
	// Before diff. Bad JSON is no longer fatal here. The CLI's --apply path
	// is what refuses to clobber.
	d, _ := lockdown.LoadDefaults()
	target := filepath.Join(t.TempDir(), "settings.json")
	_ = os.WriteFile(target, []byte("this is not json"), 0o600)
	plan, err := lockdown.BuildPlan(target, d)
	if err != nil {
		t.Errorf("BuildPlan errored on opaque existing file: %v", err)
	}
	if !plan.Existed {
		t.Error("plan.Existed should be true")
	}
}

func TestWrite_WritesWithTightPerms(t *testing.T) {
	d, _ := lockdown.LoadDefaults()
	target := filepath.Join(t.TempDir(), ".claude", "settings.json")
	plan, _ := lockdown.BuildPlan(target, d)
	if err := lockdown.Write(plan); err != nil {
		t.Fatalf("Write: %v", err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file perm = %o, want 0600", perm)
	}
}

func TestTargetPath_LocalToggle(t *testing.T) {
	if got := lockdown.TargetPath("/tmp/a", false); !strings.HasSuffix(got, "/settings.json") {
		t.Errorf("TargetPath(false) = %q", got)
	}
	if got := lockdown.TargetPath("/tmp/a", true); !strings.HasSuffix(got, "/settings.local.json") {
		t.Errorf("TargetPath(true) = %q", got)
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func toStringSlice(v any) []string {
	out := []string{}
	if arr, ok := v.([]any); ok {
		for _, x := range arr {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
	}
	return out
}

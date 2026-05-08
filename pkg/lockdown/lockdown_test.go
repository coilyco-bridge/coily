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
		"Bash(python:*)", "Bash(bash:*)",
		// aws/kubectl/gh are denied wholesale: every call routes through
		// coily ops <bin>, which is the audit + argv-validation chokepoint.
		// The previous design enumerated read-verb allows + write-verb
		// denies, which only existed because Claude Code's prefix-only
		// permission syntax could not match `aws * describe-*` generically.
		"Bash(aws:*)", "Bash(kubectl:*)", "Bash(gh:*)",
	}
	for _, rule := range mustDeny {
		if !contains(d.Deny, rule) {
			t.Errorf("deny list missing required rule %q", rule)
		}
	}
}

func TestLoadDefaults_DeniesWindowsExecution(t *testing.T) {
	d, _ := lockdown.LoadDefaults()
	mustDeny := []string{
		// Windows shells via Bash.
		"Bash(cmd:*)", "Bash(cmd.exe:*)",
		"Bash(powershell:*)", "Bash(powershell.exe:*)",
		"Bash(pwsh:*)", "Bash(pwsh.exe:*)",
		// Scripting hosts and LOLBAS binaries via Bash.
		"Bash(wscript:*)", "Bash(cscript:*)", "Bash(mshta:*)",
		"Bash(rundll32:*)", "Bash(regsvr32:*)",
		// The PowerShell tool itself (separate from Bash).
		"PowerShell", "PowerShell(*)",
	}
	for _, rule := range mustDeny {
		if !contains(d.Deny, rule) {
			t.Errorf("deny list missing required Windows rule %q", rule)
		}
	}
}

func TestBuildPlan_OmitsDeniedMcpServersKey(t *testing.T) {
	// MCP-server gating is deliberately not lockdown's job. Output JSON must
	// not carry a deniedMcpServers key.
	d, _ := lockdown.LoadDefaults()
	target := filepath.Join(t.TempDir(), ".claude", "settings.json")
	plan, err := lockdown.BuildPlan(target, d)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	var after map[string]any
	if err := json.Unmarshal(plan.After, &after); err != nil {
		t.Fatalf("unmarshal After: %v", err)
	}
	if _, ok := after["deniedMcpServers"]; ok {
		t.Errorf("After contains deniedMcpServers; want it absent. After=%s", string(plan.After))
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

func TestRenderHookScript_PassesShellSyntaxCheck(t *testing.T) {
	d, _ := lockdown.LoadDefaults()
	body, err := lockdown.RenderHookScript(d)
	if err != nil {
		t.Fatalf("RenderHookScript: %v", err)
	}
	if !strings.Contains(body, "#!/bin/sh") {
		t.Error("hook script missing /bin/sh shebang")
	}
	// Must mention at least one well-known deny prefix.
	for _, want := range []string{"aws", "kubectl", "docker", "ssh"} {
		if !strings.Contains(body, want) {
			t.Errorf("hook script missing deny prefix %q", want)
		}
	}
}

func TestRenderHookScript_NamesCoilyWrapperOnDeny(t *testing.T) {
	d, _ := lockdown.LoadDefaults()
	body, err := lockdown.RenderHookScript(d)
	if err != nil {
		t.Fatalf("RenderHookScript: %v", err)
	}
	// Issue #61: deny-rule message must name `coily ops <bin>` as the
	// recovery path for the wrapped binaries the agent reaches for most.
	for prefix, recovery := range map[string]string{
		"gh":      "coily ops gh",
		"aws":     "coily ops aws",
		"kubectl": "coily ops kubectl",
	} {
		want := "blocked by deny rule: " + prefix + ". Recovery: use `" + recovery
		if !strings.Contains(body, want) {
			t.Errorf("hook script for %q missing recovery hint %q", prefix, want)
		}
	}
}

func TestWriteHook_Executable(t *testing.T) {
	d, _ := lockdown.LoadDefaults()
	target := filepath.Join(t.TempDir(), ".claude", "settings.json")
	plan, _ := lockdown.BuildPlan(target, d)
	if err := lockdown.Write(plan); err != nil {
		t.Fatalf("Write: %v", err)
	}
	hookPath, _, err := lockdown.WriteHook(plan.TargetPath, d)
	if err != nil {
		t.Fatalf("WriteHook: %v", err)
	}
	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatalf("stat hook: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o755 {
		t.Errorf("hook perm = %o, want 0755", perm)
	}
}

func TestWriteHook_BlocksDeniedCommand(t *testing.T) {
	// End-to-end: render the hook, write it, invoke it with a synthetic
	// PreToolUse JSON for a denied command, expect exit 2 + stderr message.
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh on PATH")
	}
	d, _ := lockdown.LoadDefaults()
	target := filepath.Join(t.TempDir(), ".claude", "settings.json")
	if err := lockdown.Write(must(lockdown.BuildPlan(target, d))); err != nil {
		t.Fatalf("Write: %v", err)
	}
	hookPath, _, err := lockdown.WriteHook(target, d)
	if err != nil {
		t.Fatalf("WriteHook: %v", err)
	}

	cases := []struct {
		name   string
		stdin  string
		wantRC int
	}{
		{"aws s3 cp denied", `{"tool_input":{"command":"aws s3 cp foo s3://b/x"}}`, 2},
		{"aws ssm get-parameter denied", `{"tool_input":{"command":"aws ssm get-parameter --name /foo"}}`, 2},
		{"kubectl apply denied", `{"tool_input":{"command":"kubectl apply -f x.yaml"}}`, 2},
		{"piped aws s3 cp denied", `{"tool_input":{"command":"echo hi | aws s3 cp - s3://b/x"}}`, 2},
		{"env-prefixed aws s3 cp denied", `{"tool_input":{"command":"env AWS_PROFILE=x aws s3 cp foo s3://b/x"}}`, 2},
		{"gh pr merge denied", `{"tool_input":{"command":"gh pr merge 123"}}`, 2},
		{"gh api denied", `{"tool_input":{"command":"gh api repos/foo/bar"}}`, 2},
		// Inverted reads: bare aws/kubectl/gh now route through coily.
		{"aws s3 ls denied (route via coily)", `{"tool_input":{"command":"aws s3 ls"}}`, 2},
		{"aws sts get-caller-identity denied", `{"tool_input":{"command":"aws sts get-caller-identity"}}`, 2},
		{"kubectl get denied (route via coily)", `{"tool_input":{"command":"kubectl get pods"}}`, 2},
		{"gh pr view denied (route via coily)", `{"tool_input":{"command":"gh pr view 123"}}`, 2},
		{"ls allowed", `{"tool_input":{"command":"ls -la"}}`, 0},
		{"empty command allowed", `{"tool_input":{"command":""}}`, 0},
		// Coily binary check: paths outside homebrew rejected, brew paths allowed.
		{"~/go/bin/coily denied", `{"tool_input":{"command":"/Users/kai/go/bin/coily ssh"}}`, 2},
		{"/tmp/coily denied", `{"tool_input":{"command":"/tmp/coily ssh kubectl get pods"}}`, 2},
		{"./bin/coily denied", `{"tool_input":{"command":"./bin/coily lockdown --check"}}`, 2},
		{"/opt/homebrew/bin/coily allowed", `{"tool_input":{"command":"/opt/homebrew/bin/coily ssh"}}`, 0},
		{"/usr/local/bin/coily allowed", `{"tool_input":{"command":"/usr/local/bin/coily kubectl"}}`, 0},
		{"linuxbrew coily allowed", `{"tool_input":{"command":"/home/linuxbrew/.linuxbrew/bin/coily ssh"}}`, 0},
		{"coily denied via piped second segment", `{"tool_input":{"command":"echo go | /tmp/coily ssh"}}`, 2},
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

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func TestMergeDenyInto_CreatesFileWithCanonicalDeny(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, ".claude", "settings.local.json")
	d, _ := lockdown.LoadDefaults()

	mutated, err := lockdown.MergeDenyInto(target, d)
	if err != nil {
		t.Fatalf("MergeDenyInto: %v", err)
	}
	if !mutated {
		t.Errorf("expected mutated=true on fresh create")
	}
	raw, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	perms, _ := got["permissions"].(map[string]any)
	if perms == nil {
		t.Fatalf("permissions key missing: %s", string(raw))
	}
	if !contains(toStringSlice(perms["deny"]), "Bash(gh:*)") {
		t.Errorf("deny list missing canonical Bash(gh:*); got %v", perms["deny"])
	}
	if perms["allow"] != nil {
		t.Errorf("allow should be absent on fresh create; got %v", perms["allow"])
	}
}

func TestMergeDenyInto_PreservesAllowAndExtraKeys(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, ".claude", "settings.local.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	original := []byte(`{
  "permissions": {
    "allow": ["Bash(gh issue *)", "Bash(jq:*)"],
    "deny": ["Bash(rm -rf:*)"]
  },
  "env": {"FOO": "bar"}
}`)
	if err := os.WriteFile(target, original, 0o600); err != nil {
		t.Fatalf("seed write: %v", err)
	}
	d, _ := lockdown.LoadDefaults()

	mutated, err := lockdown.MergeDenyInto(target, d)
	if err != nil {
		t.Fatalf("MergeDenyInto: %v", err)
	}
	if !mutated {
		t.Errorf("expected mutated=true when canonical denies absent")
	}

	raw, _ := os.ReadFile(target)
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	perms, _ := got["permissions"].(map[string]any)
	if perms == nil {
		t.Fatalf("permissions missing: %s", string(raw))
	}

	allow := toStringSlice(perms["allow"])
	if !contains(allow, "Bash(gh issue *)") || !contains(allow, "Bash(jq:*)") {
		t.Errorf("allow not preserved verbatim; got %v", allow)
	}

	deny := toStringSlice(perms["deny"])
	if !contains(deny, "Bash(gh:*)") {
		t.Errorf("canonical Bash(gh:*) not merged into deny; got %v", deny)
	}
	if !contains(deny, "Bash(rm -rf:*)") {
		t.Errorf("pre-existing user deny entry dropped; got %v", deny)
	}

	env, _ := got["env"].(map[string]any)
	if env == nil || env["FOO"] != "bar" {
		t.Errorf("top-level env key not preserved; got %v", got["env"])
	}
}

func TestMergeDenyInto_NoOpWhenAlreadyCovered(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, ".claude", "settings.local.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	d, _ := lockdown.LoadDefaults()

	if _, err := lockdown.MergeDenyInto(target, d); err != nil {
		t.Fatalf("first MergeDenyInto: %v", err)
	}

	mutated, err := lockdown.MergeDenyInto(target, d)
	if err != nil {
		t.Fatalf("second MergeDenyInto: %v", err)
	}
	if mutated {
		t.Errorf("expected mutated=false on second call (idempotent)")
	}
}

func TestMergeDenyInto_DenyBeatsExistingAllowSemantics(t *testing.T) {
	// Document the load-bearing assumption: Claude Code applies deny ahead
	// of allow within a single settings file, so merging the canonical
	// deny into a file that allow-lists `Bash(gh issue *)` produces a
	// state where the allow stays present (we don't touch it) but the
	// deny would override it at runtime. This test only proves the file
	// state - the runtime semantics live in Claude Code itself.
	dir := t.TempDir()
	target := filepath.Join(dir, ".claude", "settings.local.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(target, []byte(`{"permissions":{"allow":["Bash(gh issue *)"]}}`), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	d, _ := lockdown.LoadDefaults()
	if _, err := lockdown.MergeDenyInto(target, d); err != nil {
		t.Fatalf("MergeDenyInto: %v", err)
	}
	raw, _ := os.ReadFile(target)
	var got map[string]any
	_ = json.Unmarshal(raw, &got)
	perms, _ := got["permissions"].(map[string]any)
	allow := toStringSlice(perms["allow"])
	deny := toStringSlice(perms["deny"])
	if !contains(allow, "Bash(gh issue *)") {
		t.Errorf("user allow entry was removed; got %v", allow)
	}
	if !contains(deny, "Bash(gh:*)") {
		t.Errorf("canonical deny not present alongside user allow; got %v", deny)
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

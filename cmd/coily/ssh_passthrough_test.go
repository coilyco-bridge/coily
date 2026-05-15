package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPosixQuote(t *testing.T) {
	cases := map[string]string{
		"plain":           "plain",
		"path/with/slash": "path/with/slash",
		"k=v":             "k=v",
		"--flag=val":      "--flag=val",
		"":                "''",
		"with space":      "'with space'",
		"semi;colon":      "'semi;colon'",
		"back`tick":       "'back`tick'",
		"it's mine":       `'it'\''s mine'`,
	}
	for in, want := range cases {
		got := posixQuote(in)
		if got != want {
			t.Errorf("posixQuote(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizePassthroughRest(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"strips leading dash-dash and leading coily", []string{"--", "coily", "whoami"}, []string{"whoami"}},
		{"strips leading dash-dash only", []string{"--", "systemctl", "status", "forgejo"}, []string{"systemctl", "status", "forgejo"}},
		{"strips leading coily only (no separator)", []string{"coily", "ops", "aws", "sts", "get-caller-identity"}, []string{"ops", "aws", "sts", "get-caller-identity"}},
		{"no leading coily leaves rest intact", []string{"systemctl", "status", "forgejo"}, []string{"systemctl", "status", "forgejo"}},
		{"empty stays empty", []string{}, []string{}},
		{"just dash-dash collapses", []string{"--"}, []string{}},
		{"dash-dash plus coily collapses", []string{"--", "coily"}, []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := normalizePassthroughRest(c.in)
			if len(got) != len(c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
			for i := range c.want {
				if got[i] != c.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, got[i], c.want[i])
				}
			}
		})
	}
}

func TestJoinPOSIX(t *testing.T) {
	got := joinPOSIX([]string{"coily", "--commit-scope=/home/kai", "gh", "issue", "create", "--body", "body with space; and semicolon"})
	want := `coily --commit-scope=/home/kai gh issue create --body 'body with space; and semicolon'`
	if got != want {
		t.Errorf("joinPOSIX = %q, want %q", got, want)
	}
}

// TestResolveSSHTarget_LooksAtReachableConfigs writes a .coily/coily.yaml
// in a temp repo, cd's into it, and confirms the resolver finds the
// declared target. The negative case "unknown alias" surfaces the known
// aliases in the error so the recovery path is obvious.
func TestResolveSSHTarget(t *testing.T) {
	t.Setenv("COILY_REPO_CONFIG", "") // ignore any inherited override
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, ".coily")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := `
commands:
  test: go test ./...
ssh:
  targets:
    kai-server:
      user: kai
      host: kai-server.example
      working_dir: /home/kai/projects/coilysiren
`
	if err := os.WriteFile(filepath.Join(cfgDir, "coily.yaml"), []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })

	r := &Runner{}
	tgt, err := r.resolveSSHTarget("kai-server")
	if err != nil {
		t.Fatalf("resolveSSHTarget kai-server: %v", err)
	}
	if tgt.User != "kai" || tgt.Host != "kai-server.example" || tgt.WorkingDir != "/home/kai/projects/coilysiren" {
		t.Errorf("target = %+v", tgt)
	}

	_, err = r.resolveSSHTarget("nope")
	if err == nil {
		t.Fatal("expected error for unknown alias")
	}
	if !strings.Contains(err.Error(), "kai-server") {
		t.Errorf("error does not list known alias kai-server: %v", err)
	}
}

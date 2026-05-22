package main

import "testing"

func TestAgentOSSlug(t *testing.T) {
	cases := map[string]string{
		"darwin":  "macos",
		"windows": "windows",
		"linux":   "linux",
		"plan9":   "plan9",
	}
	for goos, want := range cases {
		if got := agentOSSlug(goos); got != want {
			t.Errorf("agentOSSlug(%q) = %q, want %q", goos, got, want)
		}
	}
}

func TestAgentSessionTag(t *testing.T) {
	cases := []struct {
		name string
		sid  string
		want string
	}{
		{"empty", "", ""},
		{"short", "ab", "ab"},
		{"uuid tail", "11111111-2222-3333-4444-5555aaaaBC9d", "bc9d"},
		{"strips non-alnum", "x-y/z.q-w-e-r-t", "wert"},
	}
	for _, c := range cases {
		if got := agentSessionTag(c.sid); got != c.want {
			t.Errorf("%s: agentSessionTag(%q) = %q, want %q", c.name, c.sid, got, c.want)
		}
	}
}

func TestResolveAgentIdentity(t *testing.T) {
	noTag := resolveAgentIdentity("")
	if noTag.tag != "" {
		t.Errorf("tag should be empty with no session id, got %q", noTag.tag)
	}
	if noTag.name == "" || noTag.name[len(noTag.name)-1:] == "-" {
		t.Errorf("name %q should not be empty or trail a dash", noTag.name)
	}

	withTag := resolveAgentIdentity("deadbeef-0000-1111-2222-333344445e7f")
	if withTag.tag != "5e7f" {
		t.Errorf("tag = %q, want 5e7f", withTag.tag)
	}
	if got := withTag.name; len(got) < 5 || got[len(got)-5:] != "-5e7f" {
		t.Errorf("name %q should end with the session tag", got)
	}
}

func TestAgentWhoamiBlock(t *testing.T) {
	t.Setenv(sessionEnvVar, "deadbeef-0000-1111-2222-333344445e7f")
	block, ok := agentWhoami().(map[string]any)
	if !ok {
		t.Fatalf("agentWhoami() did not return a map")
	}
	if block["session_tag"] != "5e7f" {
		t.Errorf("session_tag = %q, want 5e7f", block["session_tag"])
	}
	if block["agent"] != "claude" {
		t.Errorf("agent = %q, want claude", block["agent"])
	}
}

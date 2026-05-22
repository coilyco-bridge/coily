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
		t.Setenv(sessionEnvVar, c.sid)
		if got := agentSessionTag(); got != c.want {
			t.Errorf("%s: agentSessionTag() = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestAgentWhoamiName(t *testing.T) {
	t.Setenv(sessionEnvVar, "")
	noTag, _ := agentWhoami().(map[string]any)
	if noTag["session_tag"] != "" {
		t.Errorf("session_tag should be empty with no session id, got %q", noTag["session_tag"])
	}

	t.Setenv(sessionEnvVar, "deadbeef-0000-1111-2222-333344445e7f")
	withTag, _ := agentWhoami().(map[string]any)
	if withTag["session_tag"] != "5e7f" {
		t.Errorf("session_tag = %q, want 5e7f", withTag["session_tag"])
	}
	name, _ := withTag["name"].(string)
	if name == "" || name[len(name)-5:] != "-5e7f" {
		t.Errorf("name %q should end with the session tag", name)
	}
}

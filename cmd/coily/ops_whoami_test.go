package main

import (
	"strings"
	"testing"
)

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
	if got := agentSessionTag(""); got != "" {
		t.Errorf("empty session id: agentSessionTag(\"\") = %q, want \"\"", got)
	}
	if got := agentSessionTag("  "); got != "" {
		t.Errorf("blank session id: agentSessionTag = %q, want \"\"", got)
	}

	const sid = "deadbeef-0000-1111-2222-333344445e7f"
	got := agentSessionTag(sid)

	// Shape: two dictatable letters then two dictatable digits.
	if len(got) != 4 {
		t.Fatalf("tag %q: want length 4", got)
	}
	if !strings.ContainsRune(tagLetters, rune(got[0])) || !strings.ContainsRune(tagLetters, rune(got[1])) {
		t.Errorf("tag %q: first two chars must be dictatable letters", got)
	}
	if !strings.ContainsRune(tagDigits, rune(got[2])) || !strings.ContainsRune(tagDigits, rune(got[3])) {
		t.Errorf("tag %q: last two chars must be dictatable digits", got)
	}

	// Deterministic: same session id always yields the same tag.
	if again := agentSessionTag(sid); again != got {
		t.Errorf("non-deterministic: %q then %q for the same session id", got, again)
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

	sid := "deadbeef-0000-1111-2222-333344445e7f"
	wantTag := agentSessionTag(sid)
	withTag := resolveAgentIdentity(sid)
	if withTag.tag != wantTag {
		t.Errorf("tag = %q, want %q", withTag.tag, wantTag)
	}
	if got := withTag.name; !strings.HasSuffix(got, "-"+wantTag) {
		t.Errorf("name %q should end with the session tag %q", got, wantTag)
	}
}

func TestAgentWhoamiBlock(t *testing.T) {
	sid := "deadbeef-0000-1111-2222-333344445e7f"
	t.Setenv(sessionEnvVar, sid)
	block, ok := agentWhoami().(map[string]any)
	if !ok {
		t.Fatalf("agentWhoami() did not return a map")
	}
	if want := agentSessionTag(sid); block["session_tag"] != want {
		t.Errorf("session_tag = %q, want %q", block["session_tag"], want)
	}
	if block["agent"] != "claude" {
		t.Errorf("agent = %q, want claude", block["agent"])
	}
}

package main

import (
	"encoding/json"
	"testing"
)

func TestParseForgejoRepoSlug(t *testing.T) {
	cases := []struct {
		in      string
		owner   string
		repo    string
		wantErr bool
	}{
		{"coilysiren/coily", "coilysiren", "coily", false},
		{"coilysiren/cli-guard", "coilysiren", "cli-guard", false},
		{"a/b.c_d-e", "a", "b.c_d-e", false},
		{"", "", "", true},
		{"single", "", "", true},
		{"a/b/c", "", "", true},
		{"/repo", "", "", true},
		{"owner/", "", "", true},
		{"-owner/repo", "", "", true},
		{"owner/-repo", "", "", true},
		{"owner/re po", "", "", true},
		{"owner/re;po", "", "", true},
	}
	for _, c := range cases {
		o, r, err := parseForgejoRepoSlug(c.in)
		gotErr := err != nil
		if gotErr != c.wantErr {
			t.Errorf("parseForgejoRepoSlug(%q) err=%v, want err=%v", c.in, err, c.wantErr)
			continue
		}
		if !c.wantErr && (o != c.owner || r != c.repo) {
			t.Errorf("parseForgejoRepoSlug(%q) = (%q,%q), want (%q,%q)", c.in, o, r, c.owner, c.repo)
		}
	}
}

func TestForgejoIssueCreateBodyShape(t *testing.T) {
	got, err := json.Marshal(forgejoIssueCreateBody{Title: "t", Body: "b"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"title":"t","body":"b"}`
	if string(got) != want {
		t.Errorf("payload = %s, want %s", got, want)
	}
}

package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestForgejoRedirectError pins the org-alias silent-no-op fix
// (coilyco-bridge/coily#178, #160): reads may follow a redirect, mutating
// methods must be refused with a canonical --repo suggestion.
func TestForgejoRedirectError(t *testing.T) {
	const alias = "https://f.me/api/v1/repos/alias/coily/issues"
	const canon = "https://f.me/api/v1/repos/canon/coily/issues"

	for _, m := range []string{http.MethodGet, http.MethodHead} {
		if err := forgejoRedirectError(m, alias, canon); err != nil {
			t.Errorf("%s should follow redirects, got %v", m, err)
		}
	}

	for _, m := range []string{http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete} {
		err := forgejoRedirectError(m, alias, canon)
		if err == nil {
			t.Fatalf("%s should refuse the redirect", m)
		}
		if !strings.Contains(err.Error(), "silently no-op") {
			t.Errorf("%s error missing rationale: %v", m, err)
		}
		if !strings.Contains(err.Error(), "--repo canon/coily") {
			t.Errorf("%s error missing canonical --repo suggestion: %v", m, err)
		}
	}
}

func TestForgejoRepoFromAPIPath(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"https://f.me/api/v1/repos/canon/coily/issues", "canon/coily", true},
		{"https://f.me/api/v1/repos/o/r/releases", "o/r", true},
		{"https://f.me/api/v1/repos/o/r", "o/r", true},
		{"https://f.me/api/v1/repos/o", "", false},
		{"https://f.me/healthz", "", false},
	}
	for _, c := range cases {
		got, ok := forgejoRepoFromAPIPath(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf("forgejoRepoFromAPIPath(%q) = (%q,%v), want (%q,%v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

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

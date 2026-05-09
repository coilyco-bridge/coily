package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestParseGhRepoFlag covers the four argv shapes gh accepts for --repo
// (long flag two-token, long flag equals, short flag two-token, short flag
// equals) plus the rejection paths that ghRepoScopeHint relies on.
func TestParseGhRepoFlag(t *testing.T) {
	cases := []struct {
		name      string
		argv      []string
		wantOwner string
		wantName  string
	}{
		{"long two-token", []string{"issue", "list", "--repo", "coilysiren/coily"}, "coilysiren", "coily"},
		{"long equals", []string{"issue", "list", "--repo=coilysiren/coily"}, "coilysiren", "coily"},
		{"short two-token", []string{"issue", "list", "-R", "coilysiren/coily"}, "coilysiren", "coily"},
		{"short equals", []string{"issue", "list", "-R=coilysiren/coily"}, "coilysiren", "coily"},
		{"missing value", []string{"issue", "list", "--repo"}, "", ""},
		{"non-slash value", []string{"issue", "list", "--repo", "coilysiren-only"}, "", ""},
		{"no flag", []string{"issue", "list"}, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o, n := parseGhRepoFlag(tc.argv)
			if o != tc.wantOwner || n != tc.wantName {
				t.Errorf("parseGhRepoFlag(%v) = (%q, %q), want (%q, %q)",
					tc.argv, o, n, tc.wantOwner, tc.wantName)
			}
		})
	}
}

// TestGhRepoScopeHint verifies the gh --repo argv hint resolves to
// ~/projects/coilysiren/<name> when (a) the owner is "coilysiren" and (b)
// that local clone exists with a .git directory. Other-owner inputs and
// missing-clone inputs return "" so the audit pipeline falls back to
// flag/env resolution and surfaces scope_unresolved truthfully.
func TestGhRepoScopeHint(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	clone := filepath.Join(home, "projects", "coilysiren", "coily")
	if err := os.MkdirAll(filepath.Join(clone, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir clone: %v", err)
	}

	cases := []struct {
		name string
		argv []string
		want string
	}{
		{
			name: "coilysiren repo with local clone",
			argv: []string{"issue", "list", "--repo", "coilysiren/coily"},
			want: clone,
		},
		{
			name: "coilysiren repo without local clone",
			argv: []string{"issue", "list", "--repo", "coilysiren/missing"},
			want: "",
		},
		{
			name: "non-coilysiren owner declines",
			argv: []string{"issue", "list", "--repo", "octocat/coily"},
			want: "",
		},
		{
			name: "no --repo flag",
			argv: []string{"issue", "list"},
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ghRepoScopeHint(tc.argv)
			if got != tc.want {
				t.Errorf("ghRepoScopeHint(%v) = %q, want %q", tc.argv, got, tc.want)
			}
		})
	}
}

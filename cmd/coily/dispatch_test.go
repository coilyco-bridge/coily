package main

import (
	"strings"
	"testing"
)

func TestParseIssueRef(t *testing.T) {
	cases := []struct {
		in     string
		wantOK bool
		owner  string
		repo   string
		num    int
	}{
		{"coilysiren/coily#136", true, "coilysiren", "coily", 136},
		{"  coilysiren/coily#136  ", true, "coilysiren", "coily", 136},
		{"https://github.com/coilysiren/coily/issues/136", true, "coilysiren", "coily", 136},
		{"https://github.com/coilysiren/coily/issues/136/", true, "coilysiren", "coily", 136},
		{"http://github.com/coilysiren/coily/issues/1", true, "coilysiren", "coily", 1},
		{"coilysiren/coily/issues/136", false, "", "", 0},
		{"coilysiren/coily#0", false, "", "", 0},
		{"coilysiren#136", false, "", "", 0},
		{"", false, "", "", 0},
		{"random gibberish", false, "", "", 0},
		{"github.com/coilysiren/coily/issues/136", false, "", "", 0}, // missing scheme
	}
	for _, tc := range cases {
		got, err := parseIssueRef(tc.in)
		if tc.wantOK {
			if err != nil {
				t.Errorf("parseIssueRef(%q): unexpected err: %v", tc.in, err)
				continue
			}
			if got.Owner != tc.owner || got.Repo != tc.repo || got.Number != tc.num {
				t.Errorf("parseIssueRef(%q) = %+v, want owner=%s repo=%s num=%d",
					tc.in, got, tc.owner, tc.repo, tc.num)
			}
		} else {
			if err == nil {
				t.Errorf("parseIssueRef(%q): expected error, got %+v", tc.in, got)
			}
		}
	}
}

func TestFirstIssueRef(t *testing.T) {
	argv := []string{"coily", "dispatch", "--dry-run", "coilysiren/coily#136"}
	ref := firstIssueRef(argv)
	if ref == nil {
		t.Fatal("firstIssueRef returned nil; expected match")
	}
	if ref.Repo != "coily" || ref.Number != 136 {
		t.Errorf("firstIssueRef = %+v, want coily#136", ref)
	}
}

func TestFirstIssueRef_NoMatch(t *testing.T) {
	argv := []string{"coily", "dispatch", "--dry-run"}
	if ref := firstIssueRef(argv); ref != nil {
		t.Errorf("firstIssueRef = %+v, want nil", ref)
	}
}

func TestSeedPrompt_IncludesIssueAndFooter(t *testing.T) {
	ref := &issueRef{Owner: "coilysiren", Repo: "coily", Number: 136}
	issue := &ghIssue{
		Number: 136,
		Title:  "coily dispatch",
		Body:   "design body here",
		State:  "OPEN",
		URL:    "https://github.com/coilysiren/coily/issues/136",
	}
	got := seedPrompt(ref, issue)
	for _, want := range []string{
		"coilysiren/coily#136",
		"coily dispatch",
		"design body here",
		"https://github.com/coilysiren/coily/issues/136",
		"AGENTS.md",
		"closes #",
		"--no-verify",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("seedPrompt missing %q in:\n%s", want, got)
		}
	}
}

func TestIssueRef_String(t *testing.T) {
	ref := issueRef{Owner: "coilysiren", Repo: "coily", Number: 136}
	if got, want := ref.String(), "coilysiren/coily#136"; got != want {
		t.Errorf("issueRef.String() = %q, want %q", got, want)
	}
}

func TestAllowedOwner(t *testing.T) {
	// Lock the owner constant so a casual edit cannot widen the boundary
	// without also tripping this test. Part of the security claim for the
	// dispatch verb (see coilysiren/coily#136).
	if allowedOwner != "coilysiren" {
		t.Errorf("allowedOwner = %q, want coilysiren; widening the boundary needs a deliberate review", allowedOwner)
	}
}

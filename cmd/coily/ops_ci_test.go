package main

import (
	"strings"
	"testing"
	"time"
)

func TestSplitOwnerRepo(t *testing.T) {
	cases := []struct {
		slug                string
		wantOwner, wantRepo string
		wantOK              bool
	}{
		{"coilysiren/coily", "coilysiren", "coily", true},
		{"coilysiren", "", "", false},
		{"coilysiren/", "", "", false},
		{"/coily", "", "", false},
		{"a/b/c", "", "", false},
		{"", "", "", false},
	}
	for _, c := range cases {
		owner, repo, ok := splitOwnerRepo(c.slug)
		if owner != c.wantOwner || repo != c.wantRepo || ok != c.wantOK {
			t.Errorf("splitOwnerRepo(%q) = (%q, %q, %v), want (%q, %q, %v)",
				c.slug, owner, repo, ok, c.wantOwner, c.wantRepo, c.wantOK)
		}
	}
}

func TestCIStatusGlyph(t *testing.T) {
	cases := []struct {
		status, conclusion, want string
	}{
		{"completed", "success", "✓ pass"},
		{"completed", "failure", "✗ fail"},
		{"completed", "cancelled", "⊘ cancel"},
		{"completed", "timed_out", "✗ timeout"},
		{"in_progress", "", "● run"},
		{"queued", "", "○ queued"},
	}
	for _, c := range cases {
		if got := ciStatusGlyph(c.status, c.conclusion); got != c.want {
			t.Errorf("ciStatusGlyph(%q, %q) = %q, want %q", c.status, c.conclusion, got, c.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		s    string
		n    int
		want string
	}{
		{"short", 10, "short"},
		{"exactly-ten", 11, "exactly-ten"},
		{"this is way too long", 10, "this is w…"},
	}
	for _, c := range cases {
		if got := truncate(c.s, c.n); got != c.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", c.s, c.n, got, c.want)
		}
		if len([]rune(truncate(c.s, c.n))) > c.n {
			t.Errorf("truncate(%q, %d) overran the rune cap", c.s, c.n)
		}
	}
}

func TestCIAge(t *testing.T) {
	if got := ciAge(""); got != "?" {
		t.Errorf("ciAge(empty) = %q, want ?", got)
	}
	if got := ciAge("not-a-timestamp"); got != "not-a-timestamp" {
		t.Errorf("ciAge(garbage) should pass through, got %q", got)
	}
	recent := time.Now().Add(-30 * time.Second).UTC().Format(time.RFC3339)
	if got := ciAge(recent); got != "just now" {
		t.Errorf("ciAge(30s ago) = %q, want 'just now'", got)
	}
	hoursAgo := time.Now().Add(-3 * time.Hour).UTC().Format(time.RFC3339)
	if got := ciAge(hoursAgo); !strings.HasSuffix(got, "h") {
		t.Errorf("ciAge(3h ago) = %q, want an h-suffixed age", got)
	}
}

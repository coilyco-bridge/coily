package main

import (
	"strings"
	"testing"
)

// TestIsGHActionsRead_Blocked pins the gh invocations the gate must
// reject: the whole GitHub Actions / CI status surface (coilysiren/coily#305).
func TestIsGHActionsRead_Blocked(t *testing.T) {
	blocked := [][]string{
		{"run", "list", "--repo", "coilysiren/coily"},
		{"run", "view", "12345"},
		{"run", "watch"},
		{"run"},
		{"workflow", "list"},
		{"workflow", "run", "release.yml"},
		{"cache", "list"},
		{"pr", "checks", "42"},
		{"pr", "checks"},
		{"api", "/repos/coilysiren/coily/actions/runs"},
		{"api", "repos/coilysiren/coily/actions/workflows"},
		{"api", "-X", "GET", "/repos/coilysiren/coily/actions/runs/1/jobs"},
		{"api", "/repos/coilysiren/coily/actions"},
	}
	for _, argv := range blocked {
		if !isGHActionsRead(argv) {
			t.Errorf("isGHActionsRead(%v) = false, want true (Actions surface must be blocked)", argv)
		}
		if err := ghActionsGate(argv); err == nil {
			t.Errorf("ghActionsGate(%v) = nil, want a hard error", argv)
		}
	}
}

// TestIsGHActionsRead_Allowed pins the gh invocations the gate must let
// through. Scope is Actions-only: issues, PRs, releases, rate-limit
// checks, and repo reads stay untouched.
func TestIsGHActionsRead_Allowed(t *testing.T) {
	allowed := [][]string{
		{"issue", "view", "305", "--repo", "coilysiren/coily"},
		{"issue", "list", "--repo", "coilysiren/coily"},
		{"pr", "view", "42"},
		{"pr", "list"},
		{"pr", "create", "--title", "T"},
		{"repo", "view", "coilysiren/coily"},
		{"release", "list"},
		{"api", "/repos/coilysiren/coily"},
		{"api", "/repos/coilysiren/coily/issues"},
		{"api", "/rate_limit"},
		// `-f` field values that merely mention "actions" must not trip
		// the path check - they carry an `=` and are not API paths.
		{"api", "repos/coilysiren/coily/issues", "-f", "body=see the /actions/ tab"},
		{"api", "repos/coilysiren/coily/issues", "-f", "title=actions"},
		{},
	}
	for _, argv := range allowed {
		if isGHActionsRead(argv) {
			t.Errorf("isGHActionsRead(%v) = true, want false (non-Actions gh must pass)", argv)
		}
		if err := ghActionsGate(argv); err != nil {
			t.Errorf("ghActionsGate(%v) = %v, want nil", argv, err)
		}
	}
}

// TestGHActionsBlockMessage_PointsToPlaywright keeps the hard error
// legible: it must name the issue and route the operator to the
// Playwright path, since the API fallback no longer exists.
func TestGHActionsBlockMessage_PointsToPlaywright(t *testing.T) {
	for _, want := range []string{"coilysiren/coily#305", "Playwright", "/actions"} {
		if !strings.Contains(ghActionsBlockMessage, want) {
			t.Errorf("ghActionsBlockMessage missing %q", want)
		}
	}
}

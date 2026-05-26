package main

import (
	"strings"
	"testing"
)

func TestIsGHIssueCreate_Blocked(t *testing.T) {
	blocked := [][]string{
		{"issue", "create"},
		{"issue", "create", "--repo", "coilysiren/coily", "--title", "t"},
		{"issue", "new"},
		{"api", "-X", "POST", "/repos/coilysiren/coily/issues"},
		{"api", "--method", "POST", "repos/coilysiren/coily/issues"},
		{"api", "/repos/coilysiren/coily/issues", "-f", "title=hi"},
		{"api", "-f", "title=hi", "/repos/coilysiren/coily/issues"},
		{"api", "--method=POST", "/repos/coilysiren/coily/issues"},
	}
	for _, argv := range blocked {
		if !isGHIssueCreate(argv) {
			t.Errorf("isGHIssueCreate(%v) = false, want true", argv)
		}
		if err := ghIssueCreateGate(argv); err == nil {
			t.Errorf("ghIssueCreateGate(%v) = nil, want hard error", argv)
		}
	}
}

func TestIsGHIssueCreate_Allowed(t *testing.T) {
	allowed := [][]string{
		{"issue", "list"},
		{"issue", "view", "42"},
		{"issue", "edit", "42", "--add-label", "bug"},
		{"issue", "close", "42"},
		{"issue", "comment", "42", "--body", "ack"},
		{"pr", "create", "--title", "t"},
		{"api", "/repos/coilysiren/coily/issues"}, // GET (no method, no fields)
		{"api", "-X", "GET", "/repos/coilysiren/coily/issues"},
		{"api", "/repos/coilysiren/coily/issues/42"}, // single-issue endpoint
		{"api", "-X", "POST", "/repos/coilysiren/coily/issues/42/comments"},
		{"api", "-X", "PATCH", "/repos/coilysiren/coily/issues/42"},
		{"api", "/repos/coilysiren/coily/pulls"},
	}
	for _, argv := range allowed {
		if isGHIssueCreate(argv) {
			t.Errorf("isGHIssueCreate(%v) = true, want false", argv)
		}
		if err := ghIssueCreateGate(argv); err != nil {
			t.Errorf("ghIssueCreateGate(%v) = %v, want nil", argv, err)
		}
	}
}

func TestGHPreflightGate_ChainsBothGates(t *testing.T) {
	if err := ghPreflightGate([]string{"run", "list"}); err == nil {
		t.Error("preflight should still block actions surface")
	}
	if err := ghPreflightGate([]string{"issue", "create"}); err == nil {
		t.Error("preflight should block issue create")
	}
	if err := ghPreflightGate([]string{"issue", "view", "1"}); err != nil {
		t.Errorf("preflight should allow issue view, got: %v", err)
	}
	// Sanity: block message names Forgejo so recovery is obvious.
	err := ghPreflightGate([]string{"issue", "create"})
	if err == nil || !strings.Contains(err.Error(), "Forgejo") {
		t.Errorf("block message should mention Forgejo, got: %v", err)
	}
}

package main

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/audit"
)

func TestBuildDashboardView_GroupsByVerb(t *testing.T) {
	now := time.Now().Unix()
	records := []audit.Record{
		{Verb: "ops.gh", Timestamp: now - 10, Decision: audit.DecisionAccept, Argv: []string{"coily", "ops", "gh", "issue", "view"}, RepoRoot: "/repos/coily"},
		{Verb: "ops.gh", Timestamp: now - 5, Decision: audit.DecisionAccept, Argv: []string{"coily", "ops", "gh", "issue", "list"}, RepoRoot: "/repos/agentic-os-kai", Egress: []audit.EgressRow{{Host: "api.github.com", Decision: "allow", BytesDown: 100}}},
		{Verb: "ops.gh", Timestamp: now - 1, Decision: audit.DecisionAccept, Argv: []string{"coily", "ops", "gh", "issue", "view"}, RepoRoot: "/repos/coily"},
		{Verb: "dispatch", Timestamp: now - 20, Decision: audit.DecisionAccept, Argv: []string{"coily", "dispatch", "coilysiren/coily#1"}, ExitCode: 1},
	}
	view := buildDashboardView(records, now-60)

	if got, want := view.TotalRecords, len(records); got != want {
		t.Errorf("TotalRecords = %d, want %d", got, want)
	}
	if got, want := len(view.Verbs), 2; got != want {
		t.Fatalf("Verbs len = %d, want %d", got, want)
	}
	// Sorted by Total desc, so ops.gh (3) comes before dispatch (1).
	gh := view.Verbs[0]
	if gh.Verb != "ops.gh" {
		t.Errorf("first verb = %q, want ops.gh", gh.Verb)
	}
	if gh.Total != 3 {
		t.Errorf("ops.gh.Total = %d, want 3", gh.Total)
	}
	if gh.Accepted != 3 {
		t.Errorf("ops.gh.Accepted = %d, want 3", gh.Accepted)
	}
	wantRepos := []string{"agentic-os-kai", "coily"}
	if !stringSliceEqual(gh.Repos, wantRepos) {
		t.Errorf("ops.gh.Repos = %v, want %v", gh.Repos, wantRepos)
	}
	if !stringSliceEqual(gh.EgressHosts, []string{"api.github.com"}) {
		t.Errorf("ops.gh.EgressHosts = %v", gh.EgressHosts)
	}
	// Two unique argvs ("issue view", "issue list"), capped at argvSampleCap.
	if len(gh.ArgvSamples) != 2 {
		t.Errorf("ops.gh.ArgvSamples len = %d, want 2 (%v)", len(gh.ArgvSamples), gh.ArgvSamples)
	}
	if gh.LastRunUnix != now-1 {
		t.Errorf("ops.gh.LastRunUnix = %d, want %d", gh.LastRunUnix, now-1)
	}

	dispatch := view.Verbs[1]
	if dispatch.Failed != 1 {
		t.Errorf("dispatch.Failed = %d, want 1 (non-zero exit on accepted row)", dispatch.Failed)
	}
}

func TestBuildDashboardView_ArgvSampleDedup(t *testing.T) {
	// 10 identical argvs collapse to one sample.
	now := time.Now().Unix()
	var records []audit.Record
	for i := 0; i < 10; i++ {
		records = append(records, audit.Record{
			Verb:      "ops.gh",
			Timestamp: now - int64(i),
			Decision:  audit.DecisionAccept,
			Argv:      []string{"coily", "ops", "gh", "issue", "view"},
		})
	}
	view := buildDashboardView(records, now-60)
	if got, want := len(view.Verbs[0].ArgvSamples), 1; got != want {
		t.Errorf("ArgvSamples len = %d, want %d", got, want)
	}
	if got, want := view.Verbs[0].Total, 10; got != want {
		t.Errorf("Total = %d, want %d (count should not drop with dedup)", got, want)
	}
}

func TestRenderDashboard_WritesFileAndBakesData(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "dash.html")
	view := dashboardView{
		GeneratedAt:  "2026-05-13T00:00:00Z",
		WindowStart:  "2026-05-12T00:00:00Z",
		TotalRecords: 1,
		Verbs: []verbBucket{
			{Verb: "ops.gh", Total: 1, Accepted: 1, Repos: []string{"coily"}},
		},
	}
	if err := renderDashboard(out, view); err != nil {
		t.Fatalf("renderDashboard: %v", err)
	}
	body, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read out: %v", err)
	}
	for _, want := range []string{
		"<!DOCTYPE html>",
		"const DATA = {",
		`"verb": "ops.gh"`,
		`"total": 1`,
		"coily audit dashboard",
	} {
		if !bytes.Contains(body, []byte(want)) {
			t.Errorf("rendered HTML missing %q", want)
		}
	}
}

func TestParseSince_AcceptsDays(t *testing.T) {
	got, err := parseSince("7d")
	if err != nil {
		t.Fatalf("parseSince(7d): %v", err)
	}
	want := time.Now().Add(-7 * 24 * time.Hour).Unix()
	if got < want-5 || got > want+5 {
		t.Errorf("parseSince(7d) = %d, want ~%d", got, want)
	}
}

func TestEmbeddedTemplate_Parses(t *testing.T) {
	// Sanity: the embedded template should parse. If it doesn't, every
	// dashboard run fails at runtime with a parse error; this test fails
	// loudly at build time instead.
	if _, err := template.New("t").Parse(dashboardTemplate); err != nil {
		t.Fatalf("embedded template did not parse: %v", err)
	}
	if !strings.Contains(dashboardTemplate, "DataJSON") {
		t.Errorf("embedded template missing DataJSON binding")
	}
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestAddArgvSample_RedactsSecretFlagValues pins coily#252 (forward
// action for finding #228): argv re-published into the dashboard goes
// through cross-surface redaction, so a SecureString put-parameter
// invocation does not leak the value into the JSON output.
func TestAddArgvSample_RedactsSecretFlagValues(t *testing.T) {
	acc := &verbAcc{argvSeen: map[string]struct{}{}}
	acc.addArgvSample([]string{
		"coily", "ops", "aws", "ssm", "put-parameter",
		"--name", "/tailscale/oauth/backend/client-id",
		"--value", "tskey-client-VERY_SECRET_HERE",
		"--type", "SecureString",
		"--overwrite",
	})
	if len(acc.bucket.ArgvSamples) != 1 {
		t.Fatalf("ArgvSamples length = %d, want 1", len(acc.bucket.ArgvSamples))
	}
	sample := acc.bucket.ArgvSamples[0]
	if strings.Contains(sample, "VERY_SECRET_HERE") {
		t.Errorf("argv sample leaked plaintext secret: %q", sample)
	}
	if !strings.Contains(sample, "[REDACTED]") {
		t.Errorf("argv sample missing redaction marker: %q", sample)
	}
	// Non-secret tokens should survive redaction.
	for _, want := range []string{"coily", "ops", "aws", "ssm", "put-parameter", "--type", "SecureString"} {
		if !strings.Contains(sample, want) {
			t.Errorf("argv sample missing non-secret token %q: %q", want, sample)
		}
	}
}

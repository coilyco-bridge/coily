package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// actionsPageFixture is a trimmed-down github.com Actions page carrying
// three check-suite rows: a successful run, a failed run, and an
// in-progress run. The element shapes (the check_suite_<id> container,
// the run anchor with its aria-label, the commit/branch/user anchors,
// the relative-time element) mirror what github.com renders today. A
// fourth row is deliberately malformed - no run anchor - to exercise the
// skip-unparseable-rows path.
const actionsPageFixture = `<!DOCTYPE html>
<html><body>
<div class="Box-row js-socket-channel" id="check_suite_70022630903" data-url="/coilysiren/coily/actions/workflow-run/70022630903">
  <a href="/coilysiren/coily/actions/runs/26194128761" aria-label="completed successfully:  Run 239 of release. feat: add args-file to coily ops mcporter passthrough (#287)">
    <span class="markdown-title">feat: add args-file</span>
  </a>
  <span class="text-bold">release</span> #239:
  Commit <a class="Link--muted" href="/coilysiren/coily/commit/6ab15e1038b02d52d1c7ee35ba352f255f8b7c08">6ab15e1</a>
  pushed by
  <a data-hovercard-type="user" href="/coilysiren">coilysiren</a>
  <a class="branch-name" title="main" href="/coilysiren/coily/tree/refs/heads/main">main</a>
  <relative-time datetime="2026-05-20T22:40:17Z"></relative-time>
</div>
<div class="Box-row js-socket-channel" id="check_suite_70000000001" data-url="/coilysiren/coily/actions/workflow-run/70000000001">
  <a href="/coilysiren/coily/actions/runs/26190000000" aria-label="completed with failures:  Run 257 of test. fix the flaky thing">
    <span class="markdown-title">fix the flaky thing</span>
  </a>
  <span class="text-bold">test</span> #257:
  Commit <a class="Link--muted" href="/coilysiren/coily/commit/abcdef1234567">abcdef1</a>
  pushed by
  <a data-hovercard-type="user" href="/coilysiren">coilysiren</a>
  <a class="branch-name" title="dispatch/issue-267" href="/coilysiren/coily/tree/refs/heads/dispatch/issue-267">dispatch/issue-267</a>
  <relative-time datetime="2026-05-20T20:00:00Z"></relative-time>
</div>
<div class="Box-row js-socket-channel" id="check_suite_70000000002" data-url="/coilysiren/coily/actions/workflow-run/70000000002">
  <a href="/coilysiren/coily/actions/runs/26200000000" aria-label="in progress:  Run 12 of docker. building the image">
    <span class="markdown-title">building the image</span>
  </a>
  <span class="text-bold">docker</span> #12:
  pull request by
  <a data-hovercard-type="user" href="/coilysiren">coilysiren</a>
  <relative-time datetime="2026-05-20T23:00:00Z"></relative-time>
</div>
<div class="Box-row" id="check_suite_70000000003">
  <span>a row with no run anchor at all</span>
</div>
</body></html>`

func TestParseActionsRunsHTML(t *testing.T) {
	runs, err := parseActionsRunsHTML(strings.NewReader(actionsPageFixture), "coilysiren", "coily")
	if err != nil {
		t.Fatalf("parseActionsRunsHTML: %v", err)
	}
	if len(runs) != 3 {
		t.Fatalf("got %d runs, want 3 (the malformed fourth row should be skipped)", len(runs))
	}

	want := []webWorkflowRun{
		{
			ID: 26194128761, Name: "release", DisplayTitle: "feat: add args-file to coily ops mcporter passthrough (#287)",
			HeadBranch: "main", HeadSHA: "6ab15e1038b02d52d1c7ee35ba352f255f8b7c08", RunNumber: 239,
			Event: "push", Status: "completed", Conclusion: "success",
			HTMLURL:   "https://github.com/coilysiren/coily/actions/runs/26194128761",
			CreatedAt: "2026-05-20T22:40:17Z", CheckSuiteID: 70022630903,
			Actor: &actor{Login: "coilysiren"},
		},
		{
			ID: 26190000000, Name: "test", DisplayTitle: "fix the flaky thing",
			HeadBranch: "dispatch/issue-267", HeadSHA: "abcdef1234567", RunNumber: 257,
			Event: "push", Status: "completed", Conclusion: "failure",
			HTMLURL:   "https://github.com/coilysiren/coily/actions/runs/26190000000",
			CreatedAt: "2026-05-20T20:00:00Z", CheckSuiteID: 70000000001,
			Actor: &actor{Login: "coilysiren"},
		},
		{
			ID: 26200000000, Name: "docker", DisplayTitle: "building the image",
			RunNumber: 12, Event: "pull_request", Status: "in_progress", Conclusion: "",
			HTMLURL:   "https://github.com/coilysiren/coily/actions/runs/26200000000",
			CreatedAt: "2026-05-20T23:00:00Z", CheckSuiteID: 70000000002,
			Actor: &actor{Login: "coilysiren"},
		},
	}

	for i, w := range want {
		got := runs[i]
		if got.ID != w.ID || got.Name != w.Name || got.DisplayTitle != w.DisplayTitle ||
			got.HeadBranch != w.HeadBranch || got.HeadSHA != w.HeadSHA || got.RunNumber != w.RunNumber ||
			got.Event != w.Event || got.Status != w.Status || got.Conclusion != w.Conclusion ||
			got.HTMLURL != w.HTMLURL || got.CreatedAt != w.CreatedAt || got.CheckSuiteID != w.CheckSuiteID {
			t.Errorf("run %d:\n got  %+v\n want %+v", i, got, w)
		}
		if (got.Actor == nil) != (w.Actor == nil) || (got.Actor != nil && got.Actor.Login != w.Actor.Login) {
			t.Errorf("run %d actor: got %+v, want %+v", i, got.Actor, w.Actor)
		}
	}
}

func TestClassifyStatusPhrase(t *testing.T) {
	cases := []struct {
		phrase, wantStatus, wantConclusion string
	}{
		{"completed successfully", "completed", "success"},
		{"completed with failures", "completed", "failure"},
		{"startup failure", "completed", "failure"},
		{"completed cancelled", "completed", "cancelled"},
		{"completed skipped", "completed", "skipped"},
		{"timed out", "completed", "timed_out"},
		{"in progress", "in_progress", ""},
		{"queued", "queued", ""},
		{"waiting", "queued", ""},
		{"some unknown phrase", "some unknown phrase", ""},
	}
	for _, c := range cases {
		gotStatus, gotConclusion := classifyStatusPhrase(c.phrase)
		if gotStatus != c.wantStatus || gotConclusion != c.wantConclusion {
			t.Errorf("classifyStatusPhrase(%q) = (%q, %q), want (%q, %q)",
				c.phrase, gotStatus, gotConclusion, c.wantStatus, c.wantConclusion)
		}
	}
}

func TestReadActionsRunsViaWebShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/coilysiren/coily/actions" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(actionsPageFixture))
	}))
	defer srv.Close()

	orig := webRunsBaseURL
	webRunsBaseURL = srv.URL
	defer func() { webRunsBaseURL = orig }()

	raw, err := readActionsRunsViaWeb(context.Background(), "coilysiren", "coily", "")
	if err != nil {
		t.Fatalf("readActionsRunsViaWeb: %v", err)
	}
	var result webRunsResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result.TotalCount != 3 || len(result.WorkflowRuns) != 3 {
		t.Fatalf("got total_count=%d, %d runs; want 3 and 3", result.TotalCount, len(result.WorkflowRuns))
	}
}

func TestFetchActionsRunsHTMLNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// A private repo without a cookie: github.com answers 404 to
		// avoid leaking existence. The fetch must report an error so the
		// caller falls through to the next budget tier.
		http.Error(w, "Not Found", http.StatusNotFound)
	}))
	defer srv.Close()

	orig := webRunsBaseURL
	webRunsBaseURL = srv.URL
	defer func() { webRunsBaseURL = orig }()

	_, err := readActionsRunsViaWeb(context.Background(), "coilysiren", "secret", "")
	if err == nil {
		t.Fatal("expected an error for a 404 Actions page, got nil")
	}
}

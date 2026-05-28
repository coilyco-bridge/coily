package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestForgejoLogCursorsBody_Shape pins the POST body the web log route
// expects: one cursor per requested step, anchored at cursor 0 and expanded.
func TestForgejoLogCursorsBody_Shape(t *testing.T) {
	got, err := forgejoLogCursorsBody([]int{0, 1, 2})
	if err != nil {
		t.Fatalf("forgejoLogCursorsBody: %v", err)
	}
	var parsed struct {
		LogCursors []struct {
			Step     int  `json:"step"`
			Cursor   int  `json:"cursor"`
			Expanded bool `json:"expanded"`
		} `json:"logCursors"`
	}
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if len(parsed.LogCursors) != 3 {
		t.Fatalf("want 3 cursors, got %d", len(parsed.LogCursors))
	}
	for i, c := range parsed.LogCursors {
		if c.Step != i || c.Cursor != 0 || !c.Expanded {
			t.Errorf("cursor %d = %+v, want {step:%d cursor:0 expanded:true}", i, c, i)
		}
	}
}

func TestFindForgejoJobIndex(t *testing.T) {
	jobs := []string{"release", "bump-formula", "windows-assets (amd64)"}
	idx, err := findForgejoJobIndex(jobs, "bump-formula")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != 1 {
		t.Errorf("want index 1, got %d", idx)
	}
	if _, err := findForgejoJobIndex(jobs, "nope"); err == nil {
		t.Errorf("expected error for missing job name")
	}
}

func TestForgejoJobView_MaxAttempt(t *testing.T) {
	var none forgejoJobView
	if got := none.maxAttempt(); got != 1 {
		t.Errorf("empty allAttempts: want default 1, got %d", got)
	}
	var v forgejoJobView
	v.State.Run.AllAttempts = []forgejoRunAttempt{{Number: 1}, {Number: 3}, {Number: 2}}
	if got := v.maxAttempt(); got != 3 {
		t.Errorf("want max attempt 3, got %d", got)
	}
}

// TestRenderForgejoJobLog sorts steps and lines by index, labels each step
// with its summary, and emits one message per line.
func TestRenderForgejoJobLog(t *testing.T) {
	var v forgejoJobView
	v.State.CurrentJob.Steps = []forgejoJobStep{{Summary: "Set up job"}, {Summary: "run build"}}
	v.Logs.StepsLog = []forgejoStepLog{
		{Step: 1, Lines: []forgejoLogLine{{Index: 2, Message: "second"}, {Index: 1, Message: "first"}}},
		{Step: 0, Lines: []forgejoLogLine{{Index: 1, Message: "setup"}}},
	}
	got := renderForgejoJobLog(&v)
	want := strings.Join([]string{
		"==> [step 0] Set up job",
		"setup",
		"==> [step 1] run build",
		"first",
		"second",
		"",
	}, "\n")
	if got != want {
		t.Errorf("rendered log mismatch:\n got %q\nwant %q", got, want)
	}
}

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coilysiren/coily/pkg/audit"
	"github.com/coilysiren/coily/pkg/config"
)

// seedAuditLog writes the given records to a tempfile and returns its path.
// IDs are auto-populated by Append; callers can reach back into the file
// after seeding if they need to grab the generated IDs.
func seedAuditLog(t *testing.T, records []audit.Record) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")
	w := audit.NewWriter(path)
	t.Cleanup(func() { _ = w.Close() })
	for _, rec := range records {
		if err := w.Append(rec); err != nil {
			t.Fatalf("seed Append: %v", err)
		}
	}
	return path
}

// captureStdout runs fn with a temporary stdout pipe and returns whatever
// fn printed. Restores stdout in a t.Cleanup.
func captureStdout(t *testing.T, fn func() error) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = orig })

	done := make(chan struct{})
	var buf bytes.Buffer
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	fnErr := fn()
	_ = w.Close()
	<-done
	os.Stdout = orig
	if fnErr != nil {
		t.Fatalf("fn: %v", fnErr)
	}
	return buf.String()
}

func runnerWithAuditPath(path string) *Runner {
	r := &Runner{
		Cfg: &config.Config{},
	}
	r.Cfg.Audit.LogPath = path
	r.Audit = audit.NewWriter(path)
	return r
}

func TestLoadAuditRecords_FiltersByScopeAndSince(t *testing.T) {
	scope := "/repo/a"
	other := "/repo/b"
	path := seedAuditLog(t, []audit.Record{
		{Verb: "old", Timestamp: 100, CommitScope: scope, Decision: audit.DecisionAccept},
		{Verb: "match", Timestamp: 200, CommitScope: scope, Decision: audit.DecisionAccept},
		{Verb: "wrong-scope", Timestamp: 300, CommitScope: other, Decision: audit.DecisionAccept},
		{Verb: "rejected", Timestamp: 250, CommitScope: scope, Decision: audit.DecisionReject},
	})
	got, err := loadAuditRecords(path, scope, 150)
	if err != nil {
		t.Fatalf("loadAuditRecords: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d records, want 1: %+v", len(got), got)
	}
	if got[0].Verb != "match" {
		t.Errorf("verb = %q, want match", got[0].Verb)
	}
}

func TestLoadAuditRecords_ScopeMatchesDescendants(t *testing.T) {
	parent := "/repo"
	path := seedAuditLog(t, []audit.Record{
		{Verb: "self", Timestamp: 100, CommitScope: "/repo", Decision: audit.DecisionAccept},
		{Verb: "child", Timestamp: 101, CommitScope: "/repo/a", Decision: audit.DecisionAccept},
		{Verb: "grandchild", Timestamp: 102, CommitScope: "/repo/a/b", Decision: audit.DecisionAccept},
		{Verb: "sibling-prefix", Timestamp: 103, CommitScope: "/repobar", Decision: audit.DecisionAccept},
		{Verb: "unrelated", Timestamp: 104, CommitScope: "/other", Decision: audit.DecisionAccept},
	})
	got, err := loadAuditRecords(path, parent, 0)
	if err != nil {
		t.Fatalf("loadAuditRecords: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d records, want 3: %+v", len(got), got)
	}
	verbs := map[string]bool{}
	for _, r := range got {
		verbs[r.Verb] = true
	}
	for _, want := range []string{"self", "child", "grandchild"} {
		if !verbs[want] {
			t.Errorf("missing verb %q in %+v", want, verbs)
		}
	}
	for _, bad := range []string{"sibling-prefix", "unrelated"} {
		if verbs[bad] {
			t.Errorf("unexpected verb %q in %+v", bad, verbs)
		}
	}
}

func TestLoadAuditRecords_MissingFileIsEmpty(t *testing.T) {
	got, err := loadAuditRecords(filepath.Join(t.TempDir(), "nope.jsonl"), "/x", 0)
	if err != nil {
		t.Fatalf("missing file: %v", err)
	}
	if got != nil {
		t.Errorf("got %+v, want nil", got)
	}
}

func TestRunTrailer_NoMatchingRowsEmitsNone(t *testing.T) {
	scope := "/repo/x"
	path := seedAuditLog(t, []audit.Record{
		{Verb: "v", Timestamp: 100, CommitScope: "/repo/y", Decision: audit.DecisionAccept},
	})
	r := runnerWithAuditPath(path)
	cmd := r.gitTrailerCommand()
	out := captureStdout(t, func() error {
		return cmd.Run(context.Background(), []string{"trailer", "--scope", scope, "--since", "1"})
	})
	if !strings.Contains(out, "Audit-log: none") {
		t.Errorf("output %q should contain Audit-log: none", out)
	}
}

func TestRunTrailer_EmitsOnePerRow(t *testing.T) {
	scope := "/repo/x"
	path := seedAuditLog(t, []audit.Record{
		{Verb: "v1", Timestamp: 100, CommitScope: scope, Decision: audit.DecisionAccept},
		{Verb: "v2", Timestamp: 200, CommitScope: scope, Decision: audit.DecisionAccept},
	})
	r := runnerWithAuditPath(path)
	cmd := r.gitTrailerCommand()
	out := captureStdout(t, func() error {
		return cmd.Run(context.Background(), []string{"trailer", "--scope", scope, "--since", "1"})
	})
	count := strings.Count(out, "Audit-log: coily://")
	if count != 2 {
		t.Errorf("got %d Audit-log lines, want 2:\n%s", count, out)
	}
}

func TestRunTrailer_TruncatesAtMax(t *testing.T) {
	scope := "/repo/x"
	var seed []audit.Record
	for i := int64(1); i <= 25; i++ {
		seed = append(seed, audit.Record{Verb: fmt.Sprintf("v%d", i), Timestamp: i, CommitScope: scope, Decision: audit.DecisionAccept})
	}
	path := seedAuditLog(t, seed)
	r := runnerWithAuditPath(path)
	cmd := r.gitTrailerCommand()
	out := captureStdout(t, func() error {
		return cmd.Run(context.Background(), []string{"trailer", "--scope", scope, "--since", "0", "--max", "20"})
	})
	count := strings.Count(out, "Audit-log: coily://")
	if count != 20 {
		t.Errorf("got %d coily:// lines, want 20\n%s", count, out)
	}
	if !strings.Contains(out, "5 earlier rows truncated") {
		t.Errorf("output should mention truncation:\n%s", out)
	}
}

func TestMergeTrailerBlock_AppendsAfterBody(t *testing.T) {
	body := []byte("subject\n\nlong body\n")
	got := mergeTrailerBlock(body, "Audit-log: coily://1/AAAAAAAA\n")
	want := "subject\n\nlong body\n\nAudit-log: coily://1/AAAAAAAA\n"
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestMergeTrailerBlock_JoinsExistingTrailers(t *testing.T) {
	body := []byte("subject\n\nbody\n\nCo-Authored-By: x <x@x>\n")
	got := mergeTrailerBlock(body, "Audit-log: coily://1/AAAAAAAA\n")
	want := "subject\n\nbody\n\nCo-Authored-By: x <x@x>\nAudit-log: coily://1/AAAAAAAA\n"
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestIsTrailerLine(t *testing.T) {
	yes := []string{
		"Co-Authored-By: x",
		"Signed-off-by: x",
		"Audit-log: coily://1/AAAAAAAA",
	}
	no := []string{
		"",
		"plain prose",
		": lacks-key",
		"key with spaces: yes",
		"sentence with: colon in middle",
	}
	for _, s := range yes {
		if !isTrailerLine(s) {
			t.Errorf("isTrailerLine(%q) = false, want true", s)
		}
	}
	for _, s := range no {
		if isTrailerLine(s) {
			t.Errorf("isTrailerLine(%q) = true, want false", s)
		}
	}
}

func TestRunTrailerHook_SkipsMergeSource(t *testing.T) {
	dir := t.TempDir()
	msgFile := filepath.Join(dir, "MSG")
	if err := os.WriteFile(msgFile, []byte("merge stuff\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := runnerWithAuditPath(filepath.Join(t.TempDir(), "audit.jsonl"))
	cmd := r.gitTrailerHookCommand()
	if err := cmd.Run(context.Background(), []string{"trailer-hook", msgFile, "merge"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := os.ReadFile(msgFile)
	if string(got) != "merge stuff\n" {
		t.Errorf("merge source should have left file untouched, got %q", got)
	}
}

func TestFindRecordByShortID_RoundTrip(t *testing.T) {
	scope := "/repo/x"
	path := seedAuditLog(t, []audit.Record{
		{Verb: "target", Timestamp: time.Now().Unix(), CommitScope: scope, Decision: audit.DecisionAccept, Argv: []string{"a", "b"}},
	})
	// Read back the seeded ID/short.
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	dec := json.NewDecoder(f)
	var rec audit.Record
	if err := dec.Decode(&rec); err != nil {
		t.Fatal(err)
	}
	got, err := findRecordByShortID(path, rec.Timestamp, rec.ShortID())
	if err != nil {
		t.Fatalf("findRecordByShortID: %v", err)
	}
	if got == nil || got.Verb != "target" {
		t.Fatalf("got %+v, want verb=target", got)
	}

	// Negative case: missing file should return errNotFound.
	if _, err := findRecordByShortID(filepath.Join(t.TempDir(), "nope.jsonl"), 0, ""); !errors.Is(err, errNotFound) {
		t.Errorf("missing file: got %v, want errNotFound", err)
	}
}

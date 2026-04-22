package audit_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coilysiren/coily/pkg/audit"
)

func tempWriter(t *testing.T) *audit.Writer {
	t.Helper()
	dir := t.TempDir()
	return &audit.Writer{
		Path:    filepath.Join(dir, "subdir", "audit.jsonl"),
		Now:     func() time.Time { return time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC) },
		Session: "test-session",
	}
}

func TestAppend_CreatesDirectoryAndFile(t *testing.T) {
	w := tempWriter(t)
	if err := w.Append(audit.Record{Verb: "test.verb"}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	info, err := os.Stat(w.Path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	// File perms: expect 0600 (owner rw only).
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file perm = %o, want 0600", perm)
	}
}

func TestAppend_PopulatesTimestampAndSession(t *testing.T) {
	w := tempWriter(t)
	if err := w.Append(audit.Record{Verb: "test.verb"}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	records := read(t, w.Path)
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	r := records[0]
	if r.Timestamp.IsZero() {
		t.Error("timestamp was not populated")
	}
	if r.SessionID != "test-session" {
		t.Errorf("SessionID = %q, want test-session", r.SessionID)
	}
}

func TestAppend_PreservesCallerSuppliedFields(t *testing.T) {
	w := tempWriter(t)
	custom := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	rec := audit.Record{
		Verb:      "test.verb",
		Timestamp: custom,
		SessionID: "caller-session",
	}
	if err := w.Append(rec); err != nil {
		t.Fatalf("Append: %v", err)
	}
	records := read(t, w.Path)
	if !records[0].Timestamp.Equal(custom) {
		t.Errorf("timestamp = %v, want %v (caller value should win)", records[0].Timestamp, custom)
	}
	if records[0].SessionID != "caller-session" {
		t.Errorf("SessionID = %q, want caller-session", records[0].SessionID)
	}
}

func TestAppend_Appends(t *testing.T) {
	w := tempWriter(t)
	for i, verb := range []string{"a", "b", "c"} {
		if err := w.Append(audit.Record{Verb: verb}); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}
	records := read(t, w.Path)
	if len(records) != 3 {
		t.Fatalf("got %d records, want 3", len(records))
	}
	for i, want := range []string{"a", "b", "c"} {
		if records[i].Verb != want {
			t.Errorf("record %d verb = %q, want %q", i, records[i].Verb, want)
		}
	}
}

func TestAppend_ErrorsOnUnsetPath(t *testing.T) {
	w := &audit.Writer{}
	err := w.Append(audit.Record{Verb: "x"})
	if !errors.Is(err, audit.ErrPathUnset) {
		t.Errorf("err = %v, want ErrPathUnset", err)
	}
}

func TestWrap_LogsSuccess(t *testing.T) {
	w := tempWriter(t)
	called := false
	err := w.Wrap(context.Background(), "test.verb", []string{"test", "--flag"}, func() error {
		called = true
		return nil
	})
	if err != nil {
		t.Errorf("Wrap: %v", err)
	}
	if !called {
		t.Error("fn not called")
	}
	records := read(t, w.Path)
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	r := records[0]
	if r.Verb != "test.verb" {
		t.Errorf("verb = %q", r.Verb)
	}
	if r.ExitCode != 0 {
		t.Errorf("exit_code = %d, want 0", r.ExitCode)
	}
	if r.Error != "" {
		t.Errorf("error = %q, want empty", r.Error)
	}
}

func TestWrap_LogsFailure(t *testing.T) {
	w := tempWriter(t)
	want := errors.New("boom")
	err := w.Wrap(context.Background(), "test.verb", []string{"x"}, func() error {
		return want
	})
	if !errors.Is(err, want) {
		t.Errorf("Wrap err = %v, want %v", err, want)
	}
	records := read(t, w.Path)
	if records[0].ExitCode != 1 {
		t.Errorf("exit_code = %d, want 1", records[0].ExitCode)
	}
	if !strings.Contains(records[0].Error, "boom") {
		t.Errorf("error = %q, want to contain 'boom'", records[0].Error)
	}
}

func TestNewWriter_ReadsEnv(t *testing.T) {
	t.Setenv("CLAUDE_SESSION_ID", "env-session")
	w := audit.NewWriter("/tmp/x.jsonl")
	if w.Session != "env-session" {
		t.Errorf("Session = %q, want env-session", w.Session)
	}
	if w.Now == nil {
		t.Error("Now should be populated by NewWriter")
	}
}

func TestAppend_RotatesOnSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")
	w := &audit.Writer{
		Path:       path,
		Now:        func() time.Time { return time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC) },
		MaxSizeMB:  1,
		MaxBackups: 3,
	}
	// 1MB cap. Each record is ~60 bytes; pad argv to force rotation without
	// writing millions of records.
	big := strings.Repeat("x", 4096)
	for i := 0; i < 400; i++ {
		if err := w.Append(audit.Record{Verb: "v", Argv: []string{big}}); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	// Expect at least one rotated backup alongside the active file.
	if len(entries) < 2 {
		t.Fatalf("got %d files in %s, want >=2 after rotation: %v", len(entries), dir, entries)
	}
	// And at most MaxBackups + 1 (active + backups).
	if len(entries) > 4 {
		t.Errorf("got %d files, want <=4 (MaxBackups=3 plus active): %v", len(entries), entries)
	}
}

func TestReadAll_DecodesMultipleRecords(t *testing.T) {
	input := `{"ts":"2026-04-22T12:00:00Z","verb":"a"}
{"ts":"2026-04-22T12:00:01Z","verb":"b"}
`
	records, err := audit.ReadAll(bytes.NewBufferString(input))
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
	if records[0].Verb != "a" || records[1].Verb != "b" {
		t.Errorf("verbs: %v", records)
	}
}

func read(t *testing.T, path string) []audit.Record {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	records, err := audit.ReadAll(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	return records
}

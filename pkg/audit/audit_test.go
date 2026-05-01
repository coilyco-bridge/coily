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
		Path: filepath.Join(dir, "subdir", "audit.jsonl"),
		Now:  func() time.Time { return time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC) },
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

func TestAppend_PopulatesTimestamp(t *testing.T) {
	w := tempWriter(t)
	if err := w.Append(audit.Record{Verb: "test.verb"}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	records := read(t, w.Path)
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	r := records[0]
	want := time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC).Unix()
	if r.Timestamp != want {
		t.Errorf("timestamp = %d, want %d", r.Timestamp, want)
	}
}

func TestAppend_PreservesCallerSuppliedTimestamp(t *testing.T) {
	w := tempWriter(t)
	custom := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	rec := audit.Record{
		Verb:      "test.verb",
		Timestamp: custom,
	}
	if err := w.Append(rec); err != nil {
		t.Fatalf("Append: %v", err)
	}
	records := read(t, w.Path)
	if records[0].Timestamp != custom {
		t.Errorf("timestamp = %d, want %d (caller value should win)", records[0].Timestamp, custom)
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
	err := w.Wrap(context.Background(), audit.Record{Verb: "test.verb", Argv: []string{"test", "--flag"}}, func() error {
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
	err := w.Wrap(context.Background(), audit.Record{Verb: "test.verb", Argv: []string{"x"}}, func() error {
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

func TestNewWriter_PopulatesNow(t *testing.T) {
	w := audit.NewWriter("/tmp/x.jsonl")
	if w.Now == nil {
		t.Error("Now should be populated by NewWriter")
	}
	if w.Path != "/tmp/x.jsonl" {
		t.Errorf("Path = %q, want /tmp/x.jsonl", w.Path)
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

func TestAppend_CreatesNestedParentDirectories(t *testing.T) {
	// Simulate the "parent directory does not exist" path. tempWriter already
	// nests a single "subdir" level; this test goes deeper and verifies the
	// whole chain is created with 0700 perms.
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")
	w := &audit.Writer{
		Path: filepath.Join(nested, "audit.jsonl"),
		Now:  func() time.Time { return time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC) },
	}
	if err := w.Append(audit.Record{Verb: "test.verb"}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	info, err := os.Stat(nested)
	if err != nil {
		t.Fatalf("stat %s: %v", nested, err)
	}
	if !info.IsDir() {
		t.Fatalf("%s is not a directory", nested)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf("dir perm = %o, want 0700", perm)
	}
	fi, err := os.Stat(w.Path)
	if err != nil {
		t.Fatalf("stat audit file: %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Errorf("file perm = %o, want 0600", perm)
	}
}

func TestAppend_FailsWhenParentDirUnwritable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root; permission checks are bypassed")
	}
	// Parent directory exists but lacks write permission. Append should
	// return an error (mkdir or open will fail with EACCES). The Wrap layer
	// will surface it to stderr exactly once.
	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0o500); err != nil {
		t.Fatalf("mkdir locked: %v", err)
	}
	t.Cleanup(func() {
		// Restore writability so t.TempDir cleanup succeeds.
		_ = os.Chmod(locked, 0o700)
	})
	w := &audit.Writer{
		Path: filepath.Join(locked, "audit.jsonl"),
		Now:  func() time.Time { return time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC) },
	}
	err := w.Append(audit.Record{Verb: "test.verb"})
	if err == nil {
		t.Fatal("Append on unwritable parent dir: got nil err, want permission error")
	}
	if !strings.Contains(err.Error(), "audit:") {
		t.Errorf("err = %q, want wrapped audit error", err)
	}
}

func TestPreflight_FailsLoudlyOnUnwritableDir(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root; permission checks are bypassed")
	}
	// Audit Preflight is the loud-fail path. Point Path inside a 0500 parent
	// and assert Preflight returns a wrapped error naming the path.
	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0o500); err != nil {
		t.Fatalf("mkdir locked: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })

	w := &audit.Writer{
		Path: filepath.Join(locked, "subdir", "audit.jsonl"),
		Now:  func() time.Time { return time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC) },
	}
	err := w.Preflight()
	if err == nil {
		t.Fatal("Preflight on unwritable dir: got nil err, want permission error")
	}
	if !strings.Contains(err.Error(), "audit:") {
		t.Errorf("err = %q, want wrapped audit error", err)
	}
}

func TestPreflight_CreatesDirAndFile(t *testing.T) {
	dir := t.TempDir()
	w := &audit.Writer{
		Path: filepath.Join(dir, "deep", "audit.jsonl"),
		Now:  func() time.Time { return time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC) },
	}
	if err := w.Preflight(); err != nil {
		t.Fatalf("Preflight: %v", err)
	}
	info, err := os.Stat(filepath.Dir(w.Path))
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf("dir perm = %o, want 0700", perm)
	}
}

func TestWrap_PrintsStderrOnAppendFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root; permission checks are bypassed")
	}
	// Point Path at an unwritable directory so Append fails. Capture stderr
	// across several Wrap calls; every failure prints a line so the operator
	// notices something is wrong.
	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0o500); err != nil {
		t.Fatalf("mkdir locked: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })

	origStderr := os.Stderr
	r, pw, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = pw
	t.Cleanup(func() { os.Stderr = origStderr })

	w := &audit.Writer{
		Path: filepath.Join(locked, "audit.jsonl"),
		Now:  func() time.Time { return time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC) },
	}
	for i := 0; i < 3; i++ {
		_ = w.Wrap(context.Background(), audit.Record{Verb: "v", Argv: []string{"x"}}, func() error { return nil })
	}
	if err := pw.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	var captured bytes.Buffer
	if _, err := captured.ReadFrom(r); err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	got := captured.String()
	count := strings.Count(got, "audit:")
	if count < 1 {
		t.Errorf("got %d stderr warnings, want >=1. captured:\n%s", count, got)
	}
	if !strings.Contains(got, w.Path) {
		t.Errorf("warning does not name the path %q. captured:\n%s", w.Path, got)
	}
}

func TestRecord_EgressRoundTrips(t *testing.T) {
	w := tempWriter(t)
	rec := audit.Record{
		Verb: "test.verb",
		Egress: []audit.EgressRow{
			{Host: "formulae.brew.sh", Decision: audit.EgressAllow, BytesUp: 10, BytesDown: 20, DurationMS: 3},
			{Host: "evil.example", Decision: audit.EgressDeny},
		},
	}
	if err := w.Append(rec); err != nil {
		t.Fatalf("Append: %v", err)
	}
	got := read(t, w.Path)
	if len(got) != 1 {
		t.Fatalf("got %d records, want 1", len(got))
	}
	if len(got[0].Egress) != 2 {
		t.Fatalf("got %d egress rows, want 2: %+v", len(got[0].Egress), got[0].Egress)
	}
	if got[0].Egress[0].Host != "formulae.brew.sh" || got[0].Egress[0].BytesUp != 10 {
		t.Errorf("row[0] = %+v, want host=formulae.brew.sh bytes_up=10", got[0].Egress[0])
	}
	if got[0].Egress[1].Decision != audit.EgressDeny {
		t.Errorf("row[1] decision = %q, want deny", got[0].Egress[1].Decision)
	}
}

func TestRecord_EgressOmittedWhenAbsent(t *testing.T) {
	w := tempWriter(t)
	if err := w.Append(audit.Record{Verb: "test.verb"}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	body, err := os.ReadFile(w.Path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if strings.Contains(string(body), "egress") {
		t.Errorf("egress field present when no rows; line = %q", string(body))
	}
}

func TestReadAll_DecodesMultipleRecords(t *testing.T) {
	input := `{"ts":1745323200,"verb":"a"}
{"ts":1745323201,"verb":"b"}
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
	if records[0].Timestamp != 1745323200 {
		t.Errorf("records[0].Timestamp = %d, want 1745323200", records[0].Timestamp)
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

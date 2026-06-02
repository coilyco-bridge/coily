package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coilysiren/cli-guard/audit"
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

// TestWriteRecordsYAML_RedactsSecretFlagValues pins coily#252 (forward
// action for finding #228): argv re-published into audit-show output
// goes through cross-surface redaction, so a SecureString put-parameter
// invocation does not leak the value into the YAML.
func TestWriteRecordsYAML_RedactsSecretFlagValues(t *testing.T) {
	rec := audit.Record{
		ID:        "019e0000-0000-0000-0000-000000000001",
		Timestamp: 1778744466,
		Verb:      "ops.aws.ssm.put-parameter",
		Decision:  audit.DecisionAccept,
		Argv: []string{
			"coily", "ops", "aws", "ssm", "put-parameter",
			"--name", "/tailscale/oauth/backend/client-id",
			"--value", "tskey-client-VERY_SECRET_HERE",
			"--type", "SecureString",
			"--overwrite",
		},
	}
	var buf bytes.Buffer
	if err := writeRecordsYAML(&buf, []audit.Record{rec}); err != nil {
		t.Fatalf("writeRecordsYAML: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "VERY_SECRET_HERE") {
		t.Errorf("YAML leaked plaintext secret:\n%s", out)
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Errorf("YAML missing redaction marker:\n%s", out)
	}
	// Non-secret tokens stay visible (so audit-show remains useful).
	for _, want := range []string{"put-parameter", "SecureString", "--overwrite"} {
		if !strings.Contains(out, want) {
			t.Errorf("YAML missing non-secret token %q:\n%s", want, out)
		}
	}
}

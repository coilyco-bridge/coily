package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withHome stages a tempdir as $HOME and a fixed CLAUDE_CODE_SESSION_ID
// for one test. Returns the resolved session sentinel path. Restoration
// happens via t.Cleanup; no manual teardown needed in the test body.
func withHome(t *testing.T, sessionID string) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir) // belt-and-suspenders on Windows
	t.Setenv(sessionEnvVar, sessionID)
	return filepath.Join(dir, ".coily", "audit", "sessions", sessionID, "profile")
}

func TestValidateProfileName_Accepts(t *testing.T) {
	good := []string{"mobile", "mac-tower", "windows-laptop", "web", "headless", "a", "a1", "a-b-c"}
	for _, s := range good {
		if err := validateProfileName(s); err != nil {
			t.Errorf("validateProfileName(%q) unexpected error: %v", s, err)
		}
	}
}

func TestValidateProfileName_Rejects(t *testing.T) {
	bad := []string{"", "-leading", "trailing-", "Up", "with space", "with_underscore", "with.dot", strings.Repeat("a", 41)}
	for _, s := range bad {
		if err := validateProfileName(s); err == nil {
			t.Errorf("validateProfileName(%q) expected error, got nil", s)
		}
	}
}

func TestValidateSessionID_RejectsPathEscape(t *testing.T) {
	bad := []string{"../escape", "a/b", "a\\b", "a b", "a;b", strings.Repeat("a", 129)}
	for _, s := range bad {
		if err := validateSessionID(s); err == nil {
			t.Errorf("validateSessionID(%q) expected error, got nil", s)
		}
	}
}

func TestRequireSessionID_UnsetIsUserError(t *testing.T) {
	t.Setenv(sessionEnvVar, "")
	if _, err := requireSessionID(); err == nil {
		t.Fatal("expected UserError on unset env var, got nil")
	}
}

func TestSessionShowAction_NoSentinelReportsUnset(t *testing.T) {
	withHome(t, "test-session-abc")
	var buf bytes.Buffer
	if err := sessionShowAction(&buf); err != nil {
		t.Fatalf("show: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "active_profile: (unset") {
		t.Errorf("expected unset hint in show output, got:\n%s", out)
	}
	if !strings.Contains(out, "session_id: test-session-abc") {
		t.Errorf("expected session id in output, got:\n%s", out)
	}
	if !strings.Contains(out, "data_security: max") {
		t.Errorf("expected strictest data_security tier in output, got:\n%s", out)
	}
	if !strings.Contains(out, "source: unset") {
		t.Errorf("expected source: unset in output, got:\n%s", out)
	}
}

func TestSessionShowAction_ReadsSentinel(t *testing.T) {
	sentinel := withHome(t, "sess-1")
	if err := os.MkdirAll(filepath.Dir(sentinel), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(sentinel, []byte("mobile\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	var buf bytes.Buffer
	if err := sessionShowAction(&buf); err != nil {
		t.Fatalf("show: %v", err)
	}
	if !strings.Contains(buf.String(), "active_profile: mobile") {
		t.Errorf("expected active_profile: mobile, got:\n%s", buf.String())
	}
}

func TestSessionShowAction_RejectsMalformedSentinel(t *testing.T) {
	sentinel := withHome(t, "sess-2")
	if err := os.MkdirAll(filepath.Dir(sentinel), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(sentinel, []byte("Bad Name!\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	var buf bytes.Buffer
	if err := sessionShowAction(&buf); err == nil {
		t.Fatalf("expected error on malformed sentinel, got nil; output: %s", buf.String())
	}
}

func TestSessionShowAction_EmptySentinelErrors(t *testing.T) {
	sentinel := withHome(t, "sess-empty")
	if err := os.MkdirAll(filepath.Dir(sentinel), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(sentinel, []byte(""), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	var buf bytes.Buffer
	if err := sessionShowAction(&buf); err == nil {
		t.Fatalf("expected error on empty sentinel, got nil; output: %s", buf.String())
	}
}

func TestSessionClearAction_RemovesSentinel(t *testing.T) {
	sentinel := withHome(t, "sess-3")
	if err := os.MkdirAll(filepath.Dir(sentinel), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(sentinel, []byte("mac-tower\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := sessionClearAction(); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Fatalf("sentinel still present after clear: %v", err)
	}
}

func TestSessionClearAction_NoSentinelIsNoop(t *testing.T) {
	withHome(t, "sess-4")
	if err := sessionClearAction(); err != nil {
		t.Errorf("clear on absent sentinel should be no-op, got: %v", err)
	}
}

// TestRunner_SessionUseWritesSentinelAndAudits exercises the wrapped
// verb end to end. Confirms the sentinel lands on disk with the right
// content and that the audit log captures the verb call.
func TestRunner_SessionUseWritesSentinelAndAudits(t *testing.T) {
	r := newTestRunner(t)
	sentinel := withHome(t, "sess-use-1")

	cmd := r.sessionUseCommand()
	if err := cmd.Run(context.Background(), []string{"use", "mobile"}); err != nil {
		t.Fatalf("use: %v", err)
	}
	b, err := os.ReadFile(sentinel)
	if err != nil {
		t.Fatalf("read sentinel: %v", err)
	}
	if strings.TrimSpace(string(b)) != "mobile" {
		t.Errorf("sentinel content = %q, want mobile", string(b))
	}
	if err := r.Audit.Close(); err != nil {
		t.Fatalf("audit close: %v", err)
	}
	logBytes, err := os.ReadFile(r.Audit.Path)
	if err != nil {
		t.Fatalf("read audit: %v", err)
	}
	if !bytes.Contains(logBytes, []byte("session.use")) {
		t.Errorf("audit log missing session.use:\n%s", logBytes)
	}
}

func TestRunner_SessionUseRejectsBadName(t *testing.T) {
	r := newTestRunner(t)
	withHome(t, "sess-use-2")
	cmd := r.sessionUseCommand()
	if err := cmd.Run(context.Background(), []string{"use", "Bad Name"}); err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestRunner_SessionUseRequiresOneArg(t *testing.T) {
	r := newTestRunner(t)
	withHome(t, "sess-use-3")
	cmd := r.sessionUseCommand()
	if err := cmd.Run(context.Background(), []string{"use"}); err == nil {
		t.Fatal("expected error on missing arg, got nil")
	}
}

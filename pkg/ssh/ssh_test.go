package ssh_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"

	coilyssh "github.com/coilysiren/coily/pkg/ssh"
)

// TestRun_RejectsEmptyHostUser covers the cheap input-validation paths.
// Hitting these does not require a network or a real server.
func TestRun_RejectsEmptyHost(t *testing.T) {
	c := &coilyssh.Client{KnownHostsPath: writeEmptyKnownHosts(t)}
	_, _, err := c.Run(context.Background(), "", "kai", "echo hi")
	if err == nil || !strings.Contains(err.Error(), "host is empty") {
		t.Fatalf("got %v, want host-empty error", err)
	}
}

func TestRun_RejectsEmptyUser(t *testing.T) {
	c := &coilyssh.Client{KnownHostsPath: writeEmptyKnownHosts(t)}
	_, _, err := c.Run(context.Background(), "example.invalid", "", "echo hi")
	if err == nil || !strings.Contains(err.Error(), "user is empty") {
		t.Fatalf("got %v, want user-empty error", err)
	}
}

// TestKnownHostsRequired verifies the host-key callback fails closed
// when known_hosts is missing. The threat model forbids
// ssh.InsecureIgnoreHostKey, so this is the load-bearing safety check.
func TestKnownHostsRequired(t *testing.T) {
	c := &coilyssh.Client{
		KeyPath:        writeValidKey(t),
		KnownHostsPath: filepath.Join(t.TempDir(), "does-not-exist"),
	}
	_, _, err := c.Run(context.Background(), "example.invalid:22", "kai", "echo hi")
	if err == nil {
		t.Fatal("expected error for missing known_hosts")
	}
	if !errors.Is(err, coilyssh.ErrNoKnownHosts) {
		t.Fatalf("err = %v, want ErrNoKnownHosts", err)
	}
}

// TestNoAuthAvailable verifies we error out cleanly when neither
// ssh-agent nor a key file is reachable. We unset SSH_AUTH_SOCK so the
// agent path is unavailable, leave KeyPath empty, and supply a valid
// known_hosts so we exercise only the auth branch.
func TestNoAuthAvailable(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	c := &coilyssh.Client{KnownHostsPath: writeEmptyKnownHosts(t)}
	_, _, err := c.Run(context.Background(), "example.invalid:22", "kai", "echo hi")
	if !errors.Is(err, coilyssh.ErrNoAuth) {
		t.Fatalf("err = %v, want ErrNoAuth", err)
	}
}

// TestKeyPath_ReadFailure surfaces a clean error when the configured
// key file cannot be read. This catches typos in coily config without
// needing a live remote.
func TestKeyPath_ReadFailure(t *testing.T) {
	c := &coilyssh.Client{
		KeyPath:        filepath.Join(t.TempDir(), "missing-key"),
		KnownHostsPath: writeEmptyKnownHosts(t),
	}
	_, _, err := c.Run(context.Background(), "example.invalid:22", "kai", "echo hi")
	if err == nil {
		t.Fatal("expected error for missing key file")
	}
	if !strings.Contains(err.Error(), "load key") {
		t.Fatalf("err = %v, want a load-key error", err)
	}
}

// TestKeyPath_BadKey covers the "file exists but is not a private key"
// path through ssh.ParsePrivateKey.
func TestKeyPath_BadKey(t *testing.T) {
	keyPath := filepath.Join(t.TempDir(), "garbage")
	if err := os.WriteFile(keyPath, []byte("not a key"), 0o600); err != nil {
		t.Fatal(err)
	}
	c := &coilyssh.Client{
		KeyPath:        keyPath,
		KnownHostsPath: writeEmptyKnownHosts(t),
	}
	_, _, err := c.Run(context.Background(), "example.invalid:22", "kai", "echo hi")
	if err == nil {
		t.Fatal("expected parse error for garbage key")
	}
}

// TestCopyTo_MissingLocalFile covers the open-local failure path without
// hitting the wire. The dial still has to succeed to reach the file open,
// so we feed a canceled ctx; ctx plumbing reaches dial() first and fails
// there. That's enough to exercise the CopyTo entry point; live SFTP
// coverage needs an integration harness and is out of scope here.
func TestCopyTo_CancelsBeforeDial(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := &coilyssh.Client{
		KeyPath:        writeValidKey(t),
		KnownHostsPath: writeEmptyKnownHosts(t),
	}
	err := c.CopyTo(ctx, "127.0.0.1:1", "kai", "/tmp/a", "/tmp/b")
	if err == nil {
		t.Fatal("expected error from canceled ctx")
	}
	if !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

// TestCopyTo_EmptyHostUser pins the input-validation through dial().
func TestCopyTo_EmptyHost(t *testing.T) {
	c := &coilyssh.Client{KnownHostsPath: writeEmptyKnownHosts(t)}
	err := c.CopyTo(context.Background(), "", "kai", "/tmp/a", "/tmp/b")
	if err == nil || !strings.Contains(err.Error(), "host is empty") {
		t.Fatalf("got %v, want host-empty error", err)
	}
}

// TestContextCanceled_BeforeDial surfaces ctx.Err() rather than a
// network failure when the caller cancels up front. Demonstrates the
// context plumbing reaches DialContext.
func TestContextCanceled_BeforeDial(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	keyPath := writeValidKey(t)
	c := &coilyssh.Client{
		KeyPath:        keyPath,
		KnownHostsPath: writeEmptyKnownHosts(t),
	}
	_, _, err := c.Run(ctx, "127.0.0.1:1", "kai", "echo hi")
	if err == nil {
		t.Fatal("expected error from canceled ctx")
	}
	if !errors.Is(err, context.Canceled) {
		// On some platforms the canceled ctx surfaces as a wrapped dial
		// error. Accept either, as long as the underlying cause is the
		// canceled context.
		if !strings.Contains(err.Error(), "canceled") {
			t.Fatalf("err = %v, want context.Canceled", err)
		}
	}
}

// writeEmptyKnownHosts creates a known_hosts file with no entries. This
// is enough to satisfy the load step (file exists, parses as zero
// entries) without permitting any specific host. Real connections
// against this file fail the host-key check, which is exactly what we
// want for tests that should never reach the wire.
func writeEmptyKnownHosts(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "known_hosts")
	if err := os.WriteFile(p, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// writeValidKey emits a freshly-generated ed25519 private key in
// OpenSSH PEM format. Used by tests that need ParsePrivateKey to
// succeed so we can exercise code paths past authMethods. The key is
// not used to authenticate anywhere.
func writeValidKey(t *testing.T) string {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(p, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

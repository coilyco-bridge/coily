package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coilysiren/coily/pkg/config"
)

func TestLoad_ParsesEmbeddedConfig(t *testing.T) {
	c, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c == nil {
		t.Fatal("Load returned nil config")
	}
	if c.Loaded.IsZero() {
		t.Error("Loaded timestamp was not set")
	}
}

func TestLoad_HasExpectedFields(t *testing.T) {
	c, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// These are the fields every coily config must have populated. Values can
	// be whatever the committed config.yaml says, but the fields must exist.
	if c.KaiServer.TailscaleHost == "" {
		t.Error("kai_server.tailscale_host is empty")
	}
	if c.KaiServer.SSHUser == "" {
		t.Error("kai_server.ssh_user is empty")
	}
}

// withIsolatedHome points $HOME at a temp dir, chdir's into another temp dir,
// and resets the slug cache. Returns the global dir path the caller can write
// into to seed a global config.yaml.
func withIsolatedHome(t *testing.T) (homeDir, cwdDir string) {
	t.Helper()
	homeDir = t.TempDir()
	cwdDir = t.TempDir()
	t.Setenv("HOME", homeDir)
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(cwdDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
	config.ResetRepoSlugCacheForTest()
	t.Cleanup(config.ResetRepoSlugCacheForTest)
	return homeDir, cwdDir
}

func TestLoad_DefaultPathsFallToHome(t *testing.T) {
	home, _ := withIsolatedHome(t)
	c, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	wantAuditPrefix := filepath.Join(home, ".coily", "audit")
	if !strings.HasPrefix(c.Audit.LogPath, wantAuditPrefix) {
		t.Errorf("Audit.LogPath = %q, want prefix %q", c.Audit.LogPath, wantAuditPrefix)
	}
}

func TestLoad_GlobalOverlay(t *testing.T) {
	home, _ := withIsolatedHome(t)
	dir := filepath.Join(home, ".coily")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := `aws:
  profile: from-global
audit:
  max_size_mb: 99
`
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(body), 0o600); err != nil {
		t.Fatalf("write global: %v", err)
	}
	c, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.AWS.Profile != "from-global" {
		t.Errorf("AWS.Profile = %q, want from-global", c.AWS.Profile)
	}
	if c.Audit.MaxSizeMB != 99 {
		t.Errorf("Audit.MaxSizeMB = %d, want 99", c.Audit.MaxSizeMB)
	}
}

func TestLoad_LocalWinsOverGlobal(t *testing.T) {
	home, cwd := withIsolatedHome(t)
	gdir := filepath.Join(home, ".coily")
	if err := os.MkdirAll(gdir, 0o700); err != nil {
		t.Fatalf("mkdir global: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gdir, "config.yaml"), []byte("aws:\n  profile: from-global\n"), 0o600); err != nil {
		t.Fatalf("write global: %v", err)
	}
	ldir := filepath.Join(cwd, ".coily")
	if err := os.MkdirAll(ldir, 0o700); err != nil {
		t.Fatalf("mkdir local: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ldir, "config.yaml"), []byte("aws:\n  profile: from-local\n"), 0o600); err != nil {
		t.Fatalf("write local: %v", err)
	}
	c, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.AWS.Profile != "from-local" {
		t.Errorf("AWS.Profile = %q, want from-local (local must override global)", c.AWS.Profile)
	}
}

func TestLoad_LocalOnly(t *testing.T) {
	_, cwd := withIsolatedHome(t)
	ldir := filepath.Join(cwd, ".coily")
	if err := os.MkdirAll(ldir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := `kai_server:
  ssh_user: alice
`
	if err := os.WriteFile(filepath.Join(ldir, "config.yaml"), []byte(body), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	c, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.KaiServer.SSHUser != "alice" {
		t.Errorf("KaiServer.SSHUser = %q, want alice", c.KaiServer.SSHUser)
	}
}

func TestLoad_ExpandsTildeInOverride(t *testing.T) {
	home, _ := withIsolatedHome(t)
	dir := filepath.Join(home, ".coily")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := "audit:\n  log_path: ~/custom-audit.jsonl\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(body), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	c, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := filepath.Join(home, "custom-audit.jsonl")
	if c.Audit.LogPath != want {
		t.Errorf("Audit.LogPath = %q, want %q", c.Audit.LogPath, want)
	}
}

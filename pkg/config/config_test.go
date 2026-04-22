package config_test

import (
	"testing"
	"time"

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
	if c.Tokens.DefaultTTL == 0 {
		t.Error("tokens.default_ttl is zero")
	}
	// DefaultTTL should parse as a duration.
	if c.Tokens.DefaultTTL < time.Second {
		t.Errorf("tokens.default_ttl = %v, want >= 1s", c.Tokens.DefaultTTL)
	}
}

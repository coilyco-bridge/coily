// Package config loads the coily configuration from a yaml file embedded
// into the binary at build time. See docs/threat-model.md — config is NOT
// read from disk at runtime; every config change is a rebuild + sudo install.
package config

import (
	_ "embed"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

//go:embed config.yaml
var embeddedConfigBytes []byte

type Config struct {
	KaiServer KaiServer `yaml:"kai_server"`
	Audit     Audit     `yaml:"audit"`
	AWS       AWS       `yaml:"aws"`
	Tokens    Tokens    `yaml:"tokens"`
	Eco       Eco       `yaml:"eco"`
	Loaded    time.Time `yaml:"-"`
}

// Eco is the local-side config for `coily eco world` verbs. configs_dir
// points at a checkout of the eco-configs repo (the same one
// eco-cycle-prep/worldgen.py operates on). Optional: if empty, world
// verbs require --configs-dir explicitly.
type Eco struct {
	ConfigsDir string `yaml:"configs_dir"`
}

type KaiServer struct {
	TailscaleHost string `yaml:"tailscale_host"`
	SSHUser       string `yaml:"ssh_user"`
}

type Audit struct {
	LogPath    string `yaml:"log_path"`
	MaxSizeMB  int    `yaml:"max_size_mb"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAgeDays int    `yaml:"max_age_days"`
	Compress   bool   `yaml:"compress"`
}

type AWS struct {
	Profile string `yaml:"profile"`
}

type Tokens struct {
	IssuerKeyPath string        `yaml:"issuer_key_path"`
	DefaultTTL    time.Duration `yaml:"default_ttl"`
}

// Load parses the embedded config yaml. Safe to call once at startup.
func Load() (*Config, error) {
	var c Config
	if err := yaml.Unmarshal(embeddedConfigBytes, &c); err != nil {
		return nil, fmt.Errorf("parse embedded config: %w", err)
	}
	c.Loaded = time.Now()
	return &c, nil
}

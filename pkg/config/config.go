// Package config loads the coily configuration in three layers. From lowest
// to highest precedence: an embedded yaml that ships with the binary,
// ~/.coily/config.yaml (the global overlay), and ./.coily/config.yaml (the
// per-repo local overlay). Local always wins on a per-key basis. See
// docs/threat-model.md for the broader picture.
//
// The embedded layer guarantees coily can boot with no on-disk state at all.
// The global layer holds Kai's personal defaults (audit rotation knobs, AWS
// profile name). The local layer lets a repo carry its own overrides into a
// fresh checkout without touching ~/.
package config

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
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

// Load returns the layered config. Embedded is parsed first, then
// ~/.coily/config.yaml is overlaid on top, then ./.coily/config.yaml. Any
// missing layer is silently skipped. Path defaults (audit log, issuer key)
// are filled in from the homedir-based defaults when the merged config left
// them blank, and any "~/" prefix is expanded to an absolute path.
func Load() (*Config, error) {
	var c Config
	if err := yaml.Unmarshal(embeddedConfigBytes, &c); err != nil {
		return nil, fmt.Errorf("parse embedded config: %w", err)
	}

	globalPath, gerr := GlobalConfigPath()
	if gerr == nil {
		if err := overlayFromFile(&c, globalPath); err != nil {
			return nil, fmt.Errorf("global config %s: %w", globalPath, err)
		}
	}
	localPath, lerr := LocalConfigPath()
	if lerr == nil {
		if err := overlayFromFile(&c, localPath); err != nil {
			return nil, fmt.Errorf("local config %s: %w", localPath, err)
		}
	}

	if err := applyDefaults(&c); err != nil {
		return nil, err
	}
	c.Loaded = time.Now()
	return &c, nil
}

// overlayFromFile reads path (if it exists) and merges the parsed values onto
// dst. yaml.Unmarshal into an existing struct already does field-level merge -
// fields absent from the file keep their previous value, fields present
// overwrite. That is exactly the per-key precedence rule we want.
func overlayFromFile(dst *Config, path string) error {
	b, err := os.ReadFile(path) // #nosec G304 -- caller-controlled config path is the intended input
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if err := yaml.Unmarshal(b, dst); err != nil {
		return fmt.Errorf("parse: %w", err)
	}
	return nil
}

// applyDefaults fills in any path field that the embedded + overlay layers
// left blank, and expands ~/ on every path field. Defaults are computed from
// $HOME, so the binary works on a fresh laptop with no on-disk config.
func applyDefaults(c *Config) error {
	if c.Audit.LogPath == "" {
		p, err := DefaultAuditPath()
		if err != nil {
			return err
		}
		c.Audit.LogPath = p
	} else {
		c.Audit.LogPath = expandHome(c.Audit.LogPath)
	}
	if c.Tokens.IssuerKeyPath == "" {
		p, err := DefaultIssuerKeyPath()
		if err != nil {
			return err
		}
		c.Tokens.IssuerKeyPath = p
	} else {
		c.Tokens.IssuerKeyPath = expandHome(c.Tokens.IssuerKeyPath)
	}
	return nil
}

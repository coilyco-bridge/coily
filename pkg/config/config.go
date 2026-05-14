// Package config loads the coily configuration in three layers. From lowest
// to highest precedence: a Go-literal default baked into the binary,
// ~/.coily/config.yaml (the global overlay), and ./.coily/config.yaml (the
// per-repo local overlay). Local always wins on a per-key basis. See
// SECURITY.md for the broader picture.
//
// The default layer guarantees coily can boot with no on-disk state at all.
// The global layer holds Kai's personal defaults (audit rotation knobs, AWS
// profile name). The local layer lets a repo carry its own overrides into a
// fresh checkout without touching ~/.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// defaults returns the baseline config used when no overlay layer fills a
// field. Mirrors what the committed config.example.yaml would parse to.
// LogPath stays empty so applyDefaults() falls to the per-repo
// DefaultAuditPath under ~/.coily/audit/.
func defaults() Config {
	return Config{
		KaiServer: KaiServer{
			TailscaleHost: "kai-server",
			SSHUser:       "kai",
		},
		Audit: Audit{
			MaxSizeMB:  10,
			MaxBackups: 10,
			MaxAgeDays: 30,
			Compress:   false,
		},
		AWS: AWS{Profile: "default"},
		Eco: Eco{
			ServerDir: "/home/kai/Steam/steamapps/common/EcoServer",
		},
		Factorio: Factorio{
			ServerDir: "/home/kai/Steam/steamapps/common/FactorioServer",
		},
	}
}

// Config is the merged result of the three layered config sources. Loaded
// holds the moment Load returned, so callers can decide whether to refresh.
type Config struct {
	KaiServer KaiServer `yaml:"kai_server"`
	Audit     Audit     `yaml:"audit"`
	AWS       AWS       `yaml:"aws"`
	Eco       Eco       `yaml:"eco"`
	Factorio  Factorio  `yaml:"factorio"`
	Loaded    time.Time `yaml:"-"`
}

// Eco is the local-side + remote-side config for `coily eco` verbs.
// configs_dir points at a local checkout of the eco-configs repo (the
// same one eco-cycle-prep/worldgen.py operates on); used by `eco world`.
// server_dir is the absolute path to the Eco dedicated server install on
// kai-server; used by `eco mod install` as the root for Mods/<Name>/.
type Eco struct {
	ConfigsDir string `yaml:"configs_dir"`
	ServerDir  string `yaml:"server_dir"`
}

// Factorio is the remote-side config for `coily gaming factorio` verbs.
// server_dir is the absolute path to the dedicated server install on
// kai-server (the directory that contains saves/, mods/, bin/x64/factorio).
// Defaults to the Steam install path used by factorio-server-{pre,start}.sh.
type Factorio struct {
	ServerDir string `yaml:"server_dir"`
}

// KaiServer is the connection config for the home server that `coily ssh`
// and the gaming/eco/factorio verbs target.
type KaiServer struct {
	TailscaleHost string `yaml:"tailscale_host"`
	SSHUser       string `yaml:"ssh_user"`
	// SSHKeyPath is an optional path to a PEM private key used for
	// ssh/sftp auth against kai-server. ~ is expanded. When empty,
	// coily falls back to ssh-agent (SSH_AUTH_SOCK). On Windows the
	// MSYS/cygwin agent is not reachable from the Windows-native
	// binary, so an explicit key path is the working path there.
	SSHKeyPath string `yaml:"ssh_key_path"`
}

// Audit controls where the JSONL audit log lives and how lumberjack rotates
// it. LogPath defaults to ~/.coily/audit/coily.jsonl when left blank.
type Audit struct {
	LogPath    string `yaml:"log_path"`
	MaxSizeMB  int    `yaml:"max_size_mb"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAgeDays int    `yaml:"max_age_days"`
	Compress   bool   `yaml:"compress"`
	// ProfileAware is the per-host feature flag for the per-session
	// lockdown-profile evaluator (coilysiren/coily#150). Default false.
	// When true, audit rows carry a profile_decision block capturing
	// the resolved Coordinate and source. Phase 4 plumbing only;
	// the evaluator does not yet deny. Retired in phase 6.
	ProfileAware bool `yaml:"profile_aware"`
}

// AWS holds the named profile that `coily ops aws` passes through to the
// underlying aws CLI. Empty falls back to the aws CLI's own resolution.
type AWS struct {
	Profile string `yaml:"profile"`
}

// Load returns the layered config. The Go-literal default is the base, then
// ~/.coily/config.yaml is overlaid on top, then ./.coily/config.yaml. Any
// missing layer is silently skipped. The audit log path default is filled
// in from the homedir when the merged config left it blank, and any "~/"
// prefix is expanded to an absolute path.
func Load() (*Config, error) {
	c := defaults()

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
//
// COILY_AUDIT_LOG, if set, wins over both file config and the per-repo
// default. Documented as the orchestrator-friendly path override - lets
// an external runner point coily at its own log dir without writing a
// config file.
func applyDefaults(c *Config) error {
	if envPath := os.Getenv("COILY_AUDIT_LOG"); envPath != "" {
		c.Audit.LogPath = expandHome(envPath)
		return nil
	}
	if c.Audit.LogPath == "" {
		p, err := DefaultAuditPath()
		if err != nil {
			return err
		}
		c.Audit.LogPath = p
	} else {
		c.Audit.LogPath = expandHome(c.Audit.LogPath)
	}
	return nil
}

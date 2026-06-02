package main

import (
	"fmt"
	"os"
	"time"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/config"
)

// auditLogEnvVar is the orchestrator-friendly path override for the
// audit log. Set it and the resolved Audit.LogPath comes out the value
// you provided (with ~/ expanded), regardless of what the layered
// config files say. Documented as the env-var path for external runners
// to point coily at their own log dir without writing a config file.
const auditLogEnvVar = "COILY_AUDIT_LOG"

// Config is the merged result of the three layered config sources:
// the Go-literal defaults in this file, ~/.coily/config.yaml (the
// global overlay), and ./.coily/config.yaml (the per-repo local
// overlay). Local always wins on a per-key basis. Loaded holds the
// moment LoadConfig returned, so callers can decide whether to refresh.
type Config struct {
	KaiServer KaiServer    `yaml:"kai_server"`
	Audit     config.Audit `yaml:"audit"`
	AWS       AWS          `yaml:"aws"`
	Eco       Eco          `yaml:"eco"`
	Factorio  Factorio     `yaml:"factorio"`
	Forgejo   Forgejo      `yaml:"forgejo"`
	Loaded    time.Time    `yaml:"-"`
}

// Forgejo is the config for `coily ops forgejo issue ...` API verbs that
// hit the forgejo HTTP API (distinct from the in-pod `forgejo` CLI verbs
// in ops_forgejo.go, which go through k3s kubectl exec). BaseURL is the
// forgejo deployment's public origin; SSMTokenPath is the AWS SSM path
// holding the bearer API token (scope `write:repository`). See
// coilysiren/coily#69 for the rollout.
type Forgejo struct {
	BaseURL      string `yaml:"base_url"`
	SSMTokenPath string `yaml:"ssm_token_path"`
}

// defaultForgejo returns the embedded Forgejo defaults. The //nolint sits
// on one line by splitting the SSM path assignment out of the literal.
func defaultForgejo() Forgejo {
	f := Forgejo{BaseURL: "https://forgejo.coilysiren.me"}
	f.SSMTokenPath = "/forgejo/api-token" //nolint:gosec // SSM path, not a credential
	return f
}

// KaiServer identifies the home server that the gaming / eco / factorio /
// forgejo / systemd verbs target. Those verbs run only on kai-server
// itself now (the SSH transport was removed), so TailscaleHost is used
// solely to confirm coily is running on that host via hostIsLocal.
type KaiServer struct {
	TailscaleHost string `yaml:"tailscale_host"`
}

// AWS holds the named profile that `coily ops aws` passes through to
// the underlying aws CLI. Empty falls back to the aws CLI's own
// resolution.
type AWS struct {
	Profile string `yaml:"profile"`
}

// Eco is the local-side + remote-side config for `coily eco` verbs.
// ConfigsDir points at a local checkout of the eco-configs repo (the
// same one eco-cycle-prep/worldgen.py operates on); used by `eco world`.
// ServerDir is the absolute path to the Eco dedicated server install
// on kai-server; used by `eco mod install` as the root for Mods/<Name>/.
type Eco struct {
	ConfigsDir string `yaml:"configs_dir"`
	ServerDir  string `yaml:"server_dir"`
}

// Factorio is the remote-side config for `coily gaming factorio` verbs.
// ServerDir is the absolute path to the dedicated server install on
// kai-server (the directory that contains saves/, mods/, bin/x64/factorio).
// Defaults to the Steam install path used by factorio-server-{pre,start}.sh.
type Factorio struct {
	ServerDir string `yaml:"server_dir"`
}

// defaultConfig returns the baseline coily config used when no overlay
// layer fills a field. Mirrors what the committed config.example.yaml
// would parse to. The Audit.LogPath stays empty so applyConfigDefaults
// falls to the per-repo DefaultAuditPath under ~/.coily/audit/.
func defaultConfig() Config {
	return Config{
		KaiServer: KaiServer{
			TailscaleHost: "kai-server",
		},
		Audit: config.Audit{
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
		Forgejo: defaultForgejo(),
	}
}

// LoadConfig returns the layered coily config. The Go-literal default
// is the base, then ~/.coily/config.yaml is overlaid on top, then
// ./.coily/config.yaml. Any missing layer is silently skipped. The
// audit log path default is filled in from the homedir when the merged
// config left it blank, and any "~/" prefix is expanded to an absolute
// path. COILY_AUDIT_LOG, if set, wins over both file config and the
// per-repo default.
func LoadConfig() (*Config, error) {
	c := defaultConfig()

	globalPath, gerr := config.GlobalConfigPath()
	if gerr == nil {
		if err := config.OverlayFile(&c, globalPath); err != nil {
			return nil, fmt.Errorf("global config %s: %w", globalPath, err)
		}
	}
	localPath, lerr := config.LocalConfigPath()
	if lerr == nil {
		if err := config.OverlayFile(&c, localPath); err != nil {
			return nil, fmt.Errorf("local config %s: %w", localPath, err)
		}
	}

	if err := applyConfigDefaults(&c); err != nil {
		return nil, err
	}
	c.Loaded = time.Now()
	return &c, nil
}

// applyConfigDefaults fills in the audit log path that the embedded and
// overlay layers left blank, and expands ~/ on the path field.
func applyConfigDefaults(c *Config) error {
	if envPath := os.Getenv(auditLogEnvVar); envPath != "" {
		c.Audit.LogPath = config.ExpandHome(envPath)
		return nil
	}
	if c.Audit.LogPath == "" {
		p, err := config.DefaultAuditPath()
		if err != nil {
			return err
		}
		c.Audit.LogPath = p
	} else {
		c.Audit.LogPath = config.ExpandHome(c.Audit.LogPath)
	}
	return nil
}

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coilysiren/coily/pkg/audit"
	"github.com/coilysiren/coily/pkg/auth"
	"github.com/coilysiren/coily/pkg/config"
	"github.com/coilysiren/coily/pkg/policy"
	"github.com/coilysiren/coily/pkg/shell"
	coilyssh "github.com/coilysiren/coily/pkg/ssh"
)

// expandHome turns a leading "~/" or "~" into the user's home directory.
// Returns the input unchanged if it doesn't start with "~".
func expandHome(p string) string {
	if p == "" || !strings.HasPrefix(p, "~") {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if p == "~" {
		return home
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:])
	}
	return p
}

// defaultStatePath returns ~/.local/state/coily/<name>. Used when config is
// empty so coily always has somewhere to write state.
func defaultStatePath(name string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "state", "coily", name)
}

// Runner owns the audit writer, token verifier, shell runner, ssh client,
// and embedded config. Constructed once in main() and threaded into every
// cli.Command action via methods on this struct. Tests construct a Runner
// directly with fakes for Audit, Verifier, or SSH.
//
// The fields are interfaces (or interface-shaped) where they need to be
// swappable. Cfg is the loaded *config.Config. Runner is the shell exec
// gateway. Audit is the JSONL writer. Verifier is what policy.Enforce calls
// to validate confirmation tokens for mutating verbs. SSH is the SDK-backed
// ssh client (replaces shelling out to /usr/bin/ssh, see pkg/ssh).
type Runner struct {
	Cfg      *config.Config
	Runner   *shell.Runner
	Audit    *audit.Writer
	Verifier policy.TokenVerifier
	SSH      *coilyssh.Client
}

// NewRunner builds the production Runner from embedded config. Exits the
// process if the config does not parse, since nothing else can work without
// it. Equivalent to the old getRuntime() singleton, but called explicitly
// from main() exactly once.
func NewRunner() *Runner {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "coily: fatal: cannot load embedded config: %v\n", err)
		os.Exit(2)
	}

	auditPath := expandHome(cfg.Audit.LogPath)
	if auditPath == "" {
		auditPath = defaultStatePath("audit.jsonl")
	}
	issuerKey := expandHome(cfg.Tokens.IssuerKeyPath)
	if issuerKey == "" {
		issuerKey = defaultStatePath("token-issuer.key")
	}

	aw := audit.NewWriter(auditPath)
	aw.MaxSizeMB = cfg.Audit.MaxSizeMB
	aw.MaxBackups = cfg.Audit.MaxBackups
	aw.MaxAgeDays = cfg.Audit.MaxAgeDays
	aw.Compress = cfg.Audit.Compress

	return &Runner{
		Cfg:      cfg,
		Runner:   &shell.Runner{Stdout: os.Stdout, Stderr: os.Stderr, Stdin: os.Stdin},
		Audit:    aw,
		Verifier: auth.NewIssuer(issuerKey),
		// SSH wraps golang.org/x/crypto/ssh. Auth uses ssh-agent (KeyPath
		// empty), host keys verified against ~/.ssh/known_hosts. See
		// pkg/ssh/ssh.go.
		SSH: &coilyssh.Client{},
	}
}

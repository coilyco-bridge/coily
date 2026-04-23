package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/coilysiren/coily/pkg/audit"
	"github.com/coilysiren/coily/pkg/auth"
	"github.com/coilysiren/coily/pkg/config"
	"github.com/coilysiren/coily/pkg/policy"
	"github.com/coilysiren/coily/pkg/shell"
	coilyssh "github.com/coilysiren/coily/pkg/ssh"
)

// Runner owns the audit writer, token verifier, shell runner, ssh client,
// and loaded config. Constructed once in main() and threaded into every
// cli.Command action via methods on this struct. Tests construct a Runner
// directly with fakes for Audit, Verifier, or SSH.
//
// Cfg is the layered *config.Config (embedded defaults, overlaid by
// ~/.coily/config.yaml, then ./.coily/config.yaml). Path fields like
// Audit.LogPath and Tokens.IssuerKeyPath are already resolved to absolute
// paths by pkg/config at load time, so this struct does no path expansion.
type Runner struct {
	Cfg      *config.Config
	Runner   *shell.Runner
	Audit    *audit.Writer
	Verifier policy.TokenVerifier
	SSH      *coilyssh.Client
}

// NewRunner builds the production Runner from layered config. Exits the
// process if the config does not parse, if the audit directory is not
// writable, or if the token issuer directory cannot be created. Better to
// fail loudly at startup than silently drop audit records per call.
func NewRunner() *Runner {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "coily: fatal: cannot load config: %v\n", err)
		os.Exit(2)
	}

	aw := audit.NewWriter(cfg.Audit.LogPath)
	aw.MaxSizeMB = cfg.Audit.MaxSizeMB
	aw.MaxBackups = cfg.Audit.MaxBackups
	aw.MaxAgeDays = cfg.Audit.MaxAgeDays
	aw.Compress = cfg.Audit.Compress
	// Loud-fail if the configured audit directory is not writable. Better
	// than silently dropping records over the lifetime of the process.
	if err := aw.Preflight(); err != nil {
		fmt.Fprintf(os.Stderr, "coily: fatal: %v\n", err)
		os.Exit(2)
	}

	// Make sure the issuer key parent dir exists with 0700 before any auth
	// verb tries to read or write the key. The auth package does the same
	// mkdir on demand, but doing it here keeps the failure mode uniform
	// with the audit dir above.
	if dir := filepath.Dir(cfg.Tokens.IssuerKeyPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			fmt.Fprintf(os.Stderr, "coily: fatal: token issuer dir %s: %v\n", dir, err)
			os.Exit(2)
		}
	}

	return &Runner{
		Cfg:      cfg,
		Runner:   &shell.Runner{Stdout: os.Stdout, Stderr: os.Stderr, Stdin: os.Stdin},
		Audit:    aw,
		Verifier: auth.NewIssuer(cfg.Tokens.IssuerKeyPath),
		// SSH wraps golang.org/x/crypto/ssh. Auth uses ssh-agent (KeyPath
		// empty), host keys verified against ~/.ssh/known_hosts. See
		// pkg/ssh/ssh.go.
		SSH: &coilyssh.Client{},
	}
}

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/coilysiren/coily/pkg/audit"
	"github.com/coilysiren/coily/pkg/config"
	"github.com/coilysiren/coily/pkg/shell"
	coilyssh "github.com/coilysiren/coily/pkg/ssh"
	"github.com/coilysiren/coily/pkg/telemetry"
)

// Runner owns the audit writer, shell runner, ssh client, and loaded config.
// Constructed once in main() and threaded into every cli.Command action via
// methods on this struct. Tests construct a Runner directly with fakes for
// Audit or SSH.
//
// Cfg is the layered *config.Config (embedded defaults, overlaid by
// ~/.coily/config.yaml, then ./.coily/config.yaml). Path fields like
// Audit.LogPath are already resolved to absolute paths by pkg/config at
// load time, so this struct does no path expansion.
type Runner struct {
	Cfg    *config.Config
	Runner *shell.Runner
	Audit  *audit.Writer
	SSH    *coilyssh.Client
}

// NewRunner builds the production Runner from layered config. Exits the
// process if the config does not parse or if the audit directory is not
// writable. Better to fail loudly at startup than silently drop audit
// records per call.
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
	aw.OnRecord = func(r audit.Record) {
		telemetry.LogInvocation(
			r.Verb,
			len(r.Argv),
			r.ExitCode,
			time.Duration(r.DurationMS)*time.Millisecond,
			r.Error,
		)
	}
	// Loud-fail if the configured audit directory is not writable. Better
	// than silently dropping records over the lifetime of the process.
	if err := aw.Preflight(); err != nil {
		fmt.Fprintf(os.Stderr, "coily: fatal: %v\n", err)
		os.Exit(2)
	}

	return &Runner{
		Cfg: cfg,
		Runner: &shell.Runner{
			Stdout: os.Stdout,
			Stderr: os.Stderr,
			Stdin:  os.Stdin,
		},
		Audit: aw,
		// SSH wraps golang.org/x/crypto/ssh. When kai_server.ssh_key_path is
		// set (e.g. on Windows where the MSYS agent is unreachable from the
		// native Windows binary), auth uses that key. Otherwise it falls back
		// to ssh-agent (SSH_AUTH_SOCK). Host keys verified against
		// ~/.ssh/known_hosts either way. See pkg/ssh/ssh.go.
		SSH: &coilyssh.Client{KeyPath: expandTilde(cfg.KaiServer.SSHKeyPath)},
	}
}

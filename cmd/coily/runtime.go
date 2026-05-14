package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/coilysiren/cli-guard/audit"
	"github.com/coilysiren/cli-guard/shell"
	"github.com/coilysiren/cli-guard/verb"
	"github.com/coilysiren/coily/pkg/config"
	"github.com/coilysiren/coily/pkg/decision"
	coilyssh "github.com/coilysiren/coily/pkg/ssh"
	"github.com/urfave/cli/v3"
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

// WrapVerb wraps spec into a cli.ActionFunc. When audit.profile_aware
// is true on this Runner's config, OnEvaluate is injected so every
// audit row carries the resolved profile decision. Phase 4 of #150:
// the injected evaluator always returns Allowed=true; phase 5 puts
// per-axis decision logic behind this single chokepoint. Callers
// already passing OnEvaluate keep their value; nil callers get the
// runtime-managed evaluator.
//
// The writer argument is accepted (and forwarded to verb.Wrap) so the
// call shape `r.WrapVerb(spec, r.Audit)` is a single-token swap from
// `verb.Wrap(spec, r.Audit)`. The argument is redundant given the
// receiver carries r.Audit; phase 6 cleanup retires it once every
// call site is on this helper.
func (r *Runner) WrapVerb(spec verb.Spec, writer *audit.Writer) cli.ActionFunc {
	if r != nil && r.Cfg != nil && r.Cfg.Audit.ProfileAware && spec.OnEvaluate == nil {
		spec.OnEvaluate = func(_ context.Context, _ *cli.Command) (*audit.ProfileDecision, error) {
			sid := strings.TrimSpace(os.Getenv(sessionEnvVar))
			active := ""
			if sid != "" {
				if name, err := readSessionProfileName(sid); err == nil {
					active = name
				}
			}
			return decision.Evaluate(active)
		}
	}
	return verb.Wrap(spec, writer)
}

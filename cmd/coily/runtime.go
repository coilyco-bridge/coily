package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coilysiren/cli-guard/audit"
	"github.com/coilysiren/cli-guard/decision"
	"github.com/coilysiren/cli-guard/shell"
	coilyssh "github.com/coilysiren/cli-guard/ssh"
	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// Runner owns the audit writer, shell runner, ssh client, and loaded config.
// Constructed once in main() and threaded into every cli.Command action via
// methods on this struct. Tests construct a Runner directly with fakes for
// Audit or SSH.
//
// Cfg is the layered *Config (embedded defaults, overlaid by
// ~/.coily/config.yaml, then ./.coily/config.yaml). Path fields like
// Audit.LogPath are already resolved to absolute paths by cli-guard/config at
// load time, so this struct does no path expansion.
type Runner struct {
	Cfg    *Config
	Runner *shell.Runner
	Audit  *audit.Writer
	SSH    *coilyssh.Client
}

// NewRunner builds the production Runner from layered config. Exits the
// process if the config does not parse or if the audit directory is not
// writable. Better to fail loudly at startup than silently drop audit
// records per call.
func NewRunner() *Runner {
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "coily: fatal: cannot load config: %v\n", err)
		os.Exit(2)
	}

	aw := audit.NewWriter(cfg.Audit.LogPath)
	aw.MaxSizeMB = cfg.Audit.MaxSizeMB
	aw.MaxBackups = cfg.Audit.MaxBackups
	aw.MaxAgeDays = cfg.Audit.MaxAgeDays
	aw.Compress = cfg.Audit.Compress
	aw.SetRedactPolicy(decision.RedactPolicy())
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
		// ~/.ssh/known_hosts either way. Library lives in cli-guard/ssh.
		SSH: &coilyssh.Client{KeyPath: expandTilde(cfg.KaiServer.SSHKeyPath)},
	}
}

// WrapVerb wraps spec into a cli.ActionFunc. Always injects
// OnEvaluate so every audit row carries the resolved profile decision
// and audit.Writer can apply data_security redaction. Callers passing
// their own OnEvaluate keep their value; nil callers get the
// runtime-managed evaluator.
//
// The writer argument is accepted (and forwarded to verb.Wrap) so the
// call shape `r.WrapVerb(spec, r.Audit)` is a single-token swap from
// the prior `verb.Wrap(spec, r.Audit)`.
func (r *Runner) WrapVerb(spec verb.Spec, writer *audit.Writer) cli.ActionFunc {
	if spec.OnEvaluate == nil {
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
	if spec.ResolveInvokeCWD == nil {
		spec.ResolveInvokeCWD = resolveInvokeCWD
	}
	return verb.Wrap(spec, writer)
}

// resolveInvokeCWD picks the directory a coily verb should treat as the
// operator's working directory, so an audit row binds to where the
// operator actually invoked from. Preference order:
//
//  1. $COILY_INVOKE_CWD - explicit override.
//  2. $OLDPWD - the shell's prior directory, set when a wrapper cd's.
//  3. os.Getwd() - subprocess cwd, the previous default.
//
// Any candidate that doesn't resolve to a real directory is skipped,
// so a stale env var doesn't poison the lookup.
func resolveInvokeCWD() string {
	for _, env := range []string{"COILY_INVOKE_CWD", "OLDPWD"} {
		v := strings.TrimSpace(os.Getenv(env))
		if v == "" {
			continue
		}
		// #nosec G304 -- read-only stat for cwd routing; no file open or
		// write follows.
		if info, err := os.Stat(filepath.Clean(v)); err == nil && info.IsDir() {
			return v
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	return ""
}

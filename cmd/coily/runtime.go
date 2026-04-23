package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/coilysiren/coily/pkg/audit"
	"github.com/coilysiren/coily/pkg/auth"
	"github.com/coilysiren/coily/pkg/config"
	"github.com/coilysiren/coily/pkg/shell"
)

// runtime is the package-wide carrier for audit writer, token verifier, and
// shell runner. Verbs obtain it via getRuntime(). Constructed lazily on first
// use from layered config (embedded + ~/.coily/config.yaml + ./.coily/
// config.yaml). Singleton.
type runtime struct {
	cfg    *config.Config
	audit  *audit.Writer
	issuer *auth.Issuer
	runner *shell.Runner
}

var (
	rtOnce sync.Once
	rtInst *runtime
)

func getRuntime() *runtime {
	rtOnce.Do(func() {
		cfg, err := config.Load()
		if err != nil {
			// Runtime is required. If config won't parse, nothing else can work.
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

		// Make sure the issuer key parent dir exists with 0700 before any
		// auth verb tries to read or write the key. The auth package does
		// the same mkdir on demand, but doing it here keeps the failure mode
		// uniform with the audit dir above.
		if dir := filepath.Dir(cfg.Tokens.IssuerKeyPath); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o700); err != nil {
				fmt.Fprintf(os.Stderr, "coily: fatal: token issuer dir %s: %v\n", dir, err)
				os.Exit(2)
			}
		}

		rtInst = &runtime{
			cfg:    cfg,
			audit:  aw,
			issuer: auth.NewIssuer(cfg.Tokens.IssuerKeyPath),
			runner: &shell.Runner{
				Stdout: os.Stdout,
				Stderr: os.Stderr,
				Stdin:  os.Stdin,
			},
		}
	})
	return rtInst
}

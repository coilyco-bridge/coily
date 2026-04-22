package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/coilysiren/coily/pkg/audit"
	"github.com/coilysiren/coily/pkg/auth"
	"github.com/coilysiren/coily/pkg/config"
	"github.com/coilysiren/coily/pkg/shell"
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

// runtime is the package-wide carrier for audit writer, token verifier, and
// shell runner. Verbs obtain it via getRuntime(). Constructed lazily on first
// use from embedded config. Singleton.
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

		rtInst = &runtime{
			cfg:    cfg,
			audit:  audit.NewWriter(auditPath),
			issuer: auth.NewIssuer(issuerKey),
			runner: &shell.Runner{
				Stdout: os.Stdout,
				Stderr: os.Stderr,
				Stdin:  os.Stdin,
			},
		}
	})
	return rtInst
}

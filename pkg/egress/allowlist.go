package egress

// Allowlists is the per-binary set of upstreams each package manager is
// allowed to reach in enforce mode. Pinned in code, not user-configurable
// for v0.1 (issue #35).
//
// Phase 1 lands the brew entry only. Phase 2 fills in the remaining 11
// pkgmgrs (npm/pnpm/yarn/bun, pip/uv/pipx/poetry, cargo, gem/bundle).
var Allowlists = map[string][]string{
	"brew": {
		"formulae.brew.sh",
		"ghcr.io",
		"objects.githubusercontent.com",
		"github.com",
		"raw.githubusercontent.com",
	},
}

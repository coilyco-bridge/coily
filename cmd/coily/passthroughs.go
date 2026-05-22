package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/coilysiren/cli-guard/egress"
	"github.com/coilysiren/cli-guard/mcporter"
	"github.com/coilysiren/cli-guard/passthrough"
	"github.com/urfave/cli/v3"
)

// ptEntry describes a single pass-through wrapper. Every binary coily fronts
// (aws, gh, kubectl, docker, tailscale, plus every package manager) is
// expressed as one row here, and the per-binary ops_*.go files are gone. To
// add a new pass-through, append a row and decide which group it mounts
// under (opsCommand, pkgCommand, or builtInCommands).
//
// SkipPolicy mirrors passthrough.WithSkipPolicy. Reserved for future
// per-verb opt-outs once cli-guard grows verb-granular policy hooks.
// No coily wrapper sets it today: the threat model is verb-granular
// (`tailscale status` is execve-only, `tailscale ssh` ships argv into
// a remote shell) and a tool-granular toggle can't tell them apart.
// See coilysiren/coily#162.
//
// VerbName mirrors passthrough.WithVerbName and overrides the audit verb
// from the bare bin name. Used for binaries mounted under a group so the
// audit row reflects the user-visible path (`coily ops aws` writes
// "ops.aws", not "aws").
//
// Egress, when true, routes the wrapped binary's HTTP/HTTPS through the
// per-invocation CONNECT proxy so the audit row's egress_hosts column is
// populated. If cli-guard/egress has a registered allowlist for the binary,
// the proxy runs in ModeEnforce; otherwise it runs in ModeObserve (capture
// only). Enforce-mode allowlists for individual binaries are Phase 2 of #35.
//
// ScopeArgvHint, when non-nil, installs a fallback --commit-scope resolver
// that fires when the operator did not set the flag (or env var) and the
// verb's argv carries enough information to pick a sensible default. Today
// only `ops gh` uses this, to derive the scope from --repo coilysiren/<name>.
//
// PreflightGate, when non-nil, runs against the raw argv before the
// passthrough executes. A non-nil return aborts the invocation with that
// error and the wrapped binary never runs. Today only `ops gh` uses this,
// to keep GitHub Actions / CI status playwright-only (coilysiren/coily#305).
type ptEntry struct {
	Bin            string
	SkipPolicy     bool
	VerbName       string
	Egress         bool
	ScopeArgvHint  func(argv []string) string
	ArgvRewriter   func(argv []string) []string
	ReadCache      passthrough.ReadCacheClassifier
	SecretResolver mcporter.SecretResolver
	PreflightGate  func(argv []string) error
}

// ptOps is the pass-through set mounted under `coily ops <bin>`. Cloud +
// repo + cluster pass-throughs live here so the top-level surface stays
// small. Audit verb names are stamped "ops.<bin>" so the log reflects the
// user-visible path.
var ptOps = []ptEntry{
	{Bin: "aws", VerbName: "ops.aws", Egress: true},
	{Bin: "gh", VerbName: "ops.gh", Egress: true, ScopeArgvHint: ghRepoScopeHint, ArgvRewriter: rewriteGHForRESTAndJQFile, ReadCache: ghReadCacheClassifier, PreflightGate: ghActionsGate},
	{Bin: "kubectl", VerbName: "ops.kubectl", Egress: true},
	{Bin: "flyctl", VerbName: "ops.flyctl", Egress: true},
	{Bin: "gcloud", VerbName: "ops.gcloud", Egress: true},
	{Bin: "mcporter", VerbName: "ops.mcporter", Egress: true, ArgvRewriter: rewriteMcporterArgsFile, SecretResolver: ssmResolver()},
}

// ptTopLevel is the pass-through set mounted at the coily root. Each entry
// becomes a top-level verb (`coily docker ...`, `coily tailscale ...`).
// These don't share a category with the ops/pkg groups.
//
// Neither entry sets Egress: true on purpose. The docker and tailscale CLIs
// talk to their respective local daemons (dockerd, tailscaled) over a unix
// socket. The daemon does the actual outbound HTTPS, so HTTPS_PROXY env vars
// set on the CLI never reach the code that opens the network connection.
// Wiring the proxy here would start a listener nothing connects to.
var ptTopLevel = []ptEntry{
	{Bin: "docker"},
	{Bin: "tailscale"},
}

// ptPkg is the package-manager set mounted under `coily pkg <bin>`. Order
// is the priority order from issue #22: how often the binary shows up in
// coilysiren/* repos, plus how dangerous a missed-audit invocation would
// be.
//
// Skipped intentionally:
//   - deno, go install / go run: already denied at the lockdown layer and
//     not used as package-installation paths in the workspace.
var ptPkg = []ptEntry{
	{Bin: "pnpm", Egress: true},
	{Bin: "npm", Egress: true},
	{Bin: "yarn", Egress: true},
	{Bin: "bun", Egress: true},
	{Bin: "uv", Egress: true},
	{Bin: "pip", Egress: true},
	{Bin: "pipx", Egress: true},
	{Bin: "poetry", Egress: true},
	{Bin: "cargo", Egress: true},
	{Bin: "gem", Egress: true},
	{Bin: "bundle", Egress: true},
	// nix is a universal package manager, not language-scoped. Added for
	// the kai-server Tangled-knot build (coilysiren/infrastructure#260
	// family): the knot is built via `nix build`, and the auto-deploy
	// timer runs nix under audit.
	{Bin: "nix", Egress: true},
	// brew is NOT a thin passthrough: it has its own scoped wrapper
	// at pkgBrewCommand (coily#253) that handles formula-scoped,
	// tap-scoped, and touch-everything verbs alongside read-only
	// passthrough. Wired into pkgCommand directly.
}

// passthroughCommand builds a cli.Command from a single registry entry.
// Centralizing the option assembly here means "what does a pass-through
// look like" lives in one place, and adding a new flag (egress mode,
// future telemetry hooks, etc.) is a one-line change to this function plus
// a field on ptEntry.
func (r *Runner) passthroughCommand(e ptEntry) *cli.Command {
	var opts []passthrough.Option
	if e.SkipPolicy {
		opts = append(opts, passthrough.WithSkipPolicy())
	}
	if e.VerbName != "" {
		opts = append(opts, passthrough.WithVerbName(e.VerbName))
	}
	if e.Egress {
		if allow, ok := egress.Allowlists[e.Bin]; ok {
			opts = append(opts, passthrough.WithEgress(allow, egress.ModeEnforce))
		} else {
			// No registered allowlist: still wire the proxy in observe mode so
			// the audit row's egress_hosts column is populated. Enforcement is a
			// later phase, capture-first lights up the dashboard today (#139).
			opts = append(opts, passthrough.WithEgress(nil, egress.ModeObserve))
		}
	}
	if e.ScopeArgvHint != nil {
		opts = append(opts, passthrough.WithScopeArgvHint(e.ScopeArgvHint))
	}
	if e.ArgvRewriter != nil {
		opts = append(opts, passthrough.WithArgvRewriter(e.ArgvRewriter))
	}
	if e.ReadCache != nil {
		opts = append(opts, passthrough.WithReadCache(e.ReadCache))
	}
	if e.SecretResolver != nil {
		opts = append(opts, passthrough.WithSecretResolver(e.SecretResolver))
	}
	cmd := passthrough.Command(e.Bin, r.Runner, r.Audit, opts...)
	if e.PreflightGate != nil {
		cmd.Action = withPreflightGate(cmd.Action, e.PreflightGate)
	}
	return cmd
}

// withPreflightGate wraps a passthrough action so `gate` runs against the
// raw argv first. A non-nil gate result aborts before the wrapped binary
// (or its audit row) ever runs - the gate is a hard refusal, not a policy
// the passthrough negotiates. Used to keep GitHub Actions status
// playwright-only (coilysiren/coily#305).
func withPreflightGate(inner cli.ActionFunc, gate func(argv []string) error) cli.ActionFunc {
	return func(ctx context.Context, c *cli.Command) error {
		if err := gate(c.Args().Slice()); err != nil {
			return err
		}
		return inner(ctx, c)
	}
}

// ghRepoScopeHint reads --repo coilysiren/<name> out of `coily ops gh` argv
// and returns ~/projects/coilysiren/<name> when that local clone exists and
// is a real git repo. Returns "" otherwise (including when --repo names a
// non-coilysiren owner: those clones may live elsewhere or not at all, and
// silently binding the audit row to a same-name coilysiren clone would be a
// surprise). Recognized argv shapes: `--repo X/Y`, `--repo=X/Y`, `-R X/Y`,
// `-R=X/Y`. The function ignores positional argv beyond gh subcommands
// (gh's --repo is a top-level flag inherited by every subcommand, so the
// scan order does not matter).
func ghRepoScopeHint(argv []string) string {
	owner, name := parseGhRepoFlag(argv)
	if owner != "coilysiren" || name == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	candidate := filepath.Join(home, "projects", "coilysiren", name)
	if fi, err := os.Stat(filepath.Join(candidate, ".git")); err != nil || fi == nil {
		return ""
	}
	return candidate
}

func parseGhRepoFlag(argv []string) (owner, name string) {
	for i := 0; i < len(argv); i++ {
		tok := argv[i]
		var raw string
		switch {
		case tok == "--repo" || tok == "-R":
			if i+1 >= len(argv) {
				return "", ""
			}
			raw = argv[i+1]
		case strings.HasPrefix(tok, "--repo="):
			raw = strings.TrimPrefix(tok, "--repo=")
		case strings.HasPrefix(tok, "-R="):
			raw = strings.TrimPrefix(tok, "-R=")
		default:
			continue
		}
		o, n, ok := strings.Cut(raw, "/")
		if !ok {
			return "", ""
		}
		return o, n
	}
	return "", ""
}

// passthroughCommands is the slice form: build one cli.Command per entry,
// preserving registry order. Used by group constructors that mount their
// set under a parent verb.
func (r *Runner) passthroughCommands(entries []ptEntry) []*cli.Command {
	out := make([]*cli.Command, 0, len(entries))
	for _, e := range entries {
		out = append(out, r.passthroughCommand(e))
	}
	return out
}

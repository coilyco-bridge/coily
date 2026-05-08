package main

import (
	"github.com/coilysiren/coily/pkg/egress"
	"github.com/coilysiren/coily/pkg/passthrough"
	"github.com/urfave/cli/v3"
)

// ptEntry describes a single pass-through wrapper. Every binary coily fronts
// (aws, gh, kubectl, docker, tailscale, plus every package manager) is
// expressed as one row here, and the per-binary ops_*.go files are gone. To
// add a new pass-through, append a row and decide which group it mounts
// under (opsCommand, pkgCommand, or builtInCommands).
//
// SkipPolicy mirrors passthrough.WithSkipPolicy: enable for tools whose
// argv goes through execve straight to the underlying CLI without ever
// being handed to a shell. Unsafe for tools with exec-into-shell paths
// (kubectl, docker, ssh - argv content can reach a remote bash -c there).
//
// VerbName mirrors passthrough.WithVerbName and overrides the audit verb
// from the bare bin name. Used for binaries mounted under a group so the
// audit row reflects the user-visible path (`coily ops aws` writes
// "ops.aws", not "aws").
//
// Egress, when true, opts the wrapper into the per-binary egress allowlist
// in pkg/egress. Today only brew has an entry; the other package managers
// gain enforce mode in Phase 2 of issue #35.
type ptEntry struct {
	Bin        string
	SkipPolicy bool
	VerbName   string
	Egress     bool
}

// ptOps is the pass-through set mounted under `coily ops <bin>`. Cloud +
// repo + cluster pass-throughs live here so the top-level surface stays
// small. Audit verb names are stamped "ops.<bin>" so the log reflects the
// user-visible path.
var ptOps = []ptEntry{
	{Bin: "aws", SkipPolicy: true, VerbName: "ops.aws"},
	{Bin: "gh", SkipPolicy: true, VerbName: "ops.gh"},
	{Bin: "kubectl", VerbName: "ops.kubectl"},
}

// ptTopLevel is the pass-through set mounted at the coily root. Each entry
// becomes a top-level verb (`coily docker ...`, `coily tailscale ...`).
// These don't share a category with the ops/pkg groups.
var ptTopLevel = []ptEntry{
	{Bin: "docker"},
	{Bin: "tailscale", SkipPolicy: true},
	{Bin: "linkedin", SkipPolicy: true},
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
	{Bin: "brew", Egress: true},
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
		}
	}
	return passthrough.Command(e.Bin, r.Runner, r.Audit, opts...)
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

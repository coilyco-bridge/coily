package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// capabilityDoctrine is the standing caveat the awareness index leads with.
// The index says a surface EXISTS, never that its credentials are valid -
// so the agent verifies with a cheap read instead of asking the human or
// assuming. coilyco-bridge/coily#197.
const capabilityDoctrine = "presence != working auth: the surface existing does not mean creds are valid. " +
	"Verify with a cheap read (e.g. `coily ops aws sts get-caller-identity`) before asking the human or assuming."

// serviceGroups are the top-level groups whose non-passthrough subgroups
// are external-capability surfaces worth advertising at session open (the
// "do you have an X integration?" question tool-presence answers). Other
// groups carry coily-internal mechanics (dispatch registry, lockdown
// subcommands, pkg directory wrappers) that would be noise in a
// session-open injection. Passthrough leaves are advertised regardless of
// their group; this set only governs the service-wrapper line.
var serviceGroups = map[string]bool{"ops": true, "gaming": true}

// capabilityIndex renders the condensed surface map injected at SessionStart
// so a session opens knowing what coily fronts, without the agent reaching
// for a bare tool or asking the human a question tool-presence already
// answers. Built live from the in-process command tree (root), so it can
// never drift from what coily actually exposes. Full depth stays on-demand
// via `coily --tree[ --json]`.
//
// Structure mirrors the tree: top-level passthrough leaves (docker,
// tailscale), each group's passthrough children (ops, pkg), and each
// group's non-passthrough subgroups (ops REST wrappers, gaming servers).
func capabilityIndex(root *treeNode) string {
	topPass, passSegs, svcSegs := collectSurface(root)

	var pass []string
	if len(topPass) > 0 {
		pass = append(pass, fmt.Sprintf("coily {%s}", strings.Join(topPass, " ")))
	}
	for _, s := range passSegs {
		pass = append(pass, s.render())
	}
	var svc []string
	for _, s := range svcSegs {
		svc = append(svc, s.render())
	}

	var b strings.Builder
	fmt.Fprintf(&b, "coily capability index - %s\n", capabilityDoctrine)
	if len(pass) > 0 {
		fmt.Fprintf(&b, "audited tool passthroughs (route the bare tool through these): %s\n", strings.Join(pass, " | "))
	}
	if len(svc) > 0 {
		fmt.Fprintf(&b, "service wrappers: %s\n", strings.Join(svc, " | "))
	}
	b.WriteString("full depth on demand: `coily --tree` (human) or `coily --tree --json [subtree]` (machine).")
	return b.String()
}

// capSeg is one "coily <path> {item item ...}" segment of the index.
type capSeg struct {
	path  string
	items []string
}

func (s capSeg) render() string {
	return fmt.Sprintf("%s {%s}", s.path, strings.Join(s.items, " "))
}

// collectSurface walks the top level of the tree once and splits it into
// the three index lanes: top-level passthrough bins (docker, tailscale),
// per-group passthrough bins (ops, pkg), and per-group external-service
// subgroups (ops REST wrappers, gaming servers - serviceGroups only).
func collectSurface(root *treeNode) (topPass []string, passSegs, svcSegs []capSeg) {
	for _, c := range root.Children {
		if c.Bin != "" {
			topPass = append(topPass, c.Bin)
			continue
		}
		if len(c.Children) == 0 {
			continue
		}
		bins, svcs := classifyGroupChildren(c)
		if len(bins) > 0 {
			passSegs = append(passSegs, capSeg{"coily " + c.Name, bins})
		}
		if len(svcs) > 0 {
			svcSegs = append(svcSegs, capSeg{"coily " + c.Name, svcs})
		}
	}
	return topPass, passSegs, svcSegs
}

// classifyGroupChildren splits a group's direct children into passthrough
// bins and (for a serviceGroup) external-service subgroup names.
func classifyGroupChildren(c *treeNode) (bins, svcs []string) {
	for _, k := range c.Children {
		switch {
		case k.Bin != "":
			bins = append(bins, k.Bin)
		case len(k.Children) > 0 && serviceGroups[c.Name]:
			svcs = append(svcs, k.Name)
		}
	}
	return bins, svcs
}

// sessionStartHookOutput is the Claude Code SessionStart hook contract:
// stdout is parsed as JSON and hookSpecificOutput.additionalContext is
// injected into the session as context. Exit 0.
type sessionStartHookOutput struct {
	HookSpecificOutput sessionStartHookSpecific `json:"hookSpecificOutput"`
}

type sessionStartHookSpecific struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

// hookSessionStartCommand emits the capability index as a SessionStart hook
// payload. Wired into each repo's .claude/settings.json by `coily lockdown`
// alongside the PreToolUse hook, so every session in a locked-down repo
// opens with the surface map. coilyco-bridge/coily#197.
func (r *Runner) hookSessionStartCommand() *cli.Command {
	return &cli.Command{
		Name:  "session-start",
		Usage: "SessionStart hook. Emits the coily capability index (surface map + presence!=auth doctrine) as additionalContext.",
		Action: r.WrapVerb(verb.Spec{
			Name:       "hook.session-start",
			SkipPolicy: true,
			Action: func(_ context.Context, _ *cli.Command) error {
				return r.emitCapabilityIndex(os.Stdout)
			},
		}, r.Audit),
	}
}

// emitCapabilityIndex builds the tree from the live command set and writes
// the SessionStart JSON payload to w. Split out so a test can drive it
// against a buffer without going through the cli wrapper.
func (r *Runner) emitCapabilityIndex(w *os.File) error {
	root := buildTree(r.builtInCommands(), nil)
	out := sessionStartHookOutput{
		HookSpecificOutput: sessionStartHookSpecific{
			HookEventName:     "SessionStart",
			AdditionalContext: capabilityIndex(root),
		},
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("coily hook session-start: encode: %w", err)
	}
	return nil
}

// sortedCapabilityTokens is a test/diagnostic helper: the flat set of tool
// tokens the index advertises, sorted. Kept here so the drift test against
// wrapperRecovery reads from the same derivation the index uses.
func sortedCapabilityTokens(root *treeNode) []string {
	seen := map[string]bool{}
	var walk func(n *treeNode)
	walk = func(n *treeNode) {
		if n.Bin != "" {
			seen[n.Bin] = true
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	out := make([]string, 0, len(seen))
	for t := range seen {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

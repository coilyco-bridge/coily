package skillgen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

// PassthroughEntry is one leaf verb in the structured (yaml) form of the
// command tree. Programmatic consumers (an external orchestrator, a
// plan-time agent) read this shape; humans read the markdown form.
type PassthroughEntry struct {
	Path    []string `yaml:"path"`
	Summary string   `yaml:"summary,omitempty"`
	Flags   []string `yaml:"flags,omitempty"`
}

// RenderPassthroughsYAML walks the same cli.Command tree as
// RenderPassthroughs and emits a yaml document of PassthroughEntry rows.
// Order matches the markdown form (depth-first by command name) so a
// consumer can diff the two outputs against each other.
func RenderPassthroughsYAML(commands []*cli.Command) (string, error) {
	var entries []PassthroughEntry
	for _, c := range sorted(commands) {
		walkPassthroughEntries(&entries, []string{"coily"}, c)
	}
	doc := map[string]any{"commands": entries}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("skillgen: yaml marshal: %w", err)
	}
	return string(out), nil
}

func walkPassthroughEntries(out *[]PassthroughEntry, parent []string, c *cli.Command) {
	path := append(append([]string{}, parent...), c.Name)
	if len(c.Commands) == 0 {
		*out = append(*out, PassthroughEntry{
			Path:    path,
			Summary: strings.TrimSpace(c.Usage),
			Flags:   flagNames(c.Flags),
		})
		return
	}
	for _, sub := range sorted(c.Commands) {
		walkPassthroughEntries(out, path, sub)
	}
}

// PassthroughsFrontmatter is the YAML frontmatter prepended to the
// generated coily-passthroughs SKILL.md. The description is what Claude
// Code uses to decide when to load the skill.
//
//nolint:gosec // YAML frontmatter; gosec misreads the description body
const PassthroughsFrontmatter = `---
name: coily-passthroughs
description: |
  Use when a shell command is denied by Claude Code's permission system
  (e.g. "Permission to use Bash with command X has been denied"), when
  reaching for aws, gh, kubectl, docker, tailscale, ssh, or scp against
  Kai's homelab, AWS account, or coilysiren resources, or when checking
  whether a privileged op has a coily wrapper. The body is a flat lookup
  table of every coily command.
---
`

// RenderPassthroughs walks the passed-in cli.Command tree (typically
// the production builtInCommands() list) and emits a deterministic
// markdown lookup table:
//
//	## coily aws ssm get-parameter
//
//	One-line summary from the cli Usage field.
//
//	Flags: --name, --with-decryption, --query, --output
//
// Order is depth-first by command name. Determinism matters because the
// generated file is diff-checked in CI; non-stable iteration would make
// the diff noisy.
func RenderPassthroughs(commands []*cli.Command) string {
	var b strings.Builder
	b.WriteString(PassthroughsFrontmatter)
	b.WriteString("\n# coily passthroughs\n\n")
	b.WriteString("Auto-generated lookup table of every coily verb. Regenerate with `coily lockdown skill`.\n\n")
	b.WriteString("Format: full path, one-line summary, comma-separated flag names. No flag descriptions; click into `coily <path> --help` for those.\n\n")
	for _, c := range sorted(commands) {
		walkPassthroughs(&b, []string{"coily"}, c)
	}
	// One trailing newline; pre-commit's end-of-file-fixer rejects more.
	return strings.TrimRight(b.String(), "\n") + "\n"
}

func walkPassthroughs(b *strings.Builder, parent []string, c *cli.Command) {
	path := append(append([]string{}, parent...), c.Name)
	if len(c.Commands) == 0 {
		// Leaf.
		fmt.Fprintf(b, "## `%s`\n\n", strings.Join(path, " "))
		if u := strings.TrimSpace(c.Usage); u != "" {
			fmt.Fprintf(b, "%s\n\n", u)
		}
		if names := flagNames(c.Flags); len(names) > 0 {
			fmt.Fprintf(b, "Flags: %s\n\n", strings.Join(names, ", "))
		}
		return
	}
	for _, sub := range sorted(c.Commands) {
		walkPassthroughs(b, path, sub)
	}
}

func sorted(in []*cli.Command) []*cli.Command {
	out := append([]*cli.Command(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func flagNames(flags []cli.Flag) []string {
	names := make([]string, 0, len(flags))
	seen := map[string]bool{}
	for _, f := range flags {
		// Take the first registered name (canonical long form). Aliases
		// and short forms are skipped to keep the table scannable.
		all := f.Names()
		if len(all) == 0 {
			continue
		}
		n := all[0]
		if seen[n] {
			continue
		}
		seen[n] = true
		names = append(names, "--"+n)
	}
	sort.Strings(names)
	return names
}

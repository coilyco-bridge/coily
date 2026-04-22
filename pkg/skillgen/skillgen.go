// Package skillgen renders a Claude Code skill from coily's sub-CLI command
// manifests. Produces:
//
//	<out>/SKILL.md                 — always, concise trigger doc
//	<out>/reference/<binary>.md    — one per manifest found, full verb tree
//
// Inputs:
//
//   - <configs>/commands/<binary>.yaml, produced by cmd/subcli-scope.
//   - A list of hand-written top-level coily verbs (lockdown, whoami, etc.)
//     passed in by the caller, since those don't come from sub-CLI introspection.
//
// The generator is deterministic and takes no external state (no clocks, no
// network). Re-running it on the same inputs yields byte-identical output.
package skillgen

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Verb describes a hand-written top-level coily subcommand (not produced by
// mirroring an underlying CLI). Caller populates from the urfave/cli command
// tree or hardcoded at the call site.
type Verb struct {
	Name    string   // e.g. "lockdown"
	Usage   string   // short one-liner
	Flags   []string // e.g. ["--path", "--apply", "--local", "--replace"]
	Example string   // one-line invocation example
}

// Manifest mirrors cmd/subcli-scope's output structure. Duplicated here to
// avoid importing main.
type Manifest struct {
	Binary     string            `yaml:"binary"`
	BinVersion string            `yaml:"bin_version,omitempty"`
	Commands   []ManifestCommand `yaml:"commands"`
}

type ManifestCommand struct {
	Path     []string       `yaml:"path"`
	Help     string         `yaml:"help,omitempty"`
	Flags    []ManifestFlag `yaml:"flags,omitempty"`
	Children []string       `yaml:"children,omitempty"`
}

type ManifestFlag struct {
	Name string `yaml:"name"`
	Help string `yaml:"help,omitempty"`
}

// Options configures a generator run.
type Options struct {
	CommandsDir string // path to configs/commands/ (source manifests)
	OutDir      string // path to write SKILL.md into (and reference/ under it)
	Verbs       []Verb // hand-written top-level verbs
}

// Generate writes SKILL.md plus reference/<binary>.md per manifest in CommandsDir.
func Generate(opt Options) error {
	manifests, err := loadManifests(opt.CommandsDir)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(opt.OutDir, 0o755); err != nil {
		return fmt.Errorf("skillgen: mkdir outdir: %w", err)
	}
	refDir := filepath.Join(opt.OutDir, "reference")
	if err := os.MkdirAll(refDir, 0o755); err != nil {
		return fmt.Errorf("skillgen: mkdir refdir: %w", err)
	}

	skill := renderSkillMD(manifests, opt.Verbs)
	if err := os.WriteFile(filepath.Join(opt.OutDir, "SKILL.md"), []byte(skill), 0o644); err != nil {
		return fmt.Errorf("skillgen: write SKILL.md: %w", err)
	}

	for _, m := range manifests {
		ref := renderReferenceMD(m)
		p := filepath.Join(refDir, m.Binary+".md")
		if err := os.WriteFile(p, []byte(ref), 0o644); err != nil {
			return fmt.Errorf("skillgen: write %s: %w", p, err)
		}
	}
	return nil
}

func loadManifests(dir string) ([]Manifest, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("skillgen: read %s: %w", dir, err)
	}
	var out []Manifest
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		var m Manifest
		if err := yaml.Unmarshal(b, &m); err != nil {
			return nil, fmt.Errorf("skillgen: parse %s: %w", e.Name(), err)
		}
		if m.Binary == "" {
			continue
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Binary < out[j].Binary })
	return out, nil
}

func renderSkillMD(manifests []Manifest, verbs []Verb) string {
	var b strings.Builder
	writeSkillFrontmatter(&b)
	writeSkillIntro(&b)
	writeSkillWhenToUse(&b)
	writeSkillCommandShape(&b)
	writeSkillNativeVerbs(&b, verbs)
	writeSkillPassThroughTools(&b, manifests)
	writeSkillExamples(&b)
	writeSkillWillNotDo(&b)
	return b.String()
}

func writeSkillFrontmatter(b *strings.Builder) {
	b.WriteString("---\n")
	b.WriteString("name: coily\n")
	b.WriteString("description: Operator CLI for Kai's homelab. Use coily instead of direct aws/kubectl/gh/ssh invocations when operating against kai-server, Kai's AWS account (coilysiren), or coilysiren GitHub repos. coily is the only tool authorized for privileged ops in Kai's workspace and audit-logs every invocation.\n")
	b.WriteString("---\n\n")
}

func writeSkillIntro(b *strings.Builder) {
	b.WriteString("# coily\n\n")
	b.WriteString("coily wraps a curated subset of `aws`, `kubectl`, `gh`, and `tailscale`, plus `eco` (systemd on kai-server) and direct `ssh` to kai-server. Every invocation is argv-only (no shell metacharacter injection), policy-checked, and audit-logged.\n\n")
}

func writeSkillWhenToUse(b *strings.Builder) {
	b.WriteString("## When to use\n\n")
	b.WriteString("- Any op against kai-server, Kai's AWS account, or `coilysiren/*` GitHub repos.\n")
	b.WriteString("- Anywhere the reflex would be `aws ...`, `kubectl ...`, or `gh ...`. Prefix with `coily `.\n")
	b.WriteString("- NOT general-purpose AWS calls from work or other accounts. Use the standard `aws` CLI for those.\n\n")
}

func writeSkillCommandShape(b *strings.Builder) {
	b.WriteString("## Command shape\n\n")
	b.WriteString("`coily <tool> <verb...> [flags]`. Flags mirror the underlying CLI exactly. `coily aws ssm get-parameter --name /foo --with-decryption` is identical in meaning to `aws ssm get-parameter --name /foo --with-decryption`.\n\n")
}

func writeSkillNativeVerbs(b *strings.Builder, verbs []Verb) {
	if len(verbs) == 0 {
		return
	}
	b.WriteString("## Coily-native verbs\n\n")
	b.WriteString("These do not mirror an underlying CLI. They are coily's own operations.\n\n")
	for _, v := range verbs {
		writeSkillVerb(b, v)
	}
}

func writeSkillVerb(b *strings.Builder, v Verb) {
	b.WriteString("### `coily " + v.Name + "`\n\n")
	if v.Usage != "" {
		b.WriteString(v.Usage + "\n\n")
	}
	if len(v.Flags) > 0 {
		b.WriteString("Flags: " + strings.Join(v.Flags, ", ") + "\n\n")
	}
	if v.Example != "" {
		b.WriteString("```\n" + v.Example + "\n```\n\n")
	}
}

func writeSkillPassThroughTools(b *strings.Builder, manifests []Manifest) {
	if len(manifests) == 0 {
		return
	}
	b.WriteString("## Pass-through tools\n\n")
	b.WriteString("For each of these, `coily <tool> ...` takes the same arguments as `<tool> ...` directly. Full verb trees are in this skill's reference directory.\n\n")
	for _, m := range manifests {
		fmt.Fprintf(b, "- **`coily %s`** - %d verbs. Full reference: `reference/%s.md`.\n",
			m.Binary, countLeaves(m.Commands), m.Binary)
	}
	b.WriteString("\n")
}

func countLeaves(cmds []ManifestCommand) int {
	n := 0
	for _, c := range cmds {
		if len(c.Children) == 0 {
			n++
		}
	}
	return n
}

func writeSkillExamples(b *strings.Builder) {
	b.WriteString("## Examples\n\n")
	b.WriteString("```\n")
	b.WriteString("coily aws sts get-caller-identity\n")
	b.WriteString("coily kubectl get pods -A\n")
	b.WriteString("coily gh run list --repo coilysiren/coily\n")
	b.WriteString("coily lockdown --path . --apply\n")
	b.WriteString("```\n\n")
}

func writeSkillWillNotDo(b *strings.Builder) {
	b.WriteString("## What coily will not do\n\n")
	b.WriteString("- Open a shell. There is no `coily run` or `coily exec`, ever.\n")
	b.WriteString("- Take free-form string arguments that reach a shell. Shell metacharacters are rejected at the policy layer.\n")
	b.WriteString("- Self-update at runtime. Binary updates go through `make deploy-server` from Kai's laptop.\n")
	b.WriteString("- Perform destructive operations without a valid confirmation token (see `coily auth issue`).\n")
}

func renderReferenceMD(m Manifest) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# coily %s - full reference\n\n", m.Binary)
	if m.BinVersion != "" {
		fmt.Fprintf(&b, "Mirrors `%s`. Underlying version at scan time: %s\n\n", m.Binary, m.BinVersion)
	} else {
		fmt.Fprintf(&b, "Mirrors `%s`.\n\n", m.Binary)
	}
	b.WriteString("Command shape: `coily " + m.Binary + " <verb...> [flags]`. Flags match the underlying CLI.\n\n")

	// Group by top-level subcommand (first path element) for readability.
	groups := map[string][]ManifestCommand{}
	var topNames []string
	for _, c := range m.Commands {
		if len(c.Path) == 0 {
			continue
		}
		top := c.Path[0]
		if _, seen := groups[top]; !seen {
			topNames = append(topNames, top)
		}
		groups[top] = append(groups[top], c)
	}
	sort.Strings(topNames)

	for _, top := range topNames {
		fmt.Fprintf(&b, "## `coily %s %s`\n\n", m.Binary, top)
		for _, c := range groups[top] {
			renderCommand(&b, m.Binary, c)
		}
	}
	return b.String()
}

func renderCommand(b *strings.Builder, binary string, c ManifestCommand) {
	path := strings.Join(c.Path, " ")
	if len(c.Children) > 0 {
		// Internal node. Just list the children as a navigation hint.
		fmt.Fprintf(b, "### `coily %s %s` (group)\n\n", binary, path)
		if c.Help != "" {
			b.WriteString(c.Help + "\n\n")
		}
		b.WriteString("Subcommands: ")
		for i, ch := range c.Children {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString("`" + ch + "`")
		}
		b.WriteString("\n\n")
		return
	}
	// Leaf verb.
	fmt.Fprintf(b, "### `coily %s %s`\n\n", binary, path)
	if c.Help != "" {
		b.WriteString(c.Help + "\n\n")
	}
	if len(c.Flags) > 0 {
		b.WriteString("Flags:\n\n")
		for _, f := range c.Flags {
			if f.Help != "" {
				fmt.Fprintf(b, "- `%s` - %s\n", f.Name, f.Help)
			} else {
				fmt.Fprintf(b, "- `%s`\n", f.Name)
			}
		}
		b.WriteString("\n")
	}
}

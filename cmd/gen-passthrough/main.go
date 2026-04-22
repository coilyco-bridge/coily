// gen-passthrough reads a configs/commands/<binary>.yaml manifest and emits
// pkg/ops/<binary>/generated.go. The generated file registers a single
// Command() function that returns the full *cli.Command tree mirroring the
// underlying CLI, with every leaf wrapped through verb.Wrap so policy and
// audit are applied uniformly.
//
// Usage:
//
//	go run ./cmd/gen-passthrough <binary>   # e.g. aws, gh, kubectl
//	go run ./cmd/gen-passthrough all        # regenerate every manifest
//
// Classification of mutating vs. read-only verbs is done by prefix heuristic
// at generation time. See classifyVerb below. Wrong classification is a bug
// worth flagging: an agent-facing tool that mis-classifies a mutator as
// read-only silently drops token-gating for that verb.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// Manifest + ManifestCommand + ManifestFlag mirror pkg/skillgen (and thus
// subcli-scope's output). Kept local so this tool has no coily-internal
// dependencies beyond the yaml package.
type Manifest struct {
	Binary   string            `yaml:"binary"`
	Commands []ManifestCommand `yaml:"commands"`
}

type ManifestCommand struct {
	Path     []string       `yaml:"path"`
	Help     string         `yaml:"help,omitempty"`
	Flags    []ManifestFlag `yaml:"flags,omitempty"`
	Children []string       `yaml:"children,omitempty"`
}

type ManifestFlag struct {
	Name string `yaml:"name"`
}

func main() {
	if len(os.Args) != 2 {
		die("usage: gen-passthrough <binary | all>")
	}
	arg := os.Args[1]
	if arg == "all" {
		for _, bin := range []string{"aws", "gh", "kubectl"} {
			runOne(bin)
		}
		return
	}
	runOne(arg)
}

func runOne(binary string) {
	manifestPath := filepath.Join("configs", "commands", binary+".yaml")
	b, err := os.ReadFile(manifestPath)
	if err != nil {
		die("read %s: %v", manifestPath, err)
	}
	var m Manifest
	if err := yaml.Unmarshal(b, &m); err != nil {
		die("parse %s: %v", manifestPath, err)
	}
	if m.Binary != binary {
		die("manifest declares binary %q but called with %q", m.Binary, binary)
	}

	code, err := render(m)
	if err != nil {
		die("render %s: %v", binary, err)
	}

	outDir := filepath.Join("pkg", "ops", binary)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		die("mkdir %s: %v", outDir, err)
	}
	outPath := filepath.Join(outDir, "generated.go")
	if err := os.WriteFile(outPath, []byte(code), 0o644); err != nil {
		die("write %s: %v", outPath, err)
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", outPath)
}

// tree is the intermediate structure the template walks to render nested
// cli.Command literals. Built from the manifest's flat command list.
type tree struct {
	Name     string
	Path     []string
	Help     string
	Flags    []string
	Children []*tree
	Mutating bool
}

func buildTree(m Manifest) *tree {
	root := &tree{Name: m.Binary, Path: []string{}}
	byPath := map[string]*tree{"": root}

	// Sort commands so parents come before children.
	cmds := append([]ManifestCommand(nil), m.Commands...)
	sort.Slice(cmds, func(i, j int) bool {
		if len(cmds[i].Path) != len(cmds[j].Path) {
			return len(cmds[i].Path) < len(cmds[j].Path)
		}
		return strings.Join(cmds[i].Path, "/") < strings.Join(cmds[j].Path, "/")
	})

	for _, c := range cmds {
		node := &tree{
			Name:     c.Path[len(c.Path)-1],
			Path:     append([]string{}, c.Path...),
			Help:     sanitize(c.Help),
			Mutating: classifyVerb(c.Path),
		}
		for _, f := range c.Flags {
			node.Flags = append(node.Flags, strings.TrimPrefix(f.Name, "--"))
		}
		key := strings.Join(c.Path, "/")
		byPath[key] = node
		parentKey := strings.Join(c.Path[:len(c.Path)-1], "/")
		parent, ok := byPath[parentKey]
		if !ok {
			continue
		}
		parent.Children = append(parent.Children, node)
	}
	return root
}

// classifyVerb returns true for Mutating. Heuristic: the leaf verb name
// starts with one of a known set of mutation prefixes. Parent/group nodes
// return false; only leaves matter.
func classifyVerb(path []string) bool {
	if len(path) == 0 {
		return false
	}
	leaf := path[len(path)-1]
	// Normalize: drop numeric suffixes ("v2" etc.) and common suffixes.
	prefixes := []string{
		"create-", "delete-", "update-", "put-", "post-",
		"modify-", "change-", "apply-", "set-", "remove-",
		"add-", "register-", "deregister-",
		"associate-", "disassociate-", "attach-", "detach-",
		"enable-", "disable-", "activate-", "deactivate-",
		"start-", "stop-", "restart-", "reboot-", "terminate-",
		"rotate-", "refresh-", "reset-", "revoke-",
		"cancel-", "move-", "copy-", "rename-", "restore-",
		"patch-", "replace-", "tag-", "untag-", "label-",
		// kubectl-style bare verbs (leaves in kubectl's tree).
		"apply", "create", "delete", "patch", "replace", "edit",
		"label", "annotate", "scale", "autoscale", "set",
		"taint", "cordon", "uncordon", "drain", "expose",
		"run", "rollout-restart", "rollout-undo",
		// gh-style.
		"merge", "close", "reopen", "lock", "unlock", "pin", "unpin",
		"transfer",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(leaf, p) || leaf == strings.TrimSuffix(p, "-") {
			return true
		}
	}
	return false
}

func sanitize(s string) string {
	s = strings.ReplaceAll(s, "`", "'")
	s = strings.ReplaceAll(s, "\"", "'")
	if len(s) > 160 {
		s = s[:160] + "..."
	}
	return strings.TrimSpace(s)
}

func render(m Manifest) (string, error) {
	t := buildTree(m)
	funcs := template.FuncMap{
		"goString":   goString,
		"dottedPath": dottedPath,
		"join":       strings.Join,
		"args":       func(v ...any) []any { return v },
	}
	tpl, err := template.New("gen").Funcs(funcs).Parse(tmpl)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	data := map[string]any{
		"Binary":  m.Binary,
		"Package": m.Binary,
		"Root":    t,
	}
	if err := tpl.Execute(&sb, data); err != nil {
		return "", err
	}
	return sb.String(), nil
}

func goString(s string) string {
	return fmt.Sprintf("%q", s)
}

func dottedPath(binary string, path []string) string {
	if len(path) == 0 {
		return binary
	}
	return binary + "." + strings.Join(path, ".")
}

const tmpl = `// Code generated by cmd/gen-passthrough. DO NOT EDIT.
//
// Regenerate with: go run ./cmd/gen-passthrough {{.Binary}}
// Or in bulk:      make gen-passthrough

package {{.Package}}

import (
	"context"
	"strconv"

	"github.com/coilysiren/coily/pkg/audit"
	"github.com/coilysiren/coily/pkg/policy"
	"github.com/coilysiren/coily/pkg/shell"
	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

// BinaryName is the underlying CLI this package mirrors.
const BinaryName = {{.Binary | goString}}

// Command returns the *cli.Command tree that mirrors the upstream CLI.
// Every leaf is wrapped through verb.Wrap so policy enforcement and audit
// logging apply uniformly.
func Command(r *shell.Runner, v policy.TokenVerifier, w *audit.Writer) *cli.Command {
	return &cli.Command{
		Name:     BinaryName,
		Usage:    "Pass-through to " + BinaryName + ".",
		Commands: []*cli.Command{
{{- range .Root.Children}}
			{{template "cmd" (args . $.Binary $.Root.Name)}},
{{- end}}
		},
	}
}

{{define "cmd"}}
{{- $node := index . 0 -}}
{{- $binary := index . 1 -}}
{{- $rootName := index . 2 -}}
&cli.Command{
	Name: {{$node.Name | goString}},
	Usage: {{$node.Help | goString}},
{{- if $node.Flags}}
	Flags: []cli.Flag{
	{{- range $node.Flags}}
		&cli.StringFlag{Name: {{. | goString}}},
	{{- end}}
	},
{{- end}}
{{- if $node.Children}}
	Commands: []*cli.Command{
	{{- range $node.Children}}
		{{template "cmd" (args . $binary $rootName)}},
	{{- end}}
	},
{{- else}}
	Action: verb.Wrap(
		verb.Spec{
			Name: {{dottedPath $binary $node.Path | goString}},
			Kind: policy.{{if $node.Mutating}}Mutating{{else}}ReadOnly{{end}},
			ArgsFunc: func(c *cli.Command) (map[string]string, []string, string) {
				args := map[string]string{}
				{{- range $node.Flags}}
				args[{{printf "--%s" . | goString}}] = c.String({{. | goString}})
				{{- end}}
				return args, nil, c.String("token")
			},
			Action: func(ctx context.Context, c *cli.Command) error {
				argv := []string{ {{range $node.Path}}{{. | goString}}, {{end}} }
				{{- range $node.Flags}}
				if c.IsSet({{. | goString}}) {
					argv = append(argv, "--" + {{. | goString}}, c.String({{. | goString}}))
				}
				{{- end}}
				_ = strconv.Itoa // keep strconv imported even when no flags
				return r.Exec(ctx, BinaryName, argv...)
			},
		},
		v, w,
	),
{{- end}}
}
{{- end}}
`

// The template `args` function lets us pass multiple values into nested
// template invocations. Register it via FuncMap.
func init() {
	// no-op init; args registered below
}

func die(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "gen-passthrough: "+format+"\n", a...)
	os.Exit(1)
}

package main

import (
	"bytes"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coilysiren/coily/pkg/verbclass"
	"gopkg.in/yaml.v3"
)

// TestGeneratedFilesAreFresh asserts every committed
// pkg/ops/<binary>/generated.go matches the output of the current
// generator applied to the current manifest. Prior regression: the
// flag-typing commit didn't regenerate pkg/ops/aws/generated.go, so
// `coily aws ssm get-parameter --name ...` silently dropped --name at
// runtime. A drift check here fails fast instead of waiting for a user
// to trip over a missing flag.
func TestGeneratedFilesAreFresh(t *testing.T) {
	for _, bin := range []string{"aws", "gh", "kubectl"} {
		t.Run(bin, func(t *testing.T) {
			manifestPath := filepath.Join("..", "..", "configs", "commands", bin+".yaml")
			raw, err := os.ReadFile(manifestPath)
			if err != nil {
				t.Fatalf("read %s: %v", manifestPath, err)
			}
			var m Manifest
			if err := yaml.Unmarshal(raw, &m); err != nil {
				t.Fatalf("parse %s: %v", manifestPath, err)
			}
			code, err := render(m)
			if err != nil {
				t.Fatalf("render %s: %v", bin, err)
			}
			formatted, err := format.Source([]byte(code))
			if err != nil {
				t.Fatalf("gofmt %s: %v", bin, err)
			}
			committedPath := filepath.Join("..", "..", "pkg", "ops", bin, "generated.go")
			committed, err := os.ReadFile(committedPath)
			if err != nil {
				t.Fatalf("read %s: %v", committedPath, err)
			}
			if !bytes.Equal(formatted, committed) {
				t.Errorf("%s is stale; run `make gen-passthrough` and commit the result", committedPath)
			}
		})
	}
}

// TestRenderForwardsPositionalArgs locks in the fix for "gh api <endpoint>"
// and friends. The generated Action must append c.Args().Slice() after the
// flag block so trailing positionals reach the underlying tool. Likewise the
// ArgsFunc must hand positionals into policy.Enforce (mixed with any
// stringSlice flag values that already accumulated in `positional`).
func TestRenderForwardsPositionalArgs(t *testing.T) {
	m := Manifest{
		Binary: "gh",
		Commands: []ManifestCommand{
			{Path: []string{"api"}, Help: "Make a GitHub API request.", Flags: []ManifestFlag{{Name: "--method"}}},
		},
	}
	code, err := render(m)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	wants := []string{
		// Action appends positionals after the flag loop.
		`argv = append(argv, c.Args().Slice()...)`,
		// ArgsFunc folds positionals into the slice handed to policy.
		`positional = append(positional, c.Args().Slice()...)`,
		`return args, positional, c.String("token")`,
	}
	for _, w := range wants {
		if !strings.Contains(code, w) {
			t.Errorf("rendered code missing %q\n--- code ---\n%s", w, code)
		}
	}
}

// TestBuildTreeAssignsScopes is the smoke test for the leaf-classification
// pass. Verb classification itself lives in pkg/verbclass and has its own
// table-driven tests; this only checks that buildTree wires the bucket and
// scope onto the right nodes.
func TestBuildTreeAssignsScopes(t *testing.T) {
	m := Manifest{
		Binary: "aws",
		Commands: []ManifestCommand{
			{Path: []string{"route53"}, Help: "Route 53 group", Children: []string{"list-hosted-zones", "delete-hosted-zone", "create-hosted-zone"}},
			{Path: []string{"route53", "list-hosted-zones"}, Help: "List hosted zones."},
			{Path: []string{"route53", "delete-hosted-zone"}, Help: "Delete a hosted zone."},
			{Path: []string{"route53", "create-hosted-zone"}, Help: "Create a hosted zone."},
		},
	}
	root := buildTree(m)
	if len(root.Children) != 1 {
		t.Fatalf("got %d top-level children, want 1", len(root.Children))
	}
	r53 := root.Children[0]
	if r53.Name != "route53" {
		t.Fatalf("top-level name = %q, want route53", r53.Name)
	}
	// Group nodes get neither bucket nor scope.
	if r53.Mutating {
		t.Errorf("group node Mutating = true, want false")
	}
	if r53.Scope != "" {
		t.Errorf("group node Scope = %q, want empty", r53.Scope)
	}
	want := map[string]struct {
		mutating bool
		scope    string
	}{
		"list-hosted-zones":  {false, "aws.route53:read"},
		"create-hosted-zone": {true, "aws.route53:write"},
		"delete-hosted-zone": {true, "aws.route53:delete"},
	}
	for _, child := range r53.Children {
		w, ok := want[child.Name]
		if !ok {
			t.Fatalf("unexpected child %q", child.Name)
		}
		if child.Mutating != w.mutating {
			t.Errorf("%s Mutating = %v, want %v", child.Name, child.Mutating, w.mutating)
		}
		if child.Scope != w.scope {
			t.Errorf("%s Scope = %q, want %q", child.Name, child.Scope, w.scope)
		}
	}
}

// TestRenderEmitsScope spot-checks that the rendered template carries the
// scope through to the generated source. A regression here means an agent's
// token check would silently use the wrong scope.
func TestRenderEmitsScope(t *testing.T) {
	m := Manifest{
		Binary: "aws",
		Commands: []ManifestCommand{
			{Path: []string{"route53"}, Help: "group", Children: []string{"delete-hosted-zone"}},
			{Path: []string{"route53", "delete-hosted-zone"}, Help: "Delete a hosted zone."},
		},
	}
	code, err := render(m)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(code, `"aws.route53:delete"`) {
		t.Errorf("rendered code missing aws.route53:delete scope")
	}
	if !strings.Contains(code, "policy.Mutating") {
		t.Errorf("rendered code missing policy.Mutating for delete leaf")
	}
}

// TestVerbclassIsTheSourceOfTruth is a trivial dependency assertion. If
// pkg/verbclass changes its bucket assignment for a representative leaf,
// gen-passthrough picks it up automatically. Listed explicitly so the
// dependency is visible in test output.
func TestVerbclassIsTheSourceOfTruth(t *testing.T) {
	cases := []struct {
		path []string
		want verbclass.Bucket
	}{
		{[]string{"route53", "list-hosted-zones"}, verbclass.Read},
		{[]string{"route53", "create-hosted-zone"}, verbclass.Write},
		{[]string{"route53", "delete-hosted-zone"}, verbclass.Delete},
	}
	for _, tc := range cases {
		if got := verbclass.Classify(tc.path); got != tc.want {
			t.Errorf("Classify(%v) = %s, want %s (verbclass changed; update gen-passthrough golden)",
				tc.path, got, tc.want)
		}
	}
}

// TestMakeFlagSpec covers the per-flag conversion that drives the codegen.
// Defaults to StringFlag when Type is unset; maps the four known Type
// vocabulary values (string, bool, int, stringSlice) to the matching
// cli.*Flag constructor name.
func TestMakeFlagSpec(t *testing.T) {
	cases := []struct {
		in       ManifestFlag
		wantType string
		wantKind string
	}{
		{ManifestFlag{Name: "--method", Type: "string"}, "string", "StringFlag"},
		{ManifestFlag{Name: "--debug", Type: "bool"}, "bool", "BoolFlag"},
		{ManifestFlag{Name: "--limit", Type: "int"}, "int", "IntFlag"},
		{ManifestFlag{Name: "--field", Type: "stringSlice"}, "stringSlice", "StringSliceFlag"},
		// Unset type defaults to string.
		{ManifestFlag{Name: "--legacy"}, "string", "StringFlag"},
	}
	for _, tc := range cases {
		t.Run(tc.in.Name+"/"+tc.in.Type, func(t *testing.T) {
			got := makeFlagSpec(tc.in)
			wantBare := strings.TrimPrefix(tc.in.Name, "--")
			if got.Bare != wantBare {
				t.Errorf("Bare = %q, want %q", got.Bare, wantBare)
			}
			if got.Long != "--"+wantBare {
				t.Errorf("Long = %q, want %q", got.Long, "--"+wantBare)
			}
			if got.Type != tc.wantType {
				t.Errorf("Type = %q, want %q", got.Type, tc.wantType)
			}
			if got.FlagKind != tc.wantKind {
				t.Errorf("FlagKind = %q, want %q", got.FlagKind, tc.wantKind)
			}
		})
	}
}

// TestRender_FlagKinds asserts the generated source includes the right
// cli.*Flag constructor for each Type. This is the regression test for
// `--debug` getting StringFlag-d (bug doc 01) and `--field` losing values
// because it's repeatable (bug doc 04).
func TestRender_FlagKinds(t *testing.T) {
	m := Manifest{
		Binary: "gh",
		Commands: []ManifestCommand{
			{
				Path: []string{"api"},
				Flags: []ManifestFlag{
					{Name: "--method", Type: "string"},
					{Name: "--paginate", Type: "bool"},
					{Name: "--field", Type: "stringSlice"},
				},
			},
		},
	}
	code, err := render(m)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	wantSubstrings := []string{
		`&cli.StringFlag{Name: "method"}`,
		`&cli.BoolFlag{Name: "paginate"}`,
		`&cli.StringSliceFlag{Name: "field"}`,
		// bool forwarding: append flag without value
		`if c.Bool("paginate")`,
		`argv = append(argv, "--paginate")`,
		// stringSlice forwarding: repeated flag
		`for _, v := range c.StringSlice("field")`,
		`argv = append(argv, "--field", v)`,
	}
	for _, s := range wantSubstrings {
		if !strings.Contains(code, s) {
			t.Errorf("generated code missing substring %q.\nFull output:\n%s", s, code)
		}
	}
}

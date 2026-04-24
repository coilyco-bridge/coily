package main

import (
	"bytes"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestGeneratedFilesAreFresh asserts every committed
// pkg/ops/<binary>/generated.go matches the output of the current
// generator applied to the current manifest.
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
// flag block so trailing positionals reach the underlying tool.
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
		`argv = append(argv, c.Args().Slice()...)`,
		`positional = append(positional, c.Args().Slice()...)`,
		`return args, positional`,
	}
	for _, w := range wants {
		if !strings.Contains(code, w) {
			t.Errorf("rendered code missing %q\n--- code ---\n%s", w, code)
		}
	}
}

// TestRenderNoTokenFlag confirms the generator does NOT emit any
// coily-confirmation --token flag or scope metadata. Tokens were removed
// from coily because they added no security over the allowlist + audit +
// Claude Code deny rules - see docs/threat-model.md.
func TestRenderNoTokenFlag(t *testing.T) {
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
	forbidden := []string{
		"coily confirmation token",
		"policy.Mutating",
		"policy.ReadOnly",
		"verb.Token(c)",
		"verb.TokenFromEnv",
		`Scope:`,
		`Kind:`,
		`$COILY_TOKEN`,
	}
	for _, f := range forbidden {
		if strings.Contains(code, f) {
			t.Errorf("rendered code must not contain %q\n--- code ---\n%s", f, code)
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
// cli.*Flag constructor for each Type.
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
		`if c.Bool("paginate")`,
		`argv = append(argv, "--paginate")`,
		`for _, v := range c.StringSlice("field")`,
		`argv = append(argv, "--field", v)`,
	}
	for _, s := range wantSubstrings {
		if !strings.Contains(code, s) {
			t.Errorf("generated code missing substring %q.\nFull output:\n%s", s, code)
		}
	}
}

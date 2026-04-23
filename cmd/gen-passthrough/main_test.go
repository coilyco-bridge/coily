package main

import (
	"strings"
	"testing"
)

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

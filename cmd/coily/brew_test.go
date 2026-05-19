package main

import (
	"reflect"
	"testing"
)

func TestSplitBrewArgs(t *testing.T) {
	cases := []struct {
		name        string
		raw         []string
		wantAllow   bool
		wantForward []string
		wantForms   []string
	}{
		{
			name:        "single tap formula",
			raw:         []string{"coilysiren/tap/coily"},
			wantAllow:   false,
			wantForward: []string{"coilysiren/tap/coily"},
			wantForms:   []string{"coilysiren/tap/coily"},
		},
		{
			name:        "allow flag is consumed",
			raw:         []string{"--allow-untapped", "ripgrep"},
			wantAllow:   true,
			wantForward: []string{"ripgrep"},
			wantForms:   []string{"ripgrep"},
		},
		{
			name:        "force forwards through, formulae list excludes flags",
			raw:         []string{"--force", "coily"},
			wantAllow:   false,
			wantForward: []string{"--force", "coily"},
			wantForms:   []string{"coily"},
		},
		{
			name:        "bare upgrade",
			raw:         []string{},
			wantAllow:   false,
			wantForward: []string{},
			wantForms:   []string{},
		},
		{
			name:        "allow flag mixed with positionals",
			raw:         []string{"some-formula", "--allow-untapped", "--force"},
			wantAllow:   true,
			wantForward: []string{"some-formula", "--force"},
			wantForms:   []string{"some-formula"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotAllow, gotForward, gotForms := splitBrewArgs(tc.raw)
			if gotAllow != tc.wantAllow {
				t.Errorf("allow: got %v, want %v", gotAllow, tc.wantAllow)
			}
			if !reflect.DeepEqual(gotForward, tc.wantForward) {
				t.Errorf("forward: got %#v, want %#v", gotForward, tc.wantForward)
			}
			if !reflect.DeepEqual(gotForms, tc.wantForms) {
				t.Errorf("formulae: got %#v, want %#v", gotForms, tc.wantForms)
			}
		})
	}
}

// TestBrewServicesCommand_HasMutatingVerbs pins the closed-set surface
// for `coily brew services` (coily#249). start/stop/restart are the
// mutating verbs that need an audited wrapper. status is intentionally
// absent: it lives on `coily pkg brew services list` (read-only side).
func TestBrewServicesCommand_HasMutatingVerbs(t *testing.T) {
	r := &Runner{}
	cmd := r.brewServicesCommand()
	if cmd.Name != "services" {
		t.Fatalf("Name = %q, want \"services\"", cmd.Name)
	}
	got := make(map[string]bool, len(cmd.Commands))
	for _, c := range cmd.Commands {
		got[c.Name] = true
	}
	for _, want := range []string{"start", "stop", "restart"} {
		if !got[want] {
			t.Errorf("missing subcommand %q (got %v)", want, got)
		}
	}
	if got["status"] {
		t.Errorf("found `status` under brew services; coily#249 routes status to `coily pkg brew services list`")
	}
}

// TestBrewCommand_HasServicesSubcommand pins that `coily brew services`
// is reachable from the parent `coily brew` (coily#249).
func TestBrewCommand_HasServicesSubcommand(t *testing.T) {
	r := &Runner{}
	cmd := r.brewCommand()
	for _, c := range cmd.Commands {
		if c.Name == "services" {
			return
		}
	}
	got := make([]string, 0, len(cmd.Commands))
	for _, c := range cmd.Commands {
		got = append(got, c.Name)
	}
	t.Errorf("brewCommand missing `services` subcommand; got %v", got)
}

func TestBrewInTapScope(t *testing.T) {
	cases := map[string]bool{
		"coilysiren/tap/coily":       true,
		"coilysiren/tap/repo-recall": true,
		"coilysiren/tap/anything":    true,
		"coily":                      true,
		"repo-recall":                true,
		"arize-phoenix":              true,
		"ripgrep":                    false,
		"":                           false,
		"homebrew/core/wget":         false,
		"someuser/tap/coily":         false,
	}
	for f, want := range cases {
		t.Run(f, func(t *testing.T) {
			if got := brewInTapScope(f); got != want {
				t.Errorf("brewInTapScope(%q) = %v, want %v", f, got, want)
			}
		})
	}
}

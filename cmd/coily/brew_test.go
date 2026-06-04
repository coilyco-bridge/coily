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

// TestClassifyBrewInvocation pins the dispatcher's audit-name + scope-
// category mapping for the coily#253 single-entry-point brew surface.
// One table covers every category so future drift on any verb trips.
func TestClassifyBrewInvocation(t *testing.T) {
	r := &Runner{}
	cases := []struct {
		name     string
		argv     []string
		wantName string
	}{
		// Formula-scoped.
		{"install", []string{"install", "coily"}, "pkg.brew.install"},
		{"uninstall", []string{"uninstall", "coily"}, "pkg.brew.uninstall"},
		{"upgrade", []string{"upgrade", "coily"}, "pkg.brew.upgrade"},
		{"reinstall", []string{"reinstall", "coily"}, "pkg.brew.reinstall"},
		{"link", []string{"link", "coily"}, "pkg.brew.link"},
		{"unlink", []string{"unlink", "coily"}, "pkg.brew.unlink"},
		{"pin", []string{"pin", "coily"}, "pkg.brew.pin"},
		{"unpin", []string{"unpin", "coily"}, "pkg.brew.unpin"},
		// Tap-scoped.
		{"tap", []string{"tap", "coilysiren/homebrew-tap"}, "pkg.brew.tap"},
		{"untap", []string{"untap", "coilysiren/homebrew-tap"}, "pkg.brew.untap"},
		// Touch-everything.
		{"cleanup", []string{"cleanup"}, "pkg.brew.cleanup"},
		{"autoremove", []string{"autoremove"}, "pkg.brew.autoremove"},
		// Services formula-scoped.
		{"services start", []string{"services", "start", "coily"}, "pkg.brew.services.start"},
		{"services stop", []string{"services", "stop", "coily"}, "pkg.brew.services.stop"},
		{"services restart", []string{"services", "restart", "coily"}, "pkg.brew.services.restart"},
		{"services run", []string{"services", "run", "coily"}, "pkg.brew.services.run"},
		{"services kill", []string{"services", "kill", "coily"}, "pkg.brew.services.kill"},
		// Services touch-everything.
		{"services cleanup", []string{"services", "cleanup"}, "pkg.brew.services.cleanup"},
		// Passthrough.
		{"search", []string{"search", "ripgrep"}, "pkg.brew.search"},
		{"info", []string{"info", "ripgrep"}, "pkg.brew.info"},
		{"list", []string{"list"}, "pkg.brew.list"},
		{"update", []string{"update"}, "pkg.brew.update"},
		{"services list", []string{"services", "list"}, "pkg.brew.services.list"},
		{"services info", []string{"services", "info"}, "pkg.brew.services.info"},
		{"bare", []string{}, "pkg.brew"},
		{"unknown verb passes through", []string{"weirdverb"}, "pkg.brew.weirdverb"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotName, action, _ := r.classifyBrewInvocation(tc.argv)
			if gotName != tc.wantName {
				t.Errorf("name = %q, want %q (argv=%v)", gotName, tc.wantName, tc.argv)
			}
			if action == nil {
				t.Errorf("action is nil for argv=%v", tc.argv)
			}
		})
	}
}

// TestPkgBrewCommand_TopLevelShape pins the parent's single-entry-point
// shape: SkipFlagParsing on so brew's argv content (jq-style, --cask,
// markdown-bearing flags) survives, and no Commands subtree so the
// dispatcher owns routing.
func TestPkgBrewCommand_TopLevelShape(t *testing.T) {
	r := &Runner{}
	cmd := r.pkgBrewCommand()
	if cmd.Name != "brew" {
		t.Fatalf("Name = %q, want \"brew\"", cmd.Name)
	}
	if !cmd.SkipFlagParsing {
		t.Errorf("SkipFlagParsing must be true so brew argv flows through verbatim")
	}
	if len(cmd.Commands) != 0 {
		t.Errorf("pkgBrewCommand must not expose subcommands; the dispatcher routes internally. Got %d", len(cmd.Commands))
	}
	if cmd.Action == nil {
		t.Errorf("pkgBrewCommand must have an Action (the dispatcher)")
	}
}

func TestBrewInTapScope(t *testing.T) {
	orgs := defaultPrimaryOrgs()
	cases := map[string]bool{
		"coilysiren/tap/coily":               true,
		"coilysiren/tap/repo-recall":         true,
		"coilysiren/tap/anything":            true,
		"coilysiren/coily/coily":             true,
		"coilysiren/repo-recall/repo-recall": true,
		// post org-split primary orgs (coily#175)
		"coilyco-flight-deck/o2r/o2r": true,
		"coilyco-bridge/coily/coily":  true,
		"coily":                       true,
		"repo-recall":                 true,
		"arize-phoenix":               true,
		"ripgrep":                     false,
		"":                            false,
		"homebrew/core/wget":          false,
		"someuser/tap/coily":          false,
		"coilysiren/coily":            false,
	}
	for f, want := range cases {
		t.Run(f, func(t *testing.T) {
			if got := brewInTapScope(f, orgs); got != want {
				t.Errorf("brewInTapScope(%q) = %v, want %v", f, got, want)
			}
		})
	}
}

func TestBrewTapPositionalAllowed(t *testing.T) {
	orgs := defaultPrimaryOrgs()
	cases := map[string]bool{
		"coilysiren/tap":          true,
		"coilyco-flight-deck/o2r": true,
		"coilyco-bridge/coily":    true,
		"https://forgejo.coilysiren.me/coilyco-flight-deck/otel-a2a-relay-cli.git": true,
		"https://forgejo.coilysiren.me/coilysiren/tap":                             true,
		"someuser/tap":                    false,
		"https://github.com/someuser/tap": false,
	}
	for tap, want := range cases {
		t.Run(tap, func(t *testing.T) {
			if got := brewTapPositionalAllowed(tap, orgs); got != want {
				t.Errorf("brewTapPositionalAllowed(%q) = %v, want %v", tap, got, want)
			}
		})
	}
}

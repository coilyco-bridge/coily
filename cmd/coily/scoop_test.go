package main

import (
	"reflect"
	"testing"
)

func TestSplitScoopArgs(t *testing.T) {
	cases := []struct {
		name        string
		raw         []string
		wantAllow   bool
		wantForward []string
		wantPos     []string
	}{
		{
			name:        "single qualified app",
			raw:         []string{"coilysiren/coily"},
			wantAllow:   false,
			wantForward: []string{"coilysiren/coily"},
			wantPos:     []string{"coilysiren/coily"},
		},
		{
			name:        "allow flag is consumed",
			raw:         []string{"--allow-untapped", "git"},
			wantAllow:   true,
			wantForward: []string{"git"},
			wantPos:     []string{"git"},
		},
		{
			name:        "flag forwards through, positionals exclude flags",
			raw:         []string{"--global", "coily"},
			wantAllow:   false,
			wantForward: []string{"--global", "coily"},
			wantPos:     []string{"coily"},
		},
		{
			name:        "bare invocation",
			raw:         []string{},
			wantAllow:   false,
			wantForward: []string{},
			wantPos:     []string{},
		},
		{
			name:        "allow flag mixed with positionals",
			raw:         []string{"some-app", "--allow-untapped", "--global"},
			wantAllow:   true,
			wantForward: []string{"some-app", "--global"},
			wantPos:     []string{"some-app"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotAllow, gotForward, gotPos := splitScoopArgs(tc.raw)
			if gotAllow != tc.wantAllow {
				t.Errorf("allow: got %v, want %v", gotAllow, tc.wantAllow)
			}
			if !reflect.DeepEqual(gotForward, tc.wantForward) {
				t.Errorf("forward: got %#v, want %#v", gotForward, tc.wantForward)
			}
			if !reflect.DeepEqual(gotPos, tc.wantPos) {
				t.Errorf("positionals: got %#v, want %#v", gotPos, tc.wantPos)
			}
		})
	}
}

// TestClassifyScoopInvocation pins the dispatcher's audit-name + scope-
// category mapping for the coily#329 single-entry-point scoop surface.
// One table covers every category so future drift on any verb trips.
func TestClassifyScoopInvocation(t *testing.T) {
	r := &Runner{}
	cases := []struct {
		name     string
		argv     []string
		wantName string
	}{
		// App-scoped.
		{"install", []string{"install", "coily"}, "pkg.scoop.install"},
		{"uninstall", []string{"uninstall", "coily"}, "pkg.scoop.uninstall"},
		{"reset", []string{"reset", "coily"}, "pkg.scoop.reset"},
		{"hold", []string{"hold", "coily"}, "pkg.scoop.hold"},
		{"unhold", []string{"unhold", "coily"}, "pkg.scoop.unhold"},
		// Update splits by argv shape.
		{"update with app", []string{"update", "coily"}, "pkg.scoop.update"},
		{"bare update", []string{"update"}, "pkg.scoop.update"},
		{"update star", []string{"update", "*"}, "pkg.scoop.update"},
		// Bucket-scoped.
		{"bucket add", []string{"bucket", "add", "coilysiren"}, "pkg.scoop.bucket.add"},
		{"bucket rm", []string{"bucket", "rm", "coilysiren"}, "pkg.scoop.bucket.rm"},
		// Touch-everything.
		{"cleanup", []string{"cleanup"}, "pkg.scoop.cleanup"},
		// Passthrough.
		{"search", []string{"search", "git"}, "pkg.scoop.search"},
		{"info", []string{"info", "coily"}, "pkg.scoop.info"},
		{"list", []string{"list"}, "pkg.scoop.list"},
		{"status", []string{"status"}, "pkg.scoop.status"},
		{"bucket list", []string{"bucket", "list"}, "pkg.scoop.bucket.list"},
		{"bucket known", []string{"bucket", "known"}, "pkg.scoop.bucket.known"},
		{"bare bucket", []string{"bucket"}, "pkg.scoop.bucket"},
		{"bare", []string{}, "pkg.scoop"},
		{"unknown verb passes through", []string{"weirdverb"}, "pkg.scoop.weirdverb"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotName, action, _ := r.classifyScoopInvocation(tc.argv)
			if gotName != tc.wantName {
				t.Errorf("name = %q, want %q (argv=%v)", gotName, tc.wantName, tc.argv)
			}
			if action == nil {
				t.Errorf("action is nil for argv=%v", tc.argv)
			}
		})
	}
}

// TestPkgScoopCommand_TopLevelShape pins the parent's single-entry-point
// shape: SkipFlagParsing on so scoop's argv content survives, and no
// Commands subtree so the dispatcher owns routing.
func TestPkgScoopCommand_TopLevelShape(t *testing.T) {
	r := &Runner{}
	cmd := r.pkgScoopCommand()
	if cmd.Name != "scoop" {
		t.Fatalf("Name = %q, want \"scoop\"", cmd.Name)
	}
	if !cmd.SkipFlagParsing {
		t.Errorf("SkipFlagParsing must be true so scoop argv flows through verbatim")
	}
	if len(cmd.Commands) != 0 {
		t.Errorf("pkgScoopCommand must not expose subcommands; the dispatcher routes internally. Got %d", len(cmd.Commands))
	}
	if cmd.Action == nil {
		t.Errorf("pkgScoopCommand must have an Action (the dispatcher)")
	}
}

func TestScoopInBucketScope(t *testing.T) {
	cases := map[string]bool{
		"coilysiren/coily":       true,
		"coilysiren/anything":    true,
		"coily":                  true,
		"git":                    false,
		"":                       false,
		"someuser/coily":         false,
		"coilysiren/foo/bar":     false,
		"coilysiren":             false,
		"main/git":               false,
		"extras/notepadplusplus": false,
	}
	for a, want := range cases {
		t.Run(a, func(t *testing.T) {
			if got := scoopInBucketScope(a); got != want {
				t.Errorf("scoopInBucketScope(%q) = %v, want %v", a, got, want)
			}
		})
	}
}

func TestScoopBucketPositionalsInScope(t *testing.T) {
	cases := []struct {
		name        string
		positionals []string
		want        bool
	}{
		{"alias coilysiren", []string{"coilysiren"}, true},
		{"https url", []string{"coilysiren", "https://github.com/coilysiren/scoop-bucket"}, true},
		{"ssh url", []string{"mybucket", "git@github.com:coilysiren/scoop-bucket"}, true},
		{"alias only off-org", []string{"extras"}, false},
		{"https url off-org", []string{"extras", "https://github.com/ScoopInstaller/Extras"}, false},
		{"empty", []string{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := scoopBucketPositionalsInScope(tc.positionals); got != tc.want {
				t.Errorf("scoopBucketPositionalsInScope(%v) = %v, want %v", tc.positionals, got, tc.want)
			}
		})
	}
}

func TestScoopUpdateIsAll(t *testing.T) {
	cases := []struct {
		name        string
		positionals []string
		want        bool
	}{
		{"empty", []string{}, true},
		{"star", []string{"*"}, true},
		{"single app", []string{"coily"}, false},
		{"multiple apps", []string{"coily", "git"}, false},
		{"star plus app", []string{"*", "coily"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := scoopUpdateIsAll(tc.positionals); got != tc.want {
				t.Errorf("scoopUpdateIsAll(%v) = %v, want %v", tc.positionals, got, tc.want)
			}
		})
	}
}

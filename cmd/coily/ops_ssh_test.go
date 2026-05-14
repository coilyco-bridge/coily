package main

import (
	"strings"
	"testing"
)

// TestSystemctlVerbs_StatusIsNoSudo pins coilysiren/coily#144: status
// is a read-only inspection that systemd serves without privilege.
// Sudo-prefixing it broke non-tty SSH callers ("a terminal is required
// to read the password"). Mutating verbs (start/stop/restart/enable/
// disable/daemon-reload) keep sudo because they touch runtime state
// or /etc/systemd/system.
func TestSystemctlVerbs_StatusIsNoSudo(t *testing.T) {
	mustNoSudo := map[string]bool{"status": true}
	mustSudo := map[string]bool{
		"start": true, "stop": true, "restart": true,
		"enable": true, "disable": true, "daemon-reload": true,
	}
	seen := map[string]bool{}
	for _, v := range systemctlVerbs {
		seen[v.Name] = true
		switch {
		case mustNoSudo[v.Name] && !v.NoSudo:
			t.Errorf("verb %q must run NoSudo (read-only)", v.Name)
		case mustSudo[v.Name] && v.NoSudo:
			t.Errorf("verb %q must run sudo-prefixed (mutator)", v.Name)
		}
	}
	for name := range mustNoSudo {
		if !seen[name] {
			t.Errorf("verb %q missing from systemctlVerbs", name)
		}
	}
	for name := range mustSudo {
		if !seen[name] {
			t.Errorf("verb %q missing from systemctlVerbs", name)
		}
	}
}

func TestValidateUnitName(t *testing.T) {
	cases := []struct {
		in      string
		wantErr bool
	}{
		{"", true},
		{"-foo.service", true},
		{"foo.service", false},
		{"foo@bar.service", false},
		{"my_unit-1.service", false},
		{"foo;rm -rf /", true},
		{"foo`whoami`", true},
		{"foo$bar", true},
		{"a/b", true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			err := validateUnitName(tc.in)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateUnitName(%q) err=%v, wantErr=%v", tc.in, err, tc.wantErr)
			}
		})
	}
}

func TestValidateRepoPath(t *testing.T) {
	cases := []struct {
		in      string
		wantErr bool
	}{
		{"", true},
		{"relative/path", true},
		{"/", false},
		{"/home/kai/projects/infrastructure", false},
		{"/srv/eco-server", false},
		{"-/evil", true},
		{"/foo/../etc", true},
		{"/foo/..", true},
		{"/foo/bar baz", true},
		{"/foo\tbar", true},
		{"/foo\nbar", true},
		{"/repo.with.dots-and_dashes", false},
		{"~", false},
		{"~/", false},
		{"~/projects/coilysiren/infrastructure", false},
		{"~/foo/..", true},
		{"~root/etc", true},
		{"~/foo bar", true},
		{strings.Repeat("/a", 3000), true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			err := validateRepoPath(tc.in)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateRepoPath(%q) err=%v, wantErr=%v", tc.in, err, tc.wantErr)
			}
		})
	}
}

func TestValidateEcoModName(t *testing.T) {
	cases := []struct {
		in      string
		wantErr bool
	}{
		{"", true},
		{"EcoTelemetry", false},
		{"eco-telemetry", false},
		{"BunWulf_Educational", false},
		{"Mod.With.Dots", false},
		{"-flagy", true},
		{"foo/bar", true},
		{"foo bar", true},
		{"foo;rm", true},
		{"foo`whoami`", true},
		{"foo$bar", true},
		{strings.Repeat("a", 200), true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			err := validateEcoModName(tc.in)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateEcoModName(%q) err=%v, wantErr=%v", tc.in, err, tc.wantErr)
			}
		})
	}
}

func TestValidateGrepPattern(t *testing.T) {
	cases := []struct {
		in      string
		wantErr bool
	}{
		{"", true},
		{"hello", false},
		{"two words", false},
		{"path/to/thing", false},
		{"version=1.2.3", false},
		{"-flagy", true},
		{"has'quote", true},
		{strings.Repeat("a", 2000), true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			err := validateGrepPattern(tc.in)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateGrepPattern(%q) err=%v, wantErr=%v", tc.in, err, tc.wantErr)
			}
		})
	}
}

func TestPosixShellQuote(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", "''"},
		{"foo", "'foo'"},
		{"two words", "'two words'"},
		{"-A", "'-A'"},
		{"{.items[*].metadata.name}", "'{.items[*].metadata.name}'"},
		{"app=foo,env=bar", "'app=foo,env=bar'"},
		{"a in (b,c)", "'a in (b,c)'"},
		{"with $var", "'with $var'"},
		{"with `backtick`", "'with `backtick`'"},
		{"semi;colon", "'semi;colon'"},
		{"single'quote", `'single'\''quote'`},
		{"two''quotes", `'two'\'''\''quotes'`},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := posixShellQuote(tc.in); got != tc.want {
				t.Errorf("posixShellQuote(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

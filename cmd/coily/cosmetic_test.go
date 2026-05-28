package main

import (
	"strings"
	"testing"

	"github.com/coilysiren/cli-guard/policy"
)

func TestCosmeticSanitize(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    string
		changed bool
	}{
		{"no metachar", "a clean title", "a clean title", false},
		{"empty", "", "", false},
		{"semicolon", "a; b", "a - b", true},
		{"semicolon from issue", "cap override; relocate", "cap override - relocate", true},
		{"pipe", "a|b", "a/b", true},
		{"ampersand", "tom & jerry", "tom and jerry", true},
		{"angle brackets", "a <b> c", "a b c", true},
		{"backtick and dollar", "use `cmd` and $x", "use cmd and x", true},
		{"parens braces", "f(x){y}", "f x y", true},
		{"backslash", `a\b`, "a b", true},
		{"control chars", "a\tb\nc", "a b c", true},
		{"collapse spaces around semicolon", "a ; b", "a - b", true},
		{"multiple metachars", "a;b|c&d", "a - b/c and d", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, changed := cosmeticSanitize(tc.in)
			if got != tc.want {
				t.Errorf("cosmeticSanitize(%q) = %q, want %q", tc.in, got, tc.want)
			}
			if changed != tc.changed {
				t.Errorf("cosmeticSanitize(%q) changed = %v, want %v", tc.in, changed, tc.changed)
			}
			// Sanitized output must always be free of shell metacharacters so
			// it passes the policy gate unchanged.
			if strings.ContainsAny(got, policy.ShellMeta) {
				t.Errorf("cosmeticSanitize(%q) = %q still contains a shell metacharacter", tc.in, got)
			}
		})
	}
}

// TestCosmeticSanitizeClearsAllShellMeta proves every byte the policy gate
// rejects is removed, so any allowlisted cosmetic value clears the gate.
func TestCosmeticSanitizeClearsAllShellMeta(t *testing.T) {
	for _, b := range []byte(policy.ShellMeta) {
		in := "x" + string(b) + "y"
		got, changed := cosmeticSanitize(in)
		if !changed {
			t.Errorf("byte %q not treated as a metacharacter", b)
		}
		if strings.ContainsAny(got, policy.ShellMeta) {
			t.Errorf("sanitizing %q left a metacharacter: %q", in, got)
		}
	}
}

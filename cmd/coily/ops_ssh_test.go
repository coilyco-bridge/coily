package main

import (
	"strings"
	"testing"
)

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

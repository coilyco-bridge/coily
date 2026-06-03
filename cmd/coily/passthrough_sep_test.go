package main

import (
	"strings"
	"testing"
)

func TestStripPassthroughSeparator(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "ops aws with trailing query/output (the #37 repro)",
			in:   []string{"coily", "ops", "aws", "--", "sts", "get-caller-identity", "--query", "Account", "--output", "text"},
			want: []string{"coily", "ops", "aws", "sts", "get-caller-identity", "--query", "Account", "--output", "text"},
		},
		{
			name: "ops aws with leading global flag",
			in:   []string{"coily", "--commit-scope=/x", "ops", "aws", "--", "ssm", "get-parameter", "--name", "/p", "--query", "Parameter.Value"},
			want: []string{"coily", "--commit-scope=/x", "ops", "aws", "ssm", "get-parameter", "--name", "/p", "--query", "Parameter.Value"},
		},
		{
			name: "ops gh separator",
			in:   []string{"coily", "ops", "gh", "--", "api", "x"},
			want: []string{"coily", "ops", "gh", "api", "x"},
		},
		{
			name: "pkg group bin",
			in:   []string{"coily", "pkg", "npm", "--", "run", "build"},
			want: []string{"coily", "pkg", "npm", "run", "build"},
		},
		{
			name: "top-level passthrough bin",
			in:   []string{"coily", "docker", "--", "ps", "-a"},
			want: []string{"coily", "docker", "ps", "-a"},
		},
		{
			name: "no separator: unchanged",
			in:   []string{"coily", "ops", "aws", "sts", "get-caller-identity", "--query", "Account"},
			want: []string{"coily", "ops", "aws", "sts", "get-caller-identity", "--query", "Account"},
		},
		{
			name: "tool's own -- is NOT stripped (after a non-bin token)",
			in:   []string{"coily", "ops", "aws", "s3", "cp", "--", "file"},
			want: []string{"coily", "ops", "aws", "s3", "cp", "--", "file"},
		},
		{
			name: "ops bin name appearing as a deep arg is not a command pos",
			in:   []string{"coily", "ops", "gh", "api", "aws", "--", "x"},
			want: []string{"coily", "ops", "gh", "api", "aws", "--", "x"},
		},
		{
			name: "unknown group bin: unchanged",
			in:   []string{"coily", "ops", "notabin", "--", "x"},
			want: []string{"coily", "ops", "notabin", "--", "x"},
		},
	}
	for _, c := range cases {
		got := stripPassthroughSeparator(c.in)
		if strings.Join(got, " ") != strings.Join(c.want, " ") {
			t.Errorf("%s:\n  in   %v\n  got  %v\n  want %v", c.name, c.in, got, c.want)
		}
	}
}

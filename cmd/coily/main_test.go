package main

import (
	"reflect"
	"testing"
)

// TestLiftCommitScope verifies that --commit-scope is hoisted out of any
// position in argv to a global-flag slot right after argv[0]. urfave/cli
// requires global flags to precede the verb chain, but passthrough verbs
// use SkipFlagParsing, so without this lift the flag would be consumed by
// the wrapped binary's argv and trigger scope_unresolved (issue #101).
func TestLiftCommitScope(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "no flag is a no-op",
			in:   []string{"coily", "ops", "gh", "issue", "list", "--repo", "coilysiren/coily"},
			want: []string{"coily", "ops", "gh", "issue", "list", "--repo", "coilysiren/coily"},
		},
		{
			name: "equals form lifts from after passthrough",
			in:   []string{"coily", "ops", "gh", "--commit-scope=/tmp/repo", "issue", "list"},
			want: []string{"coily", "--commit-scope=/tmp/repo", "ops", "gh", "issue", "list"},
		},
		{
			name: "two-token form lifts from after passthrough",
			in:   []string{"coily", "ops", "gh", "issue", "list", "--commit-scope", "/tmp/repo"},
			want: []string{"coily", "--commit-scope=/tmp/repo", "ops", "gh", "issue", "list"},
		},
		{
			name: "already-correct position is normalized to equals form",
			in:   []string{"coily", "--commit-scope=/tmp/repo", "ops", "gh", "issue", "list"},
			want: []string{"coily", "--commit-scope=/tmp/repo", "ops", "gh", "issue", "list"},
		},
		{
			name: "last occurrence wins",
			in:   []string{"coily", "ops", "gh", "--commit-scope=/a", "x", "--commit-scope=/b"},
			want: []string{"coily", "--commit-scope=/b", "ops", "gh", "x"},
		},
		{
			name: "bare --commit-scope followed by a flag is dropped",
			in:   []string{"coily", "ops", "gh", "--commit-scope", "--repo", "coilysiren/coily"},
			want: []string{"coily", "ops", "gh", "--repo", "coilysiren/coily"},
		},
		{
			name: "literal substring inside a body argv element is left alone",
			in:   []string{"coily", "ops", "gh", "issue", "create", "--body", "see --commit-scope=/x in docs"},
			want: []string{"coily", "ops", "gh", "issue", "create", "--body", "see --commit-scope=/x in docs"},
		},
		{
			name: "empty argv",
			in:   []string{},
			want: []string{},
		},
		{
			name: "single-element argv",
			in:   []string{"coily"},
			want: []string{"coily"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := liftCommitScope(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("liftCommitScope(%v)\n got = %v\nwant = %v", tc.in, got, tc.want)
			}
		})
	}
}

package main

import (
	"reflect"
	"testing"
)

func TestGitVerbRewriter(t *testing.T) {
	cases := []struct {
		name string
		verb string
		argv []string
		want []string
	}{
		{"bare verb", "status", nil, []string{"status"}},
		{"verb with args", "log", []string{"--oneline", "-3"}, []string{"log", "--oneline", "-3"}},
		{"hoists -C", "pull", []string{"-C", "/repo"}, []string{"-C", "/repo", "pull"}},
		{"hoists -C with trailing args", "log", []string{"-C", "/repo", "--oneline"}, []string{"-C", "/repo", "log", "--oneline"}},
		{"lone -C is not hoisted", "status", []string{"-C"}, []string{"status", "-C"}},
	}
	for _, c := range cases {
		got := gitVerbRewriter(c.verb)(c.argv)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: gitVerbRewriter(%q)(%v) = %v, want %v", c.name, c.verb, c.argv, got, c.want)
		}
	}
}

func TestGitPassthroughCommandsCoverVerbs(t *testing.T) {
	r := &Runner{}
	cmds := r.gitPassthroughCommands()
	if len(cmds) != len(gitPassthroughVerbs) {
		t.Fatalf("built %d passthrough commands, want %d", len(cmds), len(gitPassthroughVerbs))
	}
	for i, v := range gitPassthroughVerbs {
		if cmds[i].Name != v.name {
			t.Errorf("command %d named %q, want %q", i, cmds[i].Name, v.name)
		}
	}
}

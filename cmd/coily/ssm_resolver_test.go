package main

import (
	"testing"
)

func TestSSMPathFromEnvName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"COILYSIREN_TAILNET_DOMAIN", "/coilysiren/tailnet/domain"},
		{"LUCA_REPO_RECALL_URL", "/luca/repo/recall/url"},
		{"SINGLE", "/single"},
	}
	for _, c := range cases {
		got := ssmPathFromEnvName(c.in)
		if got != c.want {
			t.Errorf("ssmPathFromEnvName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

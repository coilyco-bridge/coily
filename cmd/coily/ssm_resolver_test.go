package main

import (
	"testing"
)

func TestEnvFromSSMResolver_ParentEnvWins(t *testing.T) {
	// When the env var is already set, the resolver returns it verbatim and
	// never touches SSM (so this runs without AWS creds).
	t.Setenv("NETLIFY_AUTH_TOKEN", "from-parent-env")
	resolve := envFromSSMResolver(map[string]string{
		"NETLIFY_AUTH_TOKEN": "/coilysiren/netlify/token",
	})
	got, err := resolve()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got["NETLIFY_AUTH_TOKEN"] != "from-parent-env" {
		t.Errorf("NETLIFY_AUTH_TOKEN = %q, want parent-env value", got["NETLIFY_AUTH_TOKEN"])
	}
}

func TestEnvFromSSMResolver_EmptyMap(t *testing.T) {
	resolve := envFromSSMResolver(map[string]string{})
	got, err := resolve()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}

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

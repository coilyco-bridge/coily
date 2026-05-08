package sentry

import "testing"

func TestCommandTreeNotEmpty(t *testing.T) {
	cmd := Command(nil, nil)
	if cmd.Name != "sentry" {
		t.Fatalf("root name = %q, want sentry", cmd.Name)
	}
	if len(cmd.Commands) == 0 {
		t.Fatalf("no top-level subcommands")
	}
}

func TestSSMParamPath(t *testing.T) {
	if ssmAPIKey != "/sentry/api-key" {
		t.Errorf("ssmAPIKey = %q", ssmAPIKey)
	}
}

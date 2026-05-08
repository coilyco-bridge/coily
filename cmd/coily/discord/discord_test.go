package discord

import "testing"

func TestCommandTreeNotEmpty(t *testing.T) {
	cmd := Command(nil, nil)
	if cmd.Name != "discord" {
		t.Fatalf("root name = %q, want discord", cmd.Name)
	}
	if len(cmd.Commands) == 0 {
		t.Fatalf("no top-level subcommands")
	}
	leaves := 0
	for _, sub := range cmd.Commands {
		leaves += len(sub.Commands)
	}
	if leaves < 50 {
		t.Errorf("expected many leaves from a 1MB OpenAPI spec, got %d", leaves)
	}
}

func TestSSMParamPath(t *testing.T) {
	if ssmAPIKey != "/discord/bot-token" {
		t.Errorf("ssmAPIKey = %q", ssmAPIKey)
	}
}

func TestAPIBaseStable(t *testing.T) {
	if apiBase != "https://discord.com/api/v10" {
		t.Errorf("apiBase = %q", apiBase)
	}
}

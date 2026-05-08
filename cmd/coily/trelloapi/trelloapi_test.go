package trelloapi

import "testing"

func TestCommandTreeNotEmpty(t *testing.T) {
	cmd := Command(nil, nil)
	if cmd.Name != "trelloapi" {
		t.Fatalf("root name = %q, want trelloapi", cmd.Name)
	}
	if len(cmd.Commands) == 0 {
		t.Fatalf("no top-level subcommands")
	}
}

func TestSSMParamPaths(t *testing.T) {
	if ssmAPIKey != "/trello/api-key" {
		t.Errorf("ssmAPIKey = %q", ssmAPIKey)
	}
	if ssmAPIToken != "/trello/api-token" {
		t.Errorf("ssmAPIToken = %q", ssmAPIToken)
	}
}

package glama

import "testing"

// TestCommandTreeNotEmpty smoke-tests that the generator produced a
// non-empty cli.Command tree. The generator's correctness is validated
// in scripts/openapi-to-coily_test.py against fixtures; this asserts
// that the wired output reaches the binary.
func TestCommandTreeNotEmpty(t *testing.T) {
	cmd := Command(nil, nil)
	if cmd.Name != "glama" {
		t.Fatalf("root name = %q, want glama", cmd.Name)
	}
	if len(cmd.Commands) == 0 {
		t.Fatalf("no top-level subcommands")
	}
	leaves := 0
	for _, sub := range cmd.Commands {
		leaves += len(sub.Commands)
	}
	if leaves == 0 {
		t.Errorf("no leaf operations across %d groups", len(cmd.Commands))
	}
}

func TestSSMParamPath(t *testing.T) {
	if ssmAPIKey != "/glama/api-key" {
		t.Errorf("ssmAPIKey = %q, want /glama/api-key", ssmAPIKey)
	}
}

func TestAPIBaseStable(t *testing.T) {
	if apiBase != "https://glama.ai/api/mcp" {
		t.Errorf("apiBase = %q", apiBase)
	}
}

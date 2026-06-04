package main

import (
	"context"
	"strings"
	"testing"
)

// forgejo pr create is a hard block: Kai's release flow pushes straight to
// main, so coily never opens a PR. The Action must always error, and the
// message must explain the commit-to-main workflow.
func TestForgejoPRCreate_Disabled(t *testing.T) {
	r := &Runner{}
	cmd := r.forgejoPRCreateCommand()
	if err := cmd.Action(context.Background(), cmd); err == nil {
		t.Fatal("forgejo pr create Action = nil, want hard error")
	}
	for _, want := range []string{"disabled", "merge", "straight to main"} {
		if !strings.Contains(strings.ToLower(forgejoPRCreateBlockMessage), want) {
			t.Errorf("block message missing %q:\n%s", want, forgejoPRCreateBlockMessage)
		}
	}
}

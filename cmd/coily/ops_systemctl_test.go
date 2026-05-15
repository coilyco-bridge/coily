package main

import (
	"sort"
	"testing"
)

// TestSystemctlCommand_HasAllVerbs pins the closed-set surface so a future
// addition to systemctlVerbs auto-shows up in `coily systemctl` (and a
// future removal can't accidentally silently land).
func TestSystemctlCommand_HasAllVerbs(t *testing.T) {
	r := &Runner{}
	cmd := r.systemctlCommand()
	if cmd.Name != "systemctl" {
		t.Fatalf("Name = %q", cmd.Name)
	}
	got := make([]string, 0, len(cmd.Commands))
	for _, c := range cmd.Commands {
		got = append(got, c.Name)
	}
	sort.Strings(got)
	want := []string{"daemon-reload", "disable", "enable", "restart", "start", "status", "stop"}
	if len(got) != len(want) {
		t.Fatalf("verbs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("verbs[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

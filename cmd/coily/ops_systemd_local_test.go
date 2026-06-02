package main

import (
	"os"
	"strings"
	"testing"
)

// TestHostIsLocal_MatchesLeadingSegment pins coilysiren/coily#135's
// detection rule: the comparison is on the first dotted segment of
// hostname, so a target named "kai-server" matches a local
// "kai-server", "kai-server.local", "kai-server.tail-scale.ts.net"
// equally. Empty target and Hostname errors fall through to false, so a
// non-local host is rejected (errRemoteRemoved) rather than run locally.
func TestHostIsLocal_MatchesLeadingSegment(t *testing.T) {
	h, err := os.Hostname()
	if err != nil {
		t.Skip("os.Hostname unavailable; cannot exercise positive case")
	}
	leadingLocal := strings.SplitN(h, ".", 2)[0]
	if !hostIsLocal(leadingLocal) {
		t.Errorf("hostIsLocal(local hostname %q) = false, want true", leadingLocal)
	}
	if !hostIsLocal(strings.ToUpper(leadingLocal)) {
		t.Errorf("hostIsLocal should be case-insensitive on the leading segment")
	}
	// FQDN-shaped target whose leading segment matches local.
	if !hostIsLocal(leadingLocal + ".example.invalid") {
		t.Errorf("hostIsLocal(%q) = false, want true (leading segment matches)", leadingLocal+".example.invalid")
	}
}

func TestHostIsLocal_RejectsMismatchAndEmpty(t *testing.T) {
	if hostIsLocal("") {
		t.Error("hostIsLocal(\"\") should be false")
	}
	if hostIsLocal("definitely-not-this-hostname-xyz123") {
		t.Error("hostIsLocal of an arbitrary string should be false")
	}
}

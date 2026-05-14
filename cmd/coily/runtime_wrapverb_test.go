package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coilysiren/cli-guard/audit"
	"github.com/coilysiren/cli-guard/config"
	"github.com/coilysiren/cli-guard/profiles"
	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// runWrapVerbAndReadAudit drives one verb through r.WrapVerb and
// returns the last audit row plus the raw bytes. Closes the writer
// before reading so the buffered append is flushed.
func runWrapVerbAndReadAudit(t *testing.T, r *Runner) audit.Record {
	t.Helper()
	wrapped := r.WrapVerb(verb.Spec{
		Name:      "test.wrapverb",
		SkipScope: true,
		Action:    func(_ context.Context, _ *cli.Command) error { return nil },
	}, r.Audit)
	cmd := &cli.Command{Name: "test", Action: wrapped}
	if err := cmd.Run(context.Background(), []string{"test"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if err := r.Audit.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	b, err := os.ReadFile(r.Audit.Path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	lines := bytes.Split(bytes.TrimSpace(b), []byte("\n"))
	var rec audit.Record
	if err := json.Unmarshal(lines[len(lines)-1], &rec); err != nil {
		t.Fatalf("unmarshal: %v line=%s", err, lines[len(lines)-1])
	}
	return rec
}

// Phase 5+6 retired the audit.profile_aware feature flag. Every audit
// row from a WrapVerb-wrapped verb now carries a ProfileDecision.

func TestWrapVerb_FlagOn_SentinelButNoOverride_SourceMissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	sid := "sess-sentinel-no-override"
	t.Setenv(sessionEnvVar, sid)
	sentinel := filepath.Join(dir, ".coily", "audit", "sessions", sid, "profile")
	if err := os.MkdirAll(filepath.Dir(sentinel), 0o700); err != nil {
		t.Fatalf("mkdir sentinel: %v", err)
	}
	if err := os.WriteFile(sentinel, []byte("mobile\n"), 0o600); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}

	r := newTestRunner(t)
	r.Cfg = &config.Config{}
	rec := runWrapVerbAndReadAudit(t, r)
	if rec.ProfileDecision == nil {
		t.Fatal("flag on: expected decision attached")
	}
	if rec.ProfileDecision.Source != string(profiles.SourceMissingFile) {
		t.Errorf("source = %q, want missing_file", rec.ProfileDecision.Source)
	}
	if !rec.ProfileDecision.Allowed {
		t.Error("phase-4 plumbing must always allow")
	}
	if rec.ProfileDecision.Coordinate.DataSecurity != "max" {
		t.Errorf("data_security = %q, want max (Strictest fallback)",
			rec.ProfileDecision.Coordinate.DataSecurity)
	}
}

func TestWrapVerb_FlagOn_OverrideHit(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	override := filepath.Join(dir, ".coily", "coily.yaml")
	if err := os.MkdirAll(filepath.Dir(override), 0o700); err != nil {
		t.Fatalf("mkdir override: %v", err)
	}
	if err := os.WriteFile(override, profiles.DefaultYAML, 0o600); err != nil {
		t.Fatalf("write override: %v", err)
	}

	sid := "sess-flagon-hit"
	t.Setenv(sessionEnvVar, sid)
	sentinel := filepath.Join(dir, ".coily", "audit", "sessions", sid, "profile")
	if err := os.MkdirAll(filepath.Dir(sentinel), 0o700); err != nil {
		t.Fatalf("mkdir sentinel: %v", err)
	}
	if err := os.WriteFile(sentinel, []byte("mobile\n"), 0o600); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}

	r := newTestRunner(t)
	r.Cfg = &config.Config{}
	rec := runWrapVerbAndReadAudit(t, r)
	if rec.ProfileDecision == nil {
		t.Fatal("expected decision attached on override hit")
	}
	if rec.ProfileDecision.Source != string(profiles.SourceOverride) {
		t.Errorf("source = %q, want override", rec.ProfileDecision.Source)
	}
	if rec.ProfileDecision.Profile != "mobile" {
		t.Errorf("profile = %q, want mobile", rec.ProfileDecision.Profile)
	}
	if rec.ProfileDecision.Coordinate.NetworkEgress != "allowlisted" {
		t.Errorf("network_egress = %q, want allowlisted (mobile)",
			rec.ProfileDecision.Coordinate.NetworkEgress)
	}
}

func TestWrapVerb_FlagOn_NoSentinel_SourceUnset(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv(sessionEnvVar, "sess-no-sentinel")

	override := filepath.Join(dir, ".coily", "coily.yaml")
	_ = os.MkdirAll(filepath.Dir(override), 0o700)
	_ = os.WriteFile(override, profiles.DefaultYAML, 0o600)

	r := newTestRunner(t)
	r.Cfg = &config.Config{}
	rec := runWrapVerbAndReadAudit(t, r)
	if rec.ProfileDecision == nil {
		t.Fatal("expected decision attached")
	}
	if rec.ProfileDecision.Source != string(profiles.SourceUnset) {
		t.Errorf("source = %q, want unset", rec.ProfileDecision.Source)
	}
	if !strings.HasPrefix(rec.ProfileDecision.Reason, "no profile selected") {
		t.Errorf("reason = %q, want unset note", rec.ProfileDecision.Reason)
	}
}

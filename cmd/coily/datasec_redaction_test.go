package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/cli/decision"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/cli/profiles"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/cli/verb"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/pkg/audit"
	"github.com/urfave/cli/v3"
)

// runWithArgvAndReadRow drives one verb whose argv includes a secret
// flag, through r.WrapVerb, and returns the persisted audit row.
func runWithArgvAndReadRow(t *testing.T, r *Runner, argv []string) audit.Record {
	t.Helper()
	wrapped := r.WrapVerb(verb.Spec{
		Name:       "test.datasec",
		SkipPolicy: true,
		ArgsFunc:   func(_ *cli.Command) (map[string]string, []string) { return nil, argv },
		Action:     func(_ context.Context, _ *cli.Command) error { return nil },
	}, r.Audit)
	// Drive os.Args so the audit row captures the secret-bearing argv
	// the way it would for a real invocation.
	defer func(orig []string) { os.Args = orig }(os.Args)
	os.Args = append([]string{"coily"}, argv...)
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

func setupRedactionEnv(t *testing.T, profileName string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	sid := "sess-redact"
	t.Setenv(sessionEnvVar, sid)

	if profileName != "" {
		override := filepath.Join(dir, ".coily", "coily.yaml")
		if err := os.MkdirAll(filepath.Dir(override), 0o700); err != nil {
			t.Fatalf("mkdir override: %v", err)
		}
		if err := os.WriteFile(override, profiles.DefaultYAML, 0o600); err != nil {
			t.Fatalf("write override: %v", err)
		}
		sentinel := filepath.Join(dir, ".coily", "audit", "sessions", sid, "profile")
		if err := os.MkdirAll(filepath.Dir(sentinel), 0o700); err != nil {
			t.Fatalf("mkdir sentinel: %v", err)
		}
		if err := os.WriteFile(sentinel, []byte(profileName+"\n"), 0o600); err != nil {
			t.Fatalf("write sentinel: %v", err)
		}
	}
}

// Strictest (no override on disk) drops argv entirely when a secret
// pattern matches. data_security=max.
func TestRedaction_StrictestDropsArgv(t *testing.T) {
	setupRedactionEnv(t, "")
	r := newTestRunner(t)
	r.Cfg = &Config{}
	r.Audit.SetRedactPolicy(decision.RedactPolicy())

	rec := runWithArgvAndReadRow(t, r, []string{"ops", "aws", "ssm", "--value=hunter2"})
	if len(rec.Argv) != 0 {
		t.Errorf("expected argv dropped at max, got %v", rec.Argv)
	}
	if rec.Verb != "test.datasec" {
		t.Errorf("verb missing on max-redacted row: %q", rec.Verb)
	}
}

// mac-tower profile = data_security:medium. Secret flag value is
// replaced with [REDACTED] but argv structure preserved.
func TestRedaction_MacTowerRedactsValue(t *testing.T) {
	setupRedactionEnv(t, "mac-tower")
	r := newTestRunner(t)
	r.Cfg = &Config{}
	r.Audit.SetRedactPolicy(decision.RedactPolicy())

	rec := runWithArgvAndReadRow(t, r, []string{"ops", "aws", "ssm", "--value=hunter2"})
	if len(rec.Argv) == 0 {
		t.Fatalf("expected argv preserved at medium, got empty")
	}
	found := false
	for _, tok := range rec.Argv {
		if tok == "--value=[REDACTED]" {
			found = true
		}
		if tok == "--value=hunter2" {
			t.Errorf("medium failed to redact: %v", rec.Argv)
		}
	}
	if !found {
		t.Errorf("expected --value=[REDACTED] in argv, got %v", rec.Argv)
	}
}

// Web profile = data_security:max. Argv dropped.
func TestRedaction_WebProfileDropsArgv(t *testing.T) {
	setupRedactionEnv(t, "web")
	r := newTestRunner(t)
	r.Cfg = &Config{}
	r.Audit.SetRedactPolicy(decision.RedactPolicy())

	rec := runWithArgvAndReadRow(t, r, []string{"ops", "aws", "--password=x"})
	if len(rec.Argv) != 0 {
		t.Errorf("expected argv dropped at max, got %v", rec.Argv)
	}
}

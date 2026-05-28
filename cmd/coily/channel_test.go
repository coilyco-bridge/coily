package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func channelTestRunner(t *testing.T) *Runner {
	t.Helper()
	cfg := defaultConfig()
	return &Runner{Cfg: &cfg}
}

func TestChannelCommand_TreeShape(t *testing.T) {
	r := channelTestRunner(t)
	cmd := r.channelCommand()
	if cmd.Name != "channel" {
		t.Fatalf("root name = %q, want channel", cmd.Name)
	}
	want := []string{"list", "create", "post", "read", "state", "spec", "events", "close"}
	got := make([]string, 0, len(cmd.Commands))
	for _, sub := range cmd.Commands {
		got = append(got, sub.Name)
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("subcommands = %v, want %v", got, want)
	}
}

func TestChannelDefaults(t *testing.T) {
	c := defaultConfig()
	if c.Channel.BaseURL != "http://api" {
		t.Errorf("default base URL = %q, want http://api", c.Channel.BaseURL)
	}
	if c.Channel.SSMTokenPath != "/coilysiren/backend/datastore-token" {
		t.Errorf("default ssm token path = %q", c.Channel.SSMTokenPath)
	}
}

func TestChannelID_Validation(t *testing.T) {
	t.Run("valid_4char", func(t *testing.T) {
		id, err := channelID("read", []string{"VHGC"})
		if err != nil || id != "VHGC" {
			t.Errorf("channelID(VHGC) = %q, %v", id, err)
		}
	})
	t.Run("missing", func(t *testing.T) {
		if _, err := channelID("read", nil); err == nil {
			t.Error("channelID() with no arg: want error")
		}
	})
	t.Run("too_many", func(t *testing.T) {
		if _, err := channelID("read", []string{"A", "B"}); err == nil {
			t.Error("channelID(A,B): want error")
		}
	})
	t.Run("invalid_metachar", func(t *testing.T) {
		if _, err := channelID("read", []string{"VH;C"}); err == nil {
			t.Error("channelID(VH;C): want rejection")
		}
	})
	t.Run("too_long", func(t *testing.T) {
		if _, err := channelID("read", []string{strings.Repeat("A", 32)}); err == nil {
			t.Error("channelID(32A): want rejection")
		}
	})
}

func TestParseChannelList_Shapes(t *testing.T) {
	t.Run("bare_array", func(t *testing.T) {
		rows, err := parseChannelList([]byte(`[{"id":"VHGC","title":"t","created_by":"a","created_at":"x","closed_at":null}]`))
		if err != nil {
			t.Fatalf("bare array: %v", err)
		}
		if len(rows) != 1 || rows[0].ID != "VHGC" || rows[0].ClosedAt != "" {
			t.Errorf("bare array -> %#v", rows)
		}
	})
	t.Run("wrapper", func(t *testing.T) {
		rows, err := parseChannelList([]byte(`{"channels":[{"id":"ABCD","closed_at":"2026-01-01T00:00:00Z"}]}`))
		if err != nil {
			t.Fatalf("wrapper: %v", err)
		}
		if len(rows) != 1 || rows[0].ID != "ABCD" || rows[0].ClosedAt == "" {
			t.Errorf("wrapper -> %#v", rows)
		}
	})
	t.Run("empty", func(t *testing.T) {
		rows, err := parseChannelList([]byte("  "))
		if err != nil || rows != nil {
			t.Errorf("empty -> %#v, %v", rows, err)
		}
	})
	t.Run("invalid", func(t *testing.T) {
		if _, err := parseChannelList([]byte("not json")); err == nil {
			t.Error("invalid JSON: want error")
		}
	})
}

func TestRenderChannelTable_Status(t *testing.T) {
	body := []byte(`[{"id":"OPEN","closed_at":null},{"id":"DONE","closed_at":"2026-01-01T00:00:00Z"}]`)
	out := captureStdout(t, func() error { return renderChannelTable(body) })
	if !strings.Contains(out, "OPEN") || !strings.Contains(out, "CLOSED") {
		t.Errorf("table missing status columns:\n%s", out)
	}
	if !strings.Contains(out, "ID") || !strings.Contains(out, "STATUS") {
		t.Errorf("table missing header:\n%s", out)
	}
}

func TestRenderChannelTable_Empty(t *testing.T) {
	out := captureStdout(t, func() error { return renderChannelTable([]byte(`[]`)) })
	if !strings.Contains(out, "no channels") {
		t.Errorf("empty table -> %q", out)
	}
}

func TestReadChannelPayload_Shapes(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		v, err := readChannelPayload("")
		if err != nil {
			t.Fatalf("empty: %v", err)
		}
		if m, ok := v.(map[string]any); !ok || len(m) != 0 {
			t.Errorf("empty -> %#v, want empty map", v)
		}
	})
	t.Run("literal", func(t *testing.T) {
		v, err := readChannelPayload(`{"hello":"world"}`)
		if err != nil {
			t.Fatalf("literal: %v", err)
		}
		raw, _ := json.Marshal(v)
		if string(raw) != `{"hello":"world"}` {
			t.Errorf("literal -> %s", raw)
		}
	})
	t.Run("invalid_json", func(t *testing.T) {
		if _, err := readChannelPayload("not json"); err == nil {
			t.Error("invalid JSON: want error")
		}
	})
	t.Run("file", func(t *testing.T) {
		tmp := filepath.Join(t.TempDir(), "payload.json")
		if err := os.WriteFile(tmp, []byte(`{"k":"v"}`), 0o600); err != nil {
			t.Fatal(err)
		}
		v, err := readChannelPayload("@" + tmp)
		if err != nil {
			t.Fatalf("file: %v", err)
		}
		if m, ok := v.(map[string]any); !ok || m["k"] != "v" {
			t.Errorf("file -> %#v", v)
		}
	})
}

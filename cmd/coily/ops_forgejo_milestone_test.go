package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNormalizeForgejoMilestoneDueOn(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"2026-05-27", "2026-05-27T00:00:00Z", false},
		{"2026-12-31", "2026-12-31T00:00:00Z", false},
		{"", "", true},
		{"05/27/2026", "", true},
		{"2026-13-01", "", true},
		{"not-a-date", "", true},
	}
	for _, c := range cases {
		got, err := normalizeForgejoMilestoneDueOn(c.in, "prefix")
		gotErr := err != nil
		if gotErr != c.wantErr {
			t.Errorf("normalizeForgejoMilestoneDueOn(%q) err=%v, want err=%v", c.in, err, c.wantErr)
			continue
		}
		if !c.wantErr && got != c.want {
			t.Errorf("normalizeForgejoMilestoneDueOn(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestValidateForgejoMilestoneNumber(t *testing.T) {
	if _, err := validateForgejoMilestoneNumber(1, "p"); err != nil {
		t.Errorf("valid id 1 rejected: %v", err)
	}
	if _, err := validateForgejoMilestoneNumber(0, "p"); err == nil {
		t.Errorf("invalid id 0 accepted")
	}
	if _, err := validateForgejoMilestoneNumber(-1, "p"); err == nil {
		t.Errorf("invalid id -1 accepted")
	}
}

func TestForgejoMilestoneCreateBodyShape(t *testing.T) {
	got, err := json.Marshal(forgejoMilestoneCreateBody{Title: "W1"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"title":"W1"}`
	if string(got) != want {
		t.Errorf("payload = %s, want %s", got, want)
	}

	got2, err := json.Marshal(forgejoMilestoneCreateBody{
		Title:       "W1",
		Description: "first workstream",
		DueOn:       "2026-05-27T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, want := range []string{
		`"title":"W1"`,
		`"description":"first workstream"`,
		`"due_on":"2026-05-27T00:00:00Z"`,
	} {
		if !strings.Contains(string(got2), want) {
			t.Errorf("payload missing %q: %s", want, got2)
		}
	}
}

func TestForgejoMilestoneEditBodyOmitsEmpty(t *testing.T) {
	got, err := json.Marshal(forgejoMilestoneEditBody{})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(got) != `{}` {
		t.Errorf("empty patch should marshal to {}, got %s", got)
	}

	title := "W2"
	state := "closed"
	got2, err := json.Marshal(forgejoMilestoneEditBody{Title: &title, State: &state})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(got2)
	if !strings.Contains(s, `"title":"W2"`) || !strings.Contains(s, `"state":"closed"`) {
		t.Errorf("partial patch missing fields: %s", s)
	}
	if strings.Contains(s, `"description"`) || strings.Contains(s, `"due_on"`) {
		t.Errorf("partial patch leaked unset fields: %s", s)
	}
}

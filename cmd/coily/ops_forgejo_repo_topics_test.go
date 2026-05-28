package main

import (
	"encoding/json"
	"testing"
)

func TestValidateForgejoTopic(t *testing.T) {
	cases := []struct {
		in      string
		wantErr bool
	}{
		{"website", false},
		{"personal-site", false},
		{"platform-engineering-blog", false},
		{"a", false},
		{"a1", false},
		{"", true},
		{"-leading-hyphen", true},
		{"Upper", true},
		{"has space", true},
		{"has.dot", true},
		{"coilysiren.me", true},
		{"thisisaverylongtopicnamethatexceedsthirtyfive", true},
	}
	for _, c := range cases {
		err := validateForgejoTopic(c.in)
		if (err != nil) != c.wantErr {
			t.Errorf("validateForgejoTopic(%q) err=%v, wantErr=%v", c.in, err, c.wantErr)
		}
	}
}

func TestForgejoRepoTopicsBodyShape(t *testing.T) {
	got, err := json.Marshal(forgejoRepoTopicsBody{Topics: []string{"website", "personal-site"}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"topics":["website","personal-site"]}`
	if string(got) != want {
		t.Errorf("payload = %s, want %s", got, want)
	}

	// An empty replace set must marshal to an explicit [], not null, so forgejo
	// reads it as "clear all topics".
	gotEmpty, err := json.Marshal(forgejoRepoTopicsBody{Topics: []string{}})
	if err != nil {
		t.Fatalf("marshal empty: %v", err)
	}
	if string(gotEmpty) != `{"topics":[]}` {
		t.Errorf("empty payload = %s, want {\"topics\":[]}", gotEmpty)
	}
}

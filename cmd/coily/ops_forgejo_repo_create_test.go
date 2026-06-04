package main

import (
	"encoding/json"
	"testing"
)

func TestForgejoRepoCreateBodyShape(t *testing.T) {
	cases := []struct {
		name string
		body forgejoRepoCreateBody
		want string
	}{
		{
			name: "private default, name only",
			body: forgejoRepoCreateBodyFrom("website", "", "", false, false),
			want: `{"name":"website","private":true}`,
		},
		{
			name: "public when explicit",
			body: forgejoRepoCreateBodyFrom("website", "", "", true, false),
			want: `{"name":"website","private":false}`,
		},
		{
			name: "all fields",
			body: forgejoRepoCreateBodyFrom("website", "  the site  ", "main", false, true),
			want: `{"name":"website","private":true,"description":"the site","default_branch":"main","auto_init":true}`,
		},
		{
			name: "blank optionals are omitted",
			body: forgejoRepoCreateBodyFrom("website", "   ", "  ", true, false),
			want: `{"name":"website","private":false}`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := json.Marshal(c.body)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(got) != c.want {
				t.Errorf("payload = %s, want %s", got, c.want)
			}
		})
	}
}

func TestForgejoRepoCreatePath(t *testing.T) {
	cases := []struct {
		owner string
		login string
		want  string
	}{
		{"coilyco-bridge", "kai", "/api/v1/orgs/coilyco-bridge/repos"},
		{"kai", "kai", "/api/v1/user/repos"},
		{"KAI", "kai", "/api/v1/user/repos"},
		{"coilysiren", "kai", "/api/v1/orgs/coilysiren/repos"},
		{"kai", "", "/api/v1/orgs/kai/repos"},
	}
	for _, c := range cases {
		if got := forgejoRepoCreatePath(c.owner, c.login); got != c.want {
			t.Errorf("forgejoRepoCreatePath(%q, %q) = %q, want %q", c.owner, c.login, got, c.want)
		}
	}
}

func TestValidateForgejoCreateOwnerAndName(t *testing.T) {
	const prefix = "ops forgejo repo create"
	ownerCases := []struct {
		in      string
		wantErr bool
	}{
		{"coilyco-bridge", false},
		{"kai", false},
		{"", true},
		{"-leading", true},
		{"has space", true},
		{"has/slash", true},
	}
	for _, c := range ownerCases {
		err := validateForgejoCreateOwner(prefix, c.in)
		if (err != nil) != c.wantErr {
			t.Errorf("validateForgejoCreateOwner(%q) err=%v, wantErr=%v", c.in, err, c.wantErr)
		}
	}
	nameCases := []struct {
		in      string
		wantErr bool
	}{
		{"website", false},
		{"my.repo_name-1", false},
		{"", true},
		{"-leading", true},
		{"has space", true},
		{"has/slash", true},
	}
	for _, c := range nameCases {
		err := validateForgejoCreateName(prefix, c.in)
		if (err != nil) != c.wantErr {
			t.Errorf("validateForgejoCreateName(%q) err=%v, wantErr=%v", c.in, err, c.wantErr)
		}
	}
}

package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestPrintFindingWalkthrough_LoadBearingStrings pins the load-bearing pieces
// of the agent walkthrough: the file path, the frontmatter template, the
// section headers, and the verb-to-area suggestion line. The exact prose
// can drift; these are the parts an agent following the walkthrough must
// see verbatim or it will produce a malformed finding.
func TestPrintFindingWalkthrough_LoadBearingStrings(t *testing.T) {
	var buf bytes.Buffer
	err := printFindingWalkthrough(&buf, findingArgs{
		AuditPath: "/tmp/audit.jsonl",
		SkillsDir: "/tmp/skills",
		Today:     "2026-05-05",
		ID:        "id-xyz",
		Verb:      "aws.s3.ls",
		Slug:      "s3-passes-argv-gate-free",
	})
	if err != nil {
		t.Fatalf("printFindingWalkthrough: %v", err)
	}
	out := buf.String()

	wantSubstrings := []string{
		"/tmp/audit.jsonl",
		"/tmp/skills/coily-ops-aws-meta/findings/2026-05-05-s3-passes-argv-gate-free.md",
		"Suggested area for verb \"aws.s3.ls\": coily-ops-aws-meta",
		"date: 2026-05-05",
		"slug: s3-passes-argv-gate-free",
		"## What was observed",
		"## Why it slipped",
		"## Rule it produced",
		"`coily-<area>-meta`",
		"jq 'select(.id == \"id-xyz\")'",
		"coily-meta-improvement/SKILL.md",
	}
	for _, s := range wantSubstrings {
		if !strings.Contains(out, s) {
			t.Errorf("walkthrough missing required substring: %q", s)
		}
	}
}

// TestVerbToSkillArea_PrefixMapping covers the dotted verb -> meta skill
// mapping. The audit log records dotted verbs, the skill directories use
// kebab; this is the seam.
func TestVerbToSkillArea_PrefixMapping(t *testing.T) {
	cases := map[string]string{
		"":                          "coily-<area>-meta",
		"aws":                       "coily-ops-aws-meta",
		"aws.route53.change":        "coily-ops-aws-meta",
		"gh":                        "coily-ops-gh-meta",
		"gh.issue.create":           "coily-ops-gh-meta",
		"kubectl":                   "coily-ops-kubectl-meta",
		"docker":                    "coily-docker-meta",
		"eco.status":                "coily-gaming-eco-meta",
		"sirens.discord.ops":        "coily-sirens-discord-ops-meta",
		"audit.path":                "coily-audit-meta",
		"some-unknown-verb":         "coily-shared-meta",
		"completely.unknown.string": "coily-shared-meta",
	}
	for verb, want := range cases {
		if got := verbToSkillArea(verb); got != want {
			t.Errorf("verbToSkillArea(%q) = %q, want %q", verb, got, want)
		}
	}
}

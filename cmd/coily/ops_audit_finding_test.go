package main

import (
	"bytes"
	"os"
	"path/filepath"
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

// TestVerbAreaMap_TargetsExistOnDisk catches drift between the static
// verb -> meta-skill table and the actual `coily-<area>-meta` directories
// shipped in the repo. The walkthrough is only useful if every area name
// it suggests is a real path the agent can write into; without this test
// a renamed or removed skill would silently send agents to a missing
// directory.
//
// Skips when run outside a coily checkout (e.g. an installed-binary
// smoke test) - detected by the absence of `.claude/skills/`. In CI the
// directory is always present, so the assertion fires there.
func TestVerbAreaMap_TargetsExistOnDisk(t *testing.T) {
	skillsDir := findRepoSkillsDir(t)
	if skillsDir == "" {
		t.Skip(".claude/skills/ not found above test cwd; running outside a checkout")
	}

	// Every value in the lookup map must be a real directory.
	for prefix, area := range verbAreaMap {
		if info, err := os.Stat(filepath.Join(skillsDir, area)); err != nil || !info.IsDir() {
			t.Errorf("verbAreaMap[%q] = %q but %s is not a directory: %v",
				prefix, area, filepath.Join(skillsDir, area), err)
		}
	}

	// The agent-visible table renders the same data plus the two
	// fallbacks (coily-shared-meta, coily-security-boundary-discipline).
	// Parse the right-hand side of each `prefix -> area` line and check
	// the same property. Catches drift between map and table too.
	for _, line := range verbAreaTable() {
		i := strings.Index(line, "->")
		if i < 0 {
			t.Errorf("verbAreaTable entry missing arrow: %q", line)
			continue
		}
		// Right-hand side may have trailing parenthetical commentary
		// ("coily-shared-meta (cross-cutting)"). Take the first token.
		rhs := strings.TrimSpace(line[i+2:])
		area := strings.Fields(rhs)[0]
		if info, err := os.Stat(filepath.Join(skillsDir, area)); err != nil || !info.IsDir() {
			t.Errorf("verbAreaTable points at %q but %s is not a directory: %v",
				area, filepath.Join(skillsDir, area), err)
		}
	}
}

// findRepoSkillsDir walks up from the test's cwd looking for a
// `.claude/skills/` directory. Returns "" if none is found before the
// filesystem root. Mirrors detectSkillsDir's runtime logic but without
// the home-dir fallback - a unit test should only validate the in-tree
// skills layout, not whatever the host happens to have installed.
func findRepoSkillsDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		candidate := filepath.Join(dir, ".claude", "skills")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

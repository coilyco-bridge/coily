package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPrintFindingWalkthrough_LoadBearingStrings pins the load-bearing pieces
// of the agent walkthrough: the gh API invocation, the section headers, the
// area-slug suggestion line, the issue-template scaffolding, and the
// references block. The exact prose can drift; these are the parts an agent
// following the walkthrough must see verbatim or it will file a malformed
// finding (or none at all).
//
// As of coilysiren/coily#215, findings are GH issues, not files. The
// strings below reflect that contract.
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
		"Suggested area for verb \"aws.s3.ls\": ops-aws",
		"finding (ops-aws):",
		"## What was observed",
		"## Why it slipped",
		"## Rule it produced",
		"`finding`",
		"coily ops gh api -X POST repos/coilysiren/coily/issues",
		"\"labels\": [\"finding\"]",
		"jq 'select(.id == \"id-xyz\")'",
		filepath.Join("/tmp/skills", "coily-meta", "SKILL.md"),
		"https://github.com/coilysiren/coily/issues?q=label%3Afinding",
	}
	for _, s := range wantSubstrings {
		if !strings.Contains(out, s) {
			t.Errorf("walkthrough missing required substring: %q", s)
		}
	}

	// Negative: the old per-area file-path shape must NOT appear. Catches
	// regressions back to the on-disk findings model.
	dontWantSubstrings := []string{
		"coily-<area>-meta",
		"coily-ops-aws-meta/findings/",
		"coily-meta-improvement/SKILL.md",
		"date: 2026-05-05",
	}
	for _, s := range dontWantSubstrings {
		if strings.Contains(out, s) {
			t.Errorf("walkthrough still contains pre-#215 substring: %q", s)
		}
	}
}

// TestVerbToSkillArea_PrefixMapping covers the dotted verb -> area-slug
// mapping. The audit log records dotted verbs, the title prefix uses the
// short area slug.
func TestVerbToSkillArea_PrefixMapping(t *testing.T) {
	cases := map[string]string{
		"":                          "<area>",
		"aws":                       "ops-aws",
		"aws.route53.change":        "ops-aws",
		"gh":                        "ops-gh",
		"gh.issue.create":           "ops-gh",
		"kubectl":                   "ops-kubectl",
		"docker":                    "docker",
		"eco.status":                "gaming-eco",
		"audit.path":                "audit",
		"systemctl.restart":         "systemctl",
		"some-unknown-verb":         "shared",
		"completely.unknown.string": "shared",
	}
	for verb, want := range cases {
		if got := verbToSkillArea(verb); got != want {
			t.Errorf("verbToSkillArea(%q) = %q, want %q", verb, got, want)
		}
	}
}

// TestVerbAreaTable_RendersEveryMappedSlug catches drift between the
// internal verbAreaMap and the agent-facing verbAreaTable. Both surface
// the same data; if a verb prefix lands in one but not the other, the
// agent walkthrough is out of sync with the title prefix that gets
// suggested at runtime.
func TestVerbAreaTable_RendersEveryMappedSlug(t *testing.T) {
	tableSlugs := map[string]bool{}
	for _, line := range verbAreaTable() {
		i := strings.Index(line, "->")
		if i < 0 {
			t.Errorf("verbAreaTable entry missing arrow: %q", line)
			continue
		}
		rhs := strings.TrimSpace(line[i+2:])
		slug := strings.Fields(rhs)[0]
		tableSlugs[slug] = true
	}
	for prefix, area := range verbAreaMap {
		if !tableSlugs[area] {
			t.Errorf("verbAreaMap[%q] = %q but verbAreaTable does not render slug %q",
				prefix, area, area)
		}
	}
	// Fallback slugs from the negative branches of verbToSkillArea
	// belong in the table too, so the walkthrough can show them.
	for _, fallback := range []string{"shared", "security-boundary"} {
		if !tableSlugs[fallback] {
			t.Errorf("verbAreaTable does not render fallback slug %q", fallback)
		}
	}
}

// TestResolveInvokeCWD_PrefersExplicitOverOldpwdOverGetwd pins the
// resolution order from coilysiren/coily#109 so an agent can stop the
// audit-finding walkthrough from binding to the post-cd subprocess
// cwd by exporting COILY_INVOKE_CWD or relying on the bash-cd $OLDPWD.
func TestResolveInvokeCWD_PrefersExplicitOverOldpwdOverGetwd(t *testing.T) {
	parent := t.TempDir()
	other := t.TempDir()

	t.Setenv("COILY_INVOKE_CWD", parent)
	t.Setenv("OLDPWD", other)
	if got := resolveInvokeCWD(); got != parent {
		t.Errorf("with COILY_INVOKE_CWD set, got %q, want %q", got, parent)
	}

	t.Setenv("COILY_INVOKE_CWD", "")
	t.Setenv("OLDPWD", other)
	if got := resolveInvokeCWD(); got != other {
		t.Errorf("with only OLDPWD set, got %q, want %q", got, other)
	}

	t.Setenv("COILY_INVOKE_CWD", "/nonexistent/path/should/skip")
	t.Setenv("OLDPWD", other)
	if got := resolveInvokeCWD(); got != other {
		t.Errorf("stale COILY_INVOKE_CWD should be skipped; got %q, want %q", got, other)
	}
}

// TestPrintFindingWalkthrough_BannerOnInvokeCwdDrift covers the drift
// surface from #109: when invoke-cwd disagrees with the subprocess
// cwd, the walkthrough prints a banner naming both so the agent can't
// silently bind the audit-log read to the wrong host context.
func TestPrintFindingWalkthrough_BannerOnInvokeCwdDrift(t *testing.T) {
	parent := t.TempDir()
	t.Setenv("COILY_INVOKE_CWD", parent)
	t.Setenv("OLDPWD", "")

	var buf bytes.Buffer
	err := printFindingWalkthrough(&buf, findingArgs{
		AuditPath: "/tmp/audit.jsonl",
		SkillsDir: "/tmp/skills",
		Today:     "2026-05-14",
		Verb:      "aws.s3.ls",
	})
	if err != nil {
		t.Fatalf("printFindingWalkthrough: %v", err)
	}
	out := buf.String()

	subproc, _ := os.Getwd()
	if subproc == parent {
		t.Skip("test binary cwd happens to equal the temp dir; banner suppression is correct")
	}
	for _, want := range []string{
		"[invoke-cwd]",
		parent,
		"COILY_INVOKE_CWD",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing banner substring %q in:\n%s", want, out)
		}
	}
}

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// auditFindingCommand emits a self-contained walkthrough for an agent that
// has no prior coily context. The motivating shape: a human flags an audit
// row as a violation and tells the agent "go write a finding." The agent
// has not read coily-meta-improvement, does not know what a finding is,
// does not know where they live, does not know the schema. This command
// closes that gap in a single invocation: copy-pasteable shell and markdown
// the agent can act on without further reading.
//
// The walkthrough is intentionally redundant with coily-meta-improvement's
// SKILL.md - that skill is the source of truth for the loop, this verb
// is the on-ramp for an agent that does not have the skill loaded. When
// the loop's vocabulary or steps change, both must update; the verb pins
// the agent-facing copy of the contract.
func (r *Runner) auditFindingCommand() *cli.Command {
	return &cli.Command{
		Name:  "finding",
		Usage: "Walk an agent through writing a finding about a flagged audit event.",
		Description: `finding emits a self-contained walkthrough that takes an agent with no
prior coily context from a single flagged audit row to a written finding
file. A "finding" is the write-once observation file that feeds coily's
meta-improvement loop (see coily-meta-improvement/SKILL.md).

Use this verb when a human has flagged a row in the audit log as a
violation - a refusal that should not have been needed, an accept that
should have been a refusal, or a near-miss the gate caught but the
threat model did not anticipate. The agent runs:

    coily audit finding --id <row-id> --slug <short-kebab-slug>

reads the printed walkthrough, and produces the finding file. The flags
are optional; without them the walkthrough still works but uses
placeholders the agent must fill in.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "id",
				Usage: "audit row id (UUID v7 from `coily audit tail`) the finding is about",
			},
			&cli.StringFlag{
				Name:  "ts",
				Usage: "unix-seconds anchor for the audit row, alternative to --id",
			},
			&cli.StringFlag{
				Name:  "verb",
				Usage: "verb name from the audit row (e.g. `ops.aws.s3.ls`); used to suggest a skill area",
			},
			&cli.StringFlag{
				Name:  "slug",
				Usage: "short kebab-case slug for the finding filename and frontmatter",
			},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name:      "audit.finding",
				SkipScope: true,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--id":   c.String("id"),
						"--ts":   c.String("ts"),
						"--verb": c.String("verb"),
						"--slug": c.String("slug"),
					}, nil
				},
				Action: func(_ context.Context, c *cli.Command) error {
					return printFindingWalkthrough(os.Stdout, findingArgs{
						AuditPath: r.Cfg.Audit.LogPath,
						SkillsDir: detectSkillsDir(),
						Today:     time.Now().UTC().Format("2006-01-02"),
						ID:        c.String("id"),
						TS:        c.String("ts"),
						Verb:      c.String("verb"),
						Slug:      c.String("slug"),
					})
				},
			},
			r.Audit,
		),
	}
}

type findingArgs struct {
	AuditPath string
	SkillsDir string
	Today     string
	ID        string
	TS        string
	Verb      string
	Slug      string
}

// fln/fpf wrap fmt.Fprintln/Fprintf so the walkthrough generators can
// emit text without each call site having to discard the unused error
// return. errcheck stays satisfied by the explicit `_, _ =` here.
func fln(w io.Writer, args ...any) { _, _ = fmt.Fprintln(w, args...) }
func fpf(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

func printFindingWalkthrough(w io.Writer, a findingArgs) error {
	slug := a.Slug
	if slug == "" {
		slug = "<short-kebab-slug>"
	}
	area := verbToSkillArea(a.Verb)
	findingFile := filepath.Join(a.SkillsDir, area, "findings", a.Today+"-"+slug+".md")

	wtIntro(w)
	wtBackground(w)
	wtStep1Locate(w, a)
	wtStep2PickArea(w, a, area)
	wtStep3Write(w, a, slug, findingFile)
	wtStep4Forward(w)
	wtStep5Stop(w)
	wtReferences(w, a)
	return nil
}

func wtIntro(w io.Writer) {
	fln(w, "coily audit finding - walkthrough")
	fln(w, "=================================")
	fln(w)
	fln(w, "You (the agent) have been told a row in the audit log is a violation.")
	fln(w, "Your job is to write a finding file that records the observation. You")
	fln(w, "are NOT promoting it to a rule, NOT editing any SKILL.md, NOT changing")
	fln(w, "code. One file, then stop. Steps below.")
	fln(w)
}

func wtBackground(w io.Writer) {
	fln(w, "Background")
	fln(w, "----------")
	fln(w, "coily is the operator CLI for Kai's homelab. Every invocation lands one")
	fln(w, "JSONL row in the audit log. A `finding` is a write-once markdown file")
	fln(w, "that records something the audit log surfaced - a denial that should")
	fln(w, "not have been needed, an accept that should have been a denial, or a")
	fln(w, "near-miss the gate caught but the threat model missed. Findings live")
	fln(w, "next to the relevant `coily-<area>-meta` skill in a `findings/`")
	fln(w, "subdirectory. They feed the meta-improvement loop; they are not the")
	fln(w, "loop. See coily-meta-improvement/SKILL.md if you want the full picture.")
	fln(w)
}

func wtStep1Locate(w io.Writer, a findingArgs) {
	fln(w, "Step 1 - Locate the audit row")
	fln(w, "-----------------------------")
	fln(w, "Audit log path:", a.AuditPath)
	fln(w)
	switch {
	case a.ID != "":
		fpf(w, "    coily audit tail | jq 'select(.id == \"%s\")'\n", a.ID)
	case a.TS != "":
		fpf(w, "    coily audit tail --since=%s | jq 'select(.ts == %s)'\n", a.TS, a.TS)
	default:
		fln(w, "    coily audit tail --since=24h | jq 'select(.decision == \"reject\")'")
		fln(w, "    # or: coily audit tail | jq 'select(.id == \"<id>\")' if you have an id")
	}
	fln(w)
	fln(w, "Capture the row's: id, ts, verb, argv, decision, exit_code, error.")
	fln(w, "These are the concrete facts the finding cites. Do not paraphrase them.")
	fln(w)
}

func wtStep2PickArea(w io.Writer, a findingArgs, area string) {
	fln(w, "Step 2 - Pick the skill area")
	fln(w, "----------------------------")
	fln(w, "The verb in the audit row maps to one `coily-<area>-meta` skill. Use")
	fln(w, "the prefix table below; if the row spans multiple verbs (a cross-")
	fln(w, "cutting concern), use coily-shared-meta. If the row is about the")
	fln(w, "security boundary itself - the gate, the audit row format, the")
	fln(w, "scope binding - use coily-security-boundary-discipline.")
	fln(w)
	for _, line := range verbAreaTable() {
		fln(w, "    "+line)
	}
	fln(w)
	if a.Verb != "" {
		fpf(w, "Suggested area for verb %q: %s\n", a.Verb, area)
	} else {
		fln(w, "Pass --verb <verb> on a future invocation to get a resolved area name.")
	}
	if dirs := discoverMetaSkills(a.SkillsDir); len(dirs) > 0 {
		fln(w)
		fln(w, "Areas that already have findings/ siblings on this host:")
		for _, d := range dirs {
			fln(w, "    "+d)
		}
	}
	fln(w)
}

func wtStep3Write(w io.Writer, a findingArgs, slug, findingFile string) {
	fln(w, "Step 3 - Write the file")
	fln(w, "-----------------------")
	fln(w, "Path:")
	fln(w, "    "+findingFile)
	fln(w)
	fln(w, "Date is today (UTC). Slug is a short kebab-case noun phrase that names")
	fln(w, "the shape of the violation, not the verb. Examples from existing")
	fln(w, "findings: read-only-audit-without-gate, claude-bypasses-coily-gh-wrapper,")
	fln(w, "scope-required-from-non-git-cwd. Bad slugs: aws-bug, gh-issue, found-it.")
	fln(w)
	fln(w, "Template (write this file verbatim; replace bracketed placeholders):")
	fln(w)
	fln(w, findingTemplate(a.Today, slug))
	fln(w)
	fln(w, "Section discipline:")
	fln(w, "  - What was observed: concrete and scoped to ONE shape. Cite the audit")
	fln(w, "    row's ts/id/verb/argv. \"coily ops aws s3 ls against bucket X passed")
	fln(w, "    argv-gate-free on YYYY-MM-DD\" - not \"coily ops aws is broken.\"")
	fln(w, "  - Why it slipped: what gap in the gate, the audit, the docs, or the")
	fln(w, "    threat model let this through. If you don't know, say so honestly")
	fln(w, "    and stop - the human can fill in the gap before the file is finished.")
	fln(w, "  - Rule it produced: the rule or anti-signal as a one-line claim. May")
	fln(w, "    be empty if this finding is data, not a rule. Do not invent rules to")
	fln(w, "    fill the section.")
	fln(w)
}

func wtStep4Forward(w io.Writer) {
	fln(w, "Step 4 - File the forward action (conditional)")
	fln(w, "----------------------------------------------")
	fln(w, "If the finding implies a code change, a doc change, or a sequencing")
	fln(w, "rule addition, file it as a GitHub issue on coilysiren/coily and put")
	fln(w, "the URL in the frontmatter's `promoted_to.issue` slot. The issue is")
	fln(w, "the source of truth for what happens next - not the finding, not the")
	fln(w, "SKILL.md. A finding without a forward action is still valid: it can")
	fln(w, "be evidence for an existing rule or a documented near-miss.")
	fln(w)
	fln(w, "    coily ops gh issue create --repo coilysiren/coily \\")
	fln(w, "        --title '<one-line summary>' --body-file <path-to-issue-body.md>")
	fln(w)
}

func wtStep5Stop(w io.Writer) {
	fln(w, "Step 5 - Stop")
	fln(w, "-------------")
	fln(w, "Findings are write-once. Do not edit after creation. Do not promote")
	fln(w, "the finding to an anti-signal or sequencing rule in any SKILL.md - that")
	fln(w, "step happens in a separate review, by a human or a later agent, with")
	fln(w, "the finding as the evidence pin. Your boundary ends at Step 4.")
	fln(w)
}

func wtReferences(w io.Writer, a findingArgs) {
	fln(w, "References")
	fln(w, "----------")
	fln(w, "    audit log:        ", a.AuditPath)
	fln(w, "    skills directory: ", a.SkillsDir)
	fln(w, "    loop docs:        ", filepath.Join(a.SkillsDir, "coily-meta-improvement", "SKILL.md"))
	fln(w, "    authoring docs:   ", filepath.Join(a.SkillsDir, "coily-skill-authoring", "SKILL.md"))
}

func findingTemplate(date, slug string) string {
	return strings.Join([]string{
		"---",
		"date: " + date,
		"slug: " + slug,
		"promoted_to:",
		"  # populated later when a rule lands or an issue is filed:",
		"  # - anti-signal: \"<entry text>\"",
		"  # - sequencing-rule: \"<entry text>\"",
		"  # - issue: <url>",
		"---",
		"",
		"# " + date + " - <one-line summary of what was observed>",
		"",
		"## What was observed",
		"",
		"<concrete, scoped to one shape; cite the audit row's ts, id, verb, argv>",
		"",
		"## Why it slipped",
		"",
		"<what gap in the gate, the audit, the docs, or the threat model let this through>",
		"",
		"## Rule it produced",
		"",
		"<one-line claim; may be empty if this finding is data, not a rule>",
	}, "\n")
}

// verbAreaMap is the verb-prefix -> meta-skill lookup. Map literal rather
// than a switch so adding a verb prefix does not push gocyclo over its
// per-function limit, and so the table stays in one place next to
// verbAreaTable (which renders the same data for the agent).
var verbAreaMap = map[string]string{
	"aws":         "coily-ops-aws-meta",
	"gh":          "coily-ops-gh-meta",
	"kubectl":     "coily-ops-kubectl-meta",
	"docker":      "coily-docker-meta",
	"tailscale":   "coily-tailscale-meta",
	"modio":       "coily-modio-meta",
	"trello":      "coily-trello-meta",
	"git":         "coily-git-meta",
	"ssh":         "coily-ssh-meta",
	"lockdown":    "coily-lockdown-meta",
	"audit":       "coily-audit-meta",
	"pkg":         "coily-pkg-meta",
	"repo":        "coily-repo-meta",
	"eco":         "coily-gaming-eco-meta",
	"factorio":    "coily-gaming-factorio-meta",
	"icarus":      "coily-gaming-icarus-meta",
	"core_keeper": "coily-gaming-core-keeper-meta",
	"corekeeper":  "coily-gaming-core-keeper-meta",
}

// verbToSkillArea maps an audit-row verb to the `coily-<area>-meta` skill
// directory most likely to host the finding. Best-effort prefix match;
// when the verb does not map cleanly the caller is told to use
// coily-shared-meta or coily-security-boundary-discipline. The mapping is
// intentionally inlined rather than auto-derived from the cli tree because
// the verb name in the audit log is dotted (e.g. `aws.route53.change-...`)
// while the cli tree is hierarchical, and the agent reading this output
// needs a flat lookup.
func verbToSkillArea(v string) string {
	if v == "" {
		return "coily-<area>-meta"
	}
	head := v
	if i := strings.IndexAny(v, ".-"); i > 0 {
		head = v[:i]
	}
	if area, ok := verbAreaMap[head]; ok {
		return area
	}
	return "coily-shared-meta"
}

func verbAreaTable() []string {
	return []string{
		"aws.*       -> coily-ops-aws-meta",
		"gh.*        -> coily-ops-gh-meta",
		"kubectl.*   -> coily-ops-kubectl-meta",
		"docker.*    -> coily-docker-meta",
		"tailscale.* -> coily-tailscale-meta",
		"ssh.*       -> coily-ssh-meta",
		"git.*       -> coily-git-meta",
		"audit.*     -> coily-audit-meta",
		"lockdown.*  -> coily-lockdown-meta",
		"modio.*     -> coily-modio-meta",
		"trello.*    -> coily-trello-meta",
		"pkg.*       -> coily-pkg-meta",
		"repo.*      -> coily-repo-meta",
		"eco.*       -> coily-gaming-eco-meta",
		"factorio.*  -> coily-gaming-factorio-meta",
		"icarus.*    -> coily-gaming-icarus-meta",
		"<other>     -> coily-shared-meta (cross-cutting)",
		"<security>  -> coily-security-boundary-discipline (gate/audit/scope itself)",
	}
}

// detectSkillsDir picks the most plausible skills directory for the
// host running this command. Walks up from cwd looking for a `.claude/
// skills/` directory; falls back to `~/.claude/skills/`. The walkthrough
// uses this to print absolute paths the agent can act on.
func detectSkillsDir() string {
	if cwd, err := os.Getwd(); err == nil {
		dir := cwd
		for {
			candidate := filepath.Join(dir, ".claude", "skills")
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				return candidate
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".claude", "skills")
	}
	return ".claude/skills"
}

// discoverMetaSkills lists `coily-*-meta` directories under skillsDir that
// already have a `findings/` sibling. Empty list when the directory does
// not exist or when no meta skills have findings yet; the walkthrough
// degrades gracefully in that case.
func discoverMetaSkills(skillsDir string) []string {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "coily-") {
			continue
		}
		if !strings.HasSuffix(name, "-meta") && name != "coily-security-boundary-discipline" {
			continue
		}
		findings := filepath.Join(skillsDir, name, "findings")
		if info, err := os.Stat(findings); err == nil && info.IsDir() {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

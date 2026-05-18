package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// auditFindingCommand emits a self-contained walkthrough for an agent that
// has no prior coily context. The motivating shape: a human flags an audit
// row as a violation and tells the agent "go write a finding." The agent
// has not read coily-meta, does not know what a finding is, does not know
// the schema. This command closes that gap in a single invocation:
// copy-pasteable shell and markdown the agent can act on without further
// reading.
//
// As of coilysiren/coily#215, findings are GitHub issues on coilysiren/coily
// with label `finding`, not files on disk. The walkthrough below files an
// issue, never writes a markdown file. The pre-#215 on-disk finding shape
// is preserved in git history; the migration mapping lives in
// coily-meta/findings-migration.md.
//
// The walkthrough is intentionally redundant with coily-meta's SKILL.md -
// that skill is the source of truth for the loop; this verb is the on-ramp
// for an agent that does not have the skill loaded. When the loop's
// vocabulary or steps change, both must update; the verb pins the
// agent-facing copy of the contract.
func (r *Runner) auditFindingCommand() *cli.Command {
	return &cli.Command{
		Name:  "finding",
		Usage: "Walk an agent through filing a finding GitHub issue about a flagged audit event.",
		Description: `finding emits a self-contained walkthrough that takes an agent with no
prior coily context from a single flagged audit row to a filed GitHub
issue on coilysiren/coily with label ` + "`finding`" + `. A "finding" is the
write-once observation issue that feeds coily's meta-improvement loop
(see coily-meta/SKILL.md).

Use this verb when a human has flagged a row in the audit log as a
violation - a refusal that should not have been needed, an accept that
should have been a refusal, or a near-miss the gate caught but the
threat model did not anticipate. The agent runs:

    coily audit finding --id <row-id> --slug <short-kebab-slug>

reads the printed walkthrough, and files the issue. The flags are
optional; without them the walkthrough still works but uses placeholders
the agent must fill in.`,
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
				Usage: "verb name from the audit row (e.g. `ops.aws.s3.ls`); used to suggest a title prefix",
			},
			&cli.StringFlag{
				Name:  "slug",
				Usage: "short kebab-case slug for the finding title and forward-action reference",
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

	wtCWDBanner(w, a.SkillsDir)
	wtIntro(w)
	wtBackground(w)
	wtStep1Locate(w, a)
	wtStep2ClassifyArea(w, a, area)
	wtStep3FileIssue(w, a, area, slug)
	wtStep4Stop(w)
	wtReferences(w, a)
	return nil
}

// wtCWDBanner surfaces invoke-cwd drift at the top of the walkthrough.
// If the resolved skills directory does not sit under the agent's
// invoke-cwd (because they did `cd <repo> && coily ...` and the
// resolver picked up the parent via $OLDPWD / $COILY_INVOKE_CWD), say
// so explicitly. Quiet when invoke-cwd matches the subprocess cwd,
// when COILY_INVOKE_CWD/OLDPWD aren't set, or when the resolution
// can't be determined. See coilysiren/coily#109.
func wtCWDBanner(w io.Writer, skillsDir string) {
	invoke := resolveInvokeCWD()
	subproc, err := os.Getwd()
	if err != nil || invoke == "" || invoke == subproc {
		return
	}
	source := "OLDPWD"
	if os.Getenv("COILY_INVOKE_CWD") != "" {
		source = "COILY_INVOKE_CWD"
	}
	fln(w, "[invoke-cwd]")
	fln(w, "    invoke-cwd:    ", invoke, "(from $"+source+")")
	fln(w, "    subprocess-cwd:", subproc)
	fln(w, "    skills-dir:    ", skillsDir)
	fln(w, "Walkthrough paths below are bound to the invoke-cwd, not the subprocess.")
	fln(w, "If you intended the subprocess cwd, re-run from there without the cd.")
	fln(w)
}

func wtIntro(w io.Writer) {
	fln(w, "coily audit finding - walkthrough")
	fln(w, "=================================")
	fln(w)
	fln(w, "You (the agent) have been told a row in the audit log is a violation.")
	fln(w, "Your job is to file a finding GitHub issue that records the observation.")
	fln(w, "You are NOT promoting it to a rule, NOT editing any SKILL.md, NOT changing")
	fln(w, "code. One issue, then stop. Steps below.")
	fln(w)
}

func wtBackground(w io.Writer) {
	fln(w, "Background")
	fln(w, "----------")
	fln(w, "coily is the operator CLI for Kai's homelab. Every invocation lands one")
	fln(w, "JSONL row in the audit log. A `finding` is a write-once GitHub issue on")
	fln(w, "coilysiren/coily, labelled `finding`, that records something the audit")
	fln(w, "log surfaced - a denial that should not have been needed, an accept that")
	fln(w, "should have been a denial, or a near-miss the gate caught but the threat")
	fln(w, "model missed. Findings feed the meta-improvement loop documented in")
	fln(w, "coily-meta/SKILL.md. The issue is the artifact, not a file on disk.")
	fln(w)
	fln(w, "Write-once discipline: do not edit the issue body after creation. State")
	fln(w, "that changes later (a rule lands, a test pins it, an upstream patch")
	fln(w, "removes the cause) goes in comments, not the body.")
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

func wtStep2ClassifyArea(w io.Writer, a findingArgs, area string) {
	fln(w, "Step 2 - Classify the verb area")
	fln(w, "-------------------------------")
	fln(w, "The verb in the audit row maps to one area slug. The slug becomes the")
	fln(w, "title prefix on the finding issue so the per-area anti-signals in")
	fln(w, "coily-meta/SKILL.md section 9 stay scannable. Cross-cutting rows use")
	fln(w, "`shared`. Rows about the security boundary itself (the gate, the audit")
	fln(w, "row format, scope binding) use `security-boundary`.")
	fln(w)
	for _, line := range verbAreaTable() {
		fln(w, "    "+line)
	}
	fln(w)
	if a.Verb != "" {
		fpf(w, "Suggested area for verb %q: %s\n", a.Verb, area)
	} else {
		fln(w, "Pass --verb <verb> on a future invocation to get a resolved area slug.")
	}
	fln(w)
}

func wtStep3FileIssue(w io.Writer, a findingArgs, area, slug string) {
	fln(w, "Step 3 - File the issue")
	fln(w, "-----------------------")
	fln(w, "Title:")
	fpf(w, "    finding (%s): <one-line summary of what was observed>\n", area)
	fln(w)
	fln(w, "Slug is a short kebab-case noun phrase that names the shape of the")
	fln(w, "violation, not the verb. Examples: read-only-audit-without-gate,")
	fln(w, "claude-bypasses-coily-gh-wrapper, scope-required-from-non-git-cwd. Bad")
	fln(w, "slugs: aws-bug, gh-issue, found-it.")
	fln(w)
	fln(w, "Body template (write the issue body verbatim; replace bracketed placeholders):")
	fln(w)
	fln(w, findingBodyTemplate(a.Today, slug))
	fln(w)
	fln(w, "Section discipline:")
	fln(w, "  - What was observed: concrete and scoped to ONE shape. Cite the audit")
	fln(w, "    row's ts/id/verb/argv. \"coily ops aws s3 ls against bucket X passed")
	fln(w, "    argv-gate-free on YYYY-MM-DD\" - not \"coily ops aws is broken.\"")
	fln(w, "  - Why it slipped: what gap in the gate, the audit, the docs, or the")
	fln(w, "    threat model let this through. If you don't know, say so honestly")
	fln(w, "    and stop - the human can fill in the gap before the issue is filed.")
	fln(w, "  - Rule it produced: the rule or anti-signal as a one-line claim. May")
	fln(w, "    be empty if this finding is data, not a rule. Do not invent rules to")
	fln(w, "    fill the section.")
	fln(w)
	fln(w, "Filing command (uses gh API to avoid coily's metachar gate on argv):")
	fln(w)
	fln(w, "    cat >/tmp/finding.json <<EOF")
	fln(w, "    {")
	fpf(w, "      \"title\": \"finding (%s): <one-line summary>\",\n", area)
	fln(w, "      \"body\": <paste-rendered-body-as-JSON-string>,")
	fln(w, "      \"labels\": [\"finding\"]")
	fln(w, "    }")
	fln(w, "    EOF")
	fln(w, "    coily ops gh api -X POST repos/coilysiren/coily/issues --input /tmp/finding.json")
	fln(w)
	fln(w, "If the finding implies a follow-up code or doc change, file a SECOND")
	fln(w, "issue (no `finding` label) for the forward action and cross-link both in")
	fln(w, "their bodies. The finding is the observation; the forward-action issue")
	fln(w, "is the work. They are separate artifacts on purpose.")
	fln(w)
}

func wtStep4Stop(w io.Writer) {
	fln(w, "Step 4 - Stop")
	fln(w, "-------------")
	fln(w, "Findings are write-once. Do not edit the issue body after creation. Do")
	fln(w, "not promote the finding to an anti-signal or sequencing rule in any")
	fln(w, "SKILL.md - that step happens in a separate review, by a human or a")
	fln(w, "later agent, with the finding issue as the evidence pin. Your boundary")
	fln(w, "ends at Step 3.")
	fln(w)
}

func wtReferences(w io.Writer, a findingArgs) {
	fln(w, "References")
	fln(w, "----------")
	fln(w, "    audit log:        ", a.AuditPath)
	fln(w, "    skills directory: ", a.SkillsDir)
	fln(w, "    loop docs:        ", filepath.Join(a.SkillsDir, "coily-meta", "SKILL.md"))
	fln(w, "    finding index:     https://github.com/coilysiren/coily/issues?q=label%3Afinding")
}

func findingBodyTemplate(date, slug string) string {
	return strings.Join([]string{
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
		"",
		"---",
		"_Filed " + date + " via `coily audit finding`. Slug: `" + slug + "`._",
	}, "\n")
}

// verbAreaMap is the verb-prefix -> area-slug lookup. The slug is used as
// the title prefix on the finding issue: `finding (<slug>): <summary>`.
// Map literal rather than a switch so adding a verb prefix does not push
// gocyclo over its per-function limit, and so the table stays in one place
// next to verbAreaTable (which renders the same data for the agent).
var verbAreaMap = map[string]string{
	"aws":         "ops-aws",
	"gh":          "ops-gh",
	"kubectl":     "ops-kubectl",
	"docker":      "docker",
	"tailscale":   "tailscale",
	"modio":       "modio",
	"trello":      "trello",
	"git":         "git",
	"ssh":         "ssh",
	"lockdown":    "lockdown",
	"audit":       "audit",
	"pkg":         "pkg",
	"repo":        "repo",
	"eco":         "gaming-eco",
	"factorio":    "gaming-factorio",
	"icarus":      "gaming-icarus",
	"core_keeper": "gaming-core-keeper",
	"corekeeper":  "gaming-core-keeper",
	"systemctl":   "systemctl",
}

// verbToSkillArea maps an audit-row verb to the area slug used as the
// finding-issue title prefix. Best-effort prefix match; when the verb
// does not map cleanly the caller is told to use `shared` (cross-cutting)
// or `security-boundary` (gate / audit / scope itself). The mapping is
// intentionally inlined rather than auto-derived from the cli tree because
// the verb name in the audit log is dotted (e.g. `aws.route53.change-...`)
// while the cli tree is hierarchical, and the agent reading this output
// needs a flat lookup.
func verbToSkillArea(v string) string {
	if v == "" {
		return "<area>"
	}
	head := v
	if i := strings.IndexAny(v, ".-"); i > 0 {
		head = v[:i]
	}
	if area, ok := verbAreaMap[head]; ok {
		return area
	}
	return "shared"
}

func verbAreaTable() []string {
	return []string{
		"aws.*       -> ops-aws",
		"gh.*        -> ops-gh",
		"kubectl.*   -> ops-kubectl",
		"docker.*    -> docker",
		"tailscale.* -> tailscale",
		"ssh.*       -> ssh",
		"git.*       -> git",
		"audit.*     -> audit",
		"lockdown.*  -> lockdown",
		"modio.*     -> modio",
		"trello.*    -> trello",
		"pkg.*       -> pkg",
		"repo.*      -> repo",
		"systemctl.* -> systemctl",
		"eco.*       -> gaming-eco",
		"factorio.*  -> gaming-factorio",
		"icarus.*    -> gaming-icarus",
		"core_keeper.* / corekeeper.* -> gaming-core-keeper",
		"<other>     -> shared (cross-cutting)",
		"<security>  -> security-boundary (gate/audit/scope itself)",
	}
}

// resolveInvokeCWD returns the cwd the operator was sitting in when they
// kicked off coily, not the cwd of the subprocess. The two diverge when
// an agent does `cd <repo> && coily ...` to enter a target repo before
// invoking the binary; coily's getcwd then reports `<repo>/`, but the
// human's intent was bound to the parent. See coilysiren/coily#109.
//
// Resolution order:
//
//  1. $COILY_INVOKE_CWD - explicit hand-off from the agent layer.
//  2. $OLDPWD - bash/zsh export this on cd, so the bash-cd case
//     recovers automatically with no agent-side cooperation.
//  3. os.Getwd() - subprocess cwd, the previous default.
//
// Any candidate that doesn't resolve to a real directory is skipped,
// so a stale env var doesn't poison the lookup.
func resolveInvokeCWD() string {
	for _, env := range []string{"COILY_INVOKE_CWD", "OLDPWD"} {
		v := strings.TrimSpace(os.Getenv(env))
		if v == "" {
			continue
		}
		// #nosec G304 -- read-only stat for cwd routing; no file open or
		// write follows. Worst case is the walkthrough prints the wrong
		// path, which the operator sees and corrects.
		if info, err := os.Stat(filepath.Clean(v)); err == nil && info.IsDir() {
			return v
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	return ""
}

// detectSkillsDir picks the most plausible skills directory for the
// host running this command. Walks up from the resolved invoke-cwd
// (see resolveInvokeCWD) looking for a `.claude/skills/` directory;
// falls back to `~/.claude/skills/`. The walkthrough uses this to
// print the absolute path of `coily-meta/SKILL.md` and to detect
// invoke-cwd drift.
func detectSkillsDir() string {
	if cwd := resolveInvokeCWD(); cwd != "" {
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

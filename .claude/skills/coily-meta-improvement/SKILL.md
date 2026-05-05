---
name: coily-meta-improvement
description: The meta-improvement loop for coily as a system. Defines how observations (audit-log sweeps, denied invocations, incident write-ups, catalogue-review insights) become findings, how findings promote to anti-signals or sequencing rules in `coily-*-meta` skills, where unbounded data lives (per-skill `findings/` siblings), and where bounded design surface lives (each skill's SKILL.md). Distinct from individual `coily-*-meta` skills which carry verb-specific or area-specific content; this skill is the loop itself. Use when running an audit-log sweep across `~/.coily/audit/*.jsonl`, when a coily verb behaved in a way that deserves a write-up, when designing a new `coily-*-meta` skill, when reviewing whether a finding should promote to a rule, or when evaluating whether the loop itself needs adjustment. Triggers - meta-improvement, meta improvement, meta loop, audit-log sweep, audit sweep, finding, findings, promote to anti-signal, promote to rule, coily skill loop, coily learning loop, coily improvement loop, coily-*-meta, write-once finding, bounded design surface, unbounded growth, hoist vs stamp.
---

# coily-meta-improvement

The loop that produces and maintains the `coily-*-meta` skills. This file describes the loop itself. The rules and findings it produces live in the per-area meta skills.

Composes with: `coily-skill-authoring` (the rule-shape and authoring conventions), `coily-shared-meta` (the inventory and shared ops rules), every `coily-*-meta` skill (the loop's outputs).

## Vocabulary

Terms used here that come from the wider system-improvement work. Naming them explicitly so cold readers can recognize the shapes.

- **Raw → rollup → consumer.** The data layering this loop follows. Findings are raw. The bounded SKILL.md catalogues are the rollup. The agent invoking a coily verb is the consumer that inherits the rolled-up trust. The same pattern shows up in MCP-health probes, daily-routine inboxes, and incident knowledge.
- **Agent-native skills repo (T4).** The maturity tier where an ops system carries enough latent knowledge that the next agent run inherits what the prior one learned. The `coily-*-meta` family is the T4 surface for coily. T1 = unscoped destruction risk. T2 = coily as audit gate (where coily lives today). T3 = read-only and scrubbed. T4 = this skill family.
- **Continuous comprehension.** The property of always carrying an accurate, current model of the system in non-human substrate. Findings preserve raw data. The bounded surface preserves the model. When they drift, comprehension has decayed and the loop exists to catch and repair it.
- **Anti-signal codification.** The highest-leverage rule output. Negatively-framed instructions ("do not waste time on X, even though it looks right") encode expensive-earned knowledge. Positive instructions are easier to write and lower-value.
- **Failure-shape as the unit of organization.** When a cross-cutting investigation skill is needed, organize by failure-shape (`eco-investigation`, not `eco-mods`). Per-verb `coily-*-meta` skills are still organized by area because the area boundary maps to a real component boundary in the cli.

## The loop

Five steps. Each one has a where-it-lives, so the loop is auditable and the data does not pool in one place.

### 1. Observe

A concrete signal arrives. One of:

- **Audit-log sweep.** A scan of `~/.coily/audit/*.jsonl` (and Claude session-history denied-Bash entries) over a window finds a pattern - a verb invoked at unexpected scope, a deny rate that suggests a missing gate, a near-miss that the gate caught but the next operator might not.
- **Incident.** A coily verb did something the operator did not intend, or did not do something the operator did intend. Real or near-miss.
- **Catalogue review.** Reading existing anti-signals or sequencing rules in any `coily-*-meta` and noticing an implicit inverse, an unstated assumption, or a silent gap.
- **Friend report.** Someone running coily on their own host hit a shape that does not match Kai's experience.

The observation is a fact about the system, not yet a rule.

### 2. Write the finding

Land a write-once file at `coily-<area>-meta/findings/YYYY-MM-DD-<slug>.md`. Schema:

```yaml
---
date: YYYY-MM-DD
slug: short-slug
promoted_to:           # optional, populated when step 4 runs
  - anti-signal: "<entry text>"
  - sequencing-rule: "<entry text>"
  - issue: <url>
---
```

Body: three sections.

- **What was observed.** Concrete, scoped to one shape. Not "coily ops aws is broken." Specific: "coily ops aws s3 ls against bucket X passed argv-gate-free on 2026-05-05."
- **Why it slipped.** What gap (in the gate, the audit, the docs, the threat model) let this through.
- **Rule it produced.** The rule or anti-signal as a one-line claim. May be empty if this finding is data, not a rule.

Findings are write-once. They are never edited after creation. If a followup observation is worth recording, write a new finding and reference the prior one. State that changes after creation (issue closed, test landed, rule retired) lives at the GitHub issue, not in the finding.

### 3. File the forward action

If the finding implies a code change, a doc change, or a sequencing rule addition, file it as a GitHub issue on the appropriate repo (usually `coilysiren/coily`). The finding's frontmatter records the issue URL. The issue is the source of truth for "what happens next" - not the finding, not the SKILL.md.

A finding without a forward action is still valid: it can be the evidence pin for an existing rule, or it can document a near-miss that did not require a fix. The forward-action step is conditional, not mandatory.

### 4. Promote to a rule

When the finding produces or grounds a rule, edit the appropriate `coily-*-meta` skill's SKILL.md:

- **Anti-signal:** add a one-line entry to section 1, with `**Pin:**` linking the finding (and the issue, if any).
- **Sequencing rule:** add a three-line entry (rule / Why / How to apply) to section 2, with the Why citing the finding.

If the rule is generic (applies to all `coily-ops-*` not just one verb), it lives in `coily-shared-meta`. Per-verb meta skills only carry verb-specific rules. See the hoist guidance in `coily-skill-authoring`.

A finding can also fail to promote: the observation was real but did not generalize. That is fine. The finding stays in the `findings/` dir as evidence. The next finding may rhyme with it and the pair may then warrant a rule.

### 5. Verify (when applicable)

If the rule is verifiable at runtime - a `TestSecurityClaim_*` test, a CI check, a generated allowlist - the verification lives in the coily codebase, not in the skill. The skill's Pin line points at the test or check. If the issue closes without a test, the rule does not promote to "validated." It stays "pinned to issue."

This is the only step that crosses out of the skill files into code. Steps 1-4 are skill-resident. Step 5 is code-resident.

## Where unbounded data lives

The loop produces accumulating data. To keep `coily-*-meta` SKILL.md files bounded:

- **Findings:** per-skill `findings/YYYY-MM-DD-<slug>.md` siblings. Write-once. The directory IS the index. SKILL.md does not mirror it.
- **Audit-log scans:** the JSONL files are the primary record. A sweep produces findings, but the raw rows stay where they are. Do not copy raw audit data into skills.
- **Incident write-ups:** same shape as a finding, just sourced from an incident rather than a sweep.
- **Aggregations across findings:** if `findings/` grows beyond casual scan, write a Python helper in the skill dir that walks frontmatter and produces a report. Do not hand-curate aggregated views into SKILL.md. See `coily-skill-authoring` for the bias rule.

## Where bounded data lives

Each `coily-*-meta` SKILL.md carries:

- **Anti-signals catalogue.** Bounded - dozens of entries over the life of the skill.
- **Sequencing rules.** Bounded - dozens at most.
- **References.** Bounded - code paths, audit log paths, composes-with pointers, the `findings/` directory pointer.

That's it. No raw data, no per-day logs, no aggregated reports.

## When to start a new `coily-*-meta` skill

When a new top-level coily verb or grouped area lands and earns its first finding. Naming follows `coily-<area>-meta` for the design surface and `coily-<area>-usage` for the auto-generated invocation reference. See `coily-skill-authoring` for the convention.

Do not stamp empty `coily-*-meta` skills speculatively beyond a `.keep` placeholder. The convention is documented. New skills earn their SKILL.md when they earn their first finding.

## When to retire a rule or anti-signal

When the runtime fact that produced it changes (the verb is removed, the iam policy is restructured, the gate now catches what it did not before). Retire by deleting the entry from SKILL.md and adding a final finding noting the retirement. The deleted entry's git history is the audit trail.

A rule that has been pinned to a closed issue with a passing test is not retired - it is validated and stays in the catalogue as the codified version of what was learned.

## Composes with

- `coily-skill-authoring` - rule shape, naming convention, hoist guidance, write-once-findings rule.
- `coily-shared-meta` - host fleet, audit architecture, hoisted generic ops rules.
- `coily-security-boundary-discipline` - the practice this loop maintains for coily's security boundary specifically.
- Every `coily-*-meta` skill - the loop's outputs.

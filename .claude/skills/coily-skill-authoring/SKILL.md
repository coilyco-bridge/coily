---
name: coily-skill-authoring
description: Conventions for authoring `coily-*-meta` and `coily-*-usage` skills. Naming convention (`coily-<area>-meta` for design surface, `coily-<area>-usage` for auto-generated invocation reference), the catalogue-vs-rule entry shape, the write-once findings rule, the hoist-vs-stamp guidance for shared rules, the bias toward Python helpers, the flat-not-nested rule, and the encode-the-why principle. Use when creating a new `coily-*-meta` skill, when extending an existing one, when deciding whether a rule should live in `coily-shared-meta` vs a per-verb skill, or when reviewing whether a skill's structure matches the convention. Triggers - skill authoring, coily skill, SKILL.md, frontmatter, naming convention, coily-*-meta, coily-*-usage, hoist vs stamp, write-once findings, encode the why, Python helpers, flat not nested, anti-signal entry shape, sequencing rule shape.
---

# coily-skill-authoring

Conventions for authoring skills under `coily/.claude/skills/`. Read this before creating or editing any `coily-*-meta` skill.

Composes with: `coily-meta-improvement` (the loop the skills produce and consume), `coily-shared-meta` (where hoisted generic rules live).

## Location

All coily skills live at `coily/.claude/skills/<name>/SKILL.md`. Flat directory. The `setup` verb on coily symlinks each into `~/.claude/skills/<name>` so the harness picks them up.

Friends shipping coily get the full skill set via the brew formula's staged share dir. The skills are part of coily's surface, not a separate distribution.

## Naming

Predictable, flat, derived from the cli surface:

- **`coily-<area>-meta`** - design surface for one area. Carries anti-signals, sequencing rules, references. Hand-authored. Examples: `coily-ops-aws-meta`, `coily-gaming-eco-meta`, `coily-audit-meta`.
- **`coily-<area>-usage`** - invocation reference for one area. Auto-generated from the live cli tree. Flat lookup, no design content. Examples: `coily-ops-aws-usage`, `coily-gaming-eco-usage`.
- **`coily-shared-meta`** - rules and inventory shared across all areas. Host fleet, audit architecture, generic ops sequencing rules. Hand-authored.
- **`coily-shared-usage`** - the full auto-generated lookup table covering every coily verb. Per-verb `coily-<area>-usage` skills are slices of this. `coily-shared-usage` is the whole thing.
- **`coily-meta-improvement`** - the loop that produces findings and rules.
- **`coily-skill-authoring`** - this file.
- **`coily-security-boundary-discipline`**, **`coily-ops-investigation`** - friend-shippable copies of the cross-cutting practices.

The `<area>` matches the cli verb path: `coily ops aws` → `coily-ops-aws-*`. `coily gaming eco` → `coily-gaming-eco-*`. Sub-verbs (`coily ops aws s3 ls`) do not get their own skills. They are entries inside the area skill.

Do not nest. Every skill is a peer directory. The harness's symlink loader only walks top-level entries.

## Frontmatter

Two required fields: `name` and `description`. The `description` is keyword-matched for triggering, so pack aliases and natural-language phrasings into it. Lead with the canonical purpose, end with a `Triggers - <comma-list>` block.

Frontmatter goes in YAML. No other fields are required.

## Catalogue entries (anti-signals, references)

Two-line shape:

- Lead line: the claim or pointer.
- `**Pin:**` line: links to the finding, issue, or test that grounds the entry. Omit the Pin line when the entry is seeded but not yet grounded. Absence-of-Pin means seeded.

Do not write a status field. Findings are write-once and GitHub issues describe followups. If you want to know whether a Pin is still live, read the issue.

Example:

```markdown
- **"audit row is sufficient for read-only verbs."** False. The trail documents the leak. It does not prevent it.
  **Pin:** [findings/2026-05-05-read-only-audit-without-gate.md](findings/2026-05-05-read-only-audit-without-gate.md), [coily#58](https://github.com/coilysiren/coily/issues/58).
```

## Rule entries (sequencing rules)

Three-line shape:

- Lead line: the rule, as an imperative or claim.
- `**Why:**` line: the originating finding, issue, or constraint. Where the why is empirical, link the `findings/` file rather than restating the evidence inline.
- `**How to apply:**` line: when the rule fires.

Example:

```markdown
- **New iam permission lands in `cli-guard/policy` before the verb that uses it.**
  **Why:** the gate must reject before the underlying call can succeed. A verb shipping ahead of its policy entry passes through unvalidated for the time-between.
  **How to apply:** any PR that adds an aws sub-verb.
```

## Findings (write-once, unbounded)

Per-skill `findings/YYYY-MM-DD-<slug>.md` siblings. The structure of a finding and the loop that produces them is documented in `coily-meta-improvement`.

The shape that matters here:

- One file per observation. Never edited after creation.
- Frontmatter: `date`, `slug`, optional `promoted_to`.
- Body: what was observed / why it slipped / rule it produced.
- Followup state lives on the GitHub issue the finding cites, not in the finding.
- The directory IS the index. SKILL.md does not mirror it. If you need an aggregated view across findings, write a Python helper.

## Hoist vs. stamp

Rules that apply to multiple `coily-*-meta` skills live once, in `coily-shared-meta`. Per-area skills only carry rules specific to that area.

The test: if a rule mentions only generic boundary mechanics (policy-before-verb, deny-by-default for destructive, remove-policy-in-same-commit), it is shared. If a rule mentions the area's specific resource (s3 buckets, k8s contexts, the eco systemd unit), it stays in the area skill.

When in doubt, hoist. Per-area duplication is the worse failure mode: 10 places to update one rule, drift inevitable.

## Bias toward Python helpers

When a skill needs to parse files, walk a directory, query a JSONL or SQLite, or do any structured data manipulation, write a Python script in the skill directory and have SKILL.md call it. Pure prompt instructions are fine for narrative steps. Python is right for anything where determinism, speed, or testability matter.

Helpers go in the skill dir alongside SKILL.md, get committed, run with the system `python3`. Stdlib-first. Reach for third-party libraries only when stdlib genuinely doesn't suffice.

The LLM tier should focus on synthesis, not parsing. Procedure-as-prompt loses fidelity each time it is re-derived. Committed Python is auditable, fast, and the same on every host.

## Encode the why, not just the what

Every rule and anti-signal exists because something happened that produced it. Capture the something, not just the rule. A rule with no why is harder to retire correctly when the situation changes.

The catalogue and rule shapes above (Pin lines, Why lines) are the structural enforcement of this principle.

## Flat, not nested

Every skill is a peer directory directly under `.claude/skills/`. Do not nest sub-skills inside another skill's directory. Nested-skill discovery is poorly supported by the harness, and the symlink loader only walks top-level skill dirs.

When a meta-skill needs to route to other skills, the routed skills live as flat peers alongside it. The meta's job is to name them and describe when each fires. The loader handles each one independently.

## When to create a new skill

When an area earns its first finding. See `coily-meta-improvement` for the loop.

Do not pre-stamp empty `coily-*-meta` skills speculatively. An empty meta skill is documentation noise.

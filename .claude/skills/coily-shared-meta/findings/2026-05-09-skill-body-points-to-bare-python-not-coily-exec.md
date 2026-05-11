---
date: 2026-05-09
slug: skill-body-points-to-bare-python-not-coily-exec
promoted_to:
  - issue: https://github.com/coilysiren/coilyco-ai/issues/254
---

# 2026-05-09 - daily-backlog SKILL.md leads with bare `python3 .../script.py`, harness denies before coily ever sees it

## What was observed

No coily audit row exists for this denial because the rejection happened upstream of the gate. The agent followed `coily-ai/.claude/skills/daily-backlog/SKILL.md` verbatim, which under `## Procedure` says:

    python3 ~/projects/coilysiren/coilyco-ai/.claude/skills/daily-backlog/script.py

Two consecutive Bash tool calls with that argv (and a `2>&1 | tail -N` variant) returned `Permission to use Bash with command python3 ... has been denied`. The agent then retried as:

    coily exec daily-backlog

which succeeded (audit row id `019e0b18-...`, ts 1778302638, verb resolved to coily exec).

The skill body still treats the bare python invocation as canonical. The `coily exec daily-backlog` form is only reachable by reading the surrounding context (the Notes mention idempotency and replay subcommands implicitly via the script). Kai called this out as `coily audit finding` in chat: "you reached for `.claude/skills/daily-backlog/script.py` when `coily exec daily-backlog` already exists? its literally the same command."

## Why it slipped

The skill predates the `coily exec daily-backlog` verb. When the verb landed (recently, as part of the dailies-migration thread visible in coilyco-ai recent commits: `26ebdd4 add daily-<name>-auth coily exec verbs`, `7d625e9 add remaining daily-* routines as coily exec entries`, etc.), the skill body was not updated to lead with the wrapper. The bare-python invocation stayed at the top of `## Procedure` as a documented contract.

The harness deny rule for bare `python3 <path>/script.py` is correct - that path is not on the allowlist, and routing it through `coily exec` is the audited equivalent. The friction is one-sided: the skill points the agent at a non-audited path, the harness denies, the agent has to discover the wrapper from elsewhere.

This is a sibling shape to `2026-05-08-missing-ops-prefix-opaque-flag-error`. There the recovery was "did-you-mean coily ops aws"; here the recovery is "skill body should have led with coily exec daily-backlog." Both are dictation-friendliness regressions: the agent (and a tired human on the train) follows the literal canonical text.

The audit log not seeing the denial is itself a meta-finding: harness-level Bash denies for skill-prescribed commands are invisible to the coily audit pipeline. The meta-improvement loop sees the friction only when a human files it as a finding by hand, like Kai did here.

## Rule it produced

Candidate sequencing rule for `coily-shared-meta`: "Skill bodies for any routine that has a `coily exec <name>` entry must lead with the wrapper, not the underlying script invocation. Bare `python3 path/script.py` lines stay only as replay-debug detail (and only after the wrapper form is documented as the canonical entry)."

Candidate forward action on coilyco-ai: sweep `~/projects/coilysiren/coilyco-ai/.claude/skills/daily-*/SKILL.md` and any other routine skill where the body opens with `python3 <path>` while a matching `coily exec <name>` verb exists. Replace the bare invocation with the wrapper. Keep the script path as a footnote labeled "underlying script" or "for direct replay." This is one PR, ~9 files, mechanical.

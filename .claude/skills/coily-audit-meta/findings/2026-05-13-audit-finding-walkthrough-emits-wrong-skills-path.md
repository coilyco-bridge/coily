---
date: 2026-05-13
slug: audit-finding-walkthrough-emits-wrong-skills-path
promoted_to:
  - issue: https://github.com/coilysiren/coily/issues/148
---

# 2026-05-13 - audit finding walkthrough prints a coilyco-ai path for findings that actually live under coily

## What was observed

Running `coily --commit-scope=/home/kai/projects/coilysiren/coilyco-ai audit finding --id 019e23b4-73d9-7829-be9f-dd2529090b90 --slug ops-aws-passthrough-mangles-query-output-flags` (audit row `019e23bb-9a70-7cc4-bdfe-fa2b60f5dc84`, ts 1778715957, verb `audit.finding`) emitted a Step 3 instruction with this path:

```
/home/kai/projects/coilysiren/coilyco-ai/.claude/skills/coily-<area>-meta/findings/YYYY-MM-DD-<slug>.md
```

The `coily-*-meta` skill directories do not exist under `coilyco-ai/.claude/skills/`. They live under `coily/.claude/skills/`. The "References" block at the end of the walkthrough reinforces the same wrong location for `coily-meta-improvement/SKILL.md` and `coily-skill-authoring/SKILL.md`.

The agent following the walkthrough caught this on contact (the target directory was missing, ran a `find` over `/home/kai/projects/coilysiren` to locate the real `coily-ops-aws-meta`, then wrote the finding to the correct path under `coily/`). The walkthrough did not catch itself.

## Why it slipped

The walkthrough copy was probably authored when the meta skills lived (or were planned to live) under `coilyco-ai`, and the path string was hardcoded into the walkthrough emitter. When the meta skills landed under `coily/.claude/skills/` instead (where they sit alongside the source they document), the walkthrough emitter wasn't updated. Nothing in the build verifies the printed path resolves on disk before shipping a coily release.

This is a class of drift the audit-finding loop is uniquely vulnerable to. The walkthrough is the onboarding doc for new agents writing findings. If it points to a wrong path and an agent trusts it verbatim, the finding lands in an orphan tree under `coilyco-ai/.claude/skills/coily-<area>-meta/findings/` that the meta-improvement loop never reads from. The bad outcome is a finding that quietly disappears, not a loud failure.

## Rule it produced

Anti-signal candidate (data only, not promoted): "the path printed by `coily audit finding`'s walkthrough is the canonical location to write the finding." Today it is not. Verify the directory exists under `coily/.claude/skills/coily-<area>-meta/findings/` before writing. Forward action filed at [coily#148](https://github.com/coilysiren/coily/issues/148).

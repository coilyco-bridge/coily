---
date: 2026-05-08
slug: three-bare-gh-denials-before-wrapper-reach
promoted_to:
  - sequencing-rule: "harness deny on a bare external command -> retry through the coily wrapper, then run `coily audit finding`"
  # - issue: <url> (rate-limited at filing time, deferred)
---

# 2026-05-08 - Agent burned three bare `gh issue create` denials before reaching for `coily ops gh`

## What was observed

In a single turn the agent tried three permutations of bare `gh issue create` against `coilysiren/coily` and got the harness "Permission to use Bash with command X has been denied" message each time. After the third denial the agent surfaced the question to Kai instead of routing through `coily ops gh`. Kai had to instruct the agent explicitly that bare-command denials should retry via the coily wrapper, and that any denial in any context warrants a `coily audit finding`.

Concrete denied argv shapes from this turn (no audit row exists - denial is at the Claude Code permission layer, before coily is invoked):

- `gh issue create --repo coilysiren/coily --title "..." --body "..."`
- `gh issue create --repo coilysiren/coily --title "..." --body "..."` (shorter body, retry)
- `gh issue create -R coilysiren/coily -t "..." -b "..."` (short flags, retry)

Once routed through `coily --commit-scope=/Users/kai/projects/coilysiren/coily ops gh issue create ...`, the call passed argv validation and reached gh, which then failed upstream with a GitHub GraphQL rate-limit error (exit_code=3, kind=upstream_failed). The wrapper path was the right path; the harness denial taught the agent the wrong lesson on the first three attempts.

## Why it slipped

Same root gap as 2026-05-05-claude-bypasses-coily-gh-wrapper: the harness deny message does not name `coily ops gh` as the alternative, and the operating-context doc (AGENTS.md "Coily permission discipline") read denial-as-stop without carving out the wrapper-retry case. The agent's stop-on-denial rule was correctly triggered for the lockdown/security-config edge, but applied too broadly to ordinary external commands the wrapper exists for.

This finding is the second data point in 4 days for the same shape. The 2026-05-05 finding was a 35-day sweep; this one is a single live turn. Frequency confirms the rule has not been internalized from the 2026-05-05 finding alone.

## Rule it produced

Sequencing rule, now landed in `coilyco-ai/AGENTS.md` "Coily permission discipline": when the harness denies a bare external command (`gh`, `aws`, `kubectl`, `docker`, `tailscale`), retry through the coily wrapper rather than stopping. Any harness denial - in any context - also triggers a `coily audit finding` so the meta-improvement loop sees the friction.

The carve-out preserves the original "denial means stop" rule for the cases it was written for: `coily lockdown`, `.claude/settings*.json` hand-edits, deny-loosening operations. Those still stop on denial.

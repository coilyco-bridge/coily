---
date: 2026-05-05
slug: claude-bypasses-coily-gh-wrapper
promoted_to:
  - anti-signal: "the wrapper exists, therefore the agent uses it"
  - issue: https://github.com/coilysiren/coily/issues/61
---

# 2026-05-05 - 113 raw `gh` invocations denied by Claude in 35d while `coily ops gh` exists and works

## What was observed

Sweep of `~/.claude/projects/**/*.jsonl` over the 35-day window: 113 `Permission to use Bash with command X has been denied` entries where the denied command begins with `gh`. The denied invocations are mostly read-only:

- `gh repo list coilysiren --limit 50`
- `gh issue list --repo coilysiren/eco-mcp-app --state open`
- `gh run list --repo coilysiren/repo-recall --branch main --limit 5`
- `gh issue view 21 --repo coilysiren/coilyco-ai`
- `gh api repos/coilysiren/coilyco-ai/issues/21`

In the same window, `coily gh.*` verbs landed 1000+ audit rows (gh.run.list, gh, gh.issue.create, gh.api, gh.search.issues, gh.issue.view, etc.). The wrapper exists and is exercised. Claude reaches for raw `gh` 113 times anyway and gets blocked at the Claude-Code permission boundary.

By comparison: only 2 raw `docker` and 2 raw `kubectl` denials in the same window. `gh` is the standout.

## Why it slipped

Two compounding gaps:

1. **Claude's permission rules do not advertise `coily ops gh` as the auto-approved alternative.** The Bash permission denial fires before Claude has a chance to route to the wrapper. The agent learns "gh is blocked" but not "coily ops gh is the path."
2. **The 96.6% argv rejection rate on `gh.run.list` (see `metachar-gate-context-blind` finding) makes the wrapper appear unreliable.** When the agent does try the wrapper, it gets rejected. When it tries raw, it gets denied. The asymmetry teaches the agent that gh is fundamentally hard, not that it should be routed differently.

The wrapper's existence does not enforce its use. Lockdown is the mechanism that does, but the audit shows lockdown was applied only 12 times in the window with 0 failures - it works, but its rollout is incomplete across operator sessions.

## Rule it produced

Anti-signal: **"the wrapper exists, therefore the agent uses it."** False. The agent uses the path of least denial. Raw `gh` denied by Claude Code without naming the wrapper is the path the agent learns. Without lockdown actively in place for the session, raw is the default reach.

The forward shape: ensure Claude Code's permission denial messages name the wrapper. Either (a) inject a hint into the deny text from coily's lockdown layer ("blocked - try `coily ops gh ...`"), or (b) make lockdown the default-on state for any session that has coily installed.

Linked: the metachar finding above explains why the wrapper sometimes feels broken when reached for. Both findings need to land together for the gh story to actually improve.

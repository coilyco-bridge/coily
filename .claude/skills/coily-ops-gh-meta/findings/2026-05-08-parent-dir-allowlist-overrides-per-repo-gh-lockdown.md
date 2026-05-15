---
date: 2026-05-08
slug: parent-dir-allowlist-overrides-per-repo-gh-lockdown
promoted_to:
  - issue: https://github.com/coilysiren/coily/issues/61
---

# 2026-05-08 - Bare `gh issue` ran wrapper-free on May 8 because parent-dir `.claude/settings.local.json` allow-listed it

## What was observed

Working session began at cwd `/Users/kai/projects/coilysiren/` (the parent directory containing every `coilysiren/*` repo clone; itself not a git repo). During the session the agent ran three bare `gh` invocations from inside subordinate repos:

- `gh issue list --search "gif OR fixture..." --state all --limit 20` (run with cwd at `coilysiren/otel-a2a-relay`)
- `gh issue view 92 --json title,body,state` (same cwd)
- `gh issue create --title "Pin GIF fixture..." --body ...` (same cwd)
- `gh issue list --search "openclaw OR JOURNAL OR ..."` (run with cwd at `coilysiren/agentic-os-kai`)
- `gh issue view 98 --json ...` (same cwd)

All five succeeded without prompting and produced **zero** audit rows. Confirmed:

- `/Users/kai/.coily/audit/coilysiren-otel-a2a-relay.jsonl` was 0 bytes, last touched May 7 01:51, despite `gh issue create` against that repo on May 8.
- `/Users/kai/.coily/audit/coilysiren-agentic-os-kai.jsonl` last touched May 7 23:08, no May 8 rows for the bare `gh` calls.

Each subordinate repo's `.claude/settings.json` correctly contains `"Bash(gh:*)"` in its deny list - so the per-repo lockdown is intact. The bypass came from `/Users/kai/projects/coilysiren/.claude/settings.local.json`:

```json
{
  "permissions": {
    "allow": [
      "Bash(gh issue *)",
      ...
    ]
  }
}
```

That allow lives at the session-start cwd, which is the parent of every locked-down repo. Claude Code's permission engine appears to evaluate against the session-start project root, not the per-Bash-command cwd, so the parent-dir allow took precedence over the per-repo deny for the entire session.

## Why it slipped

Distinct from the 2026-05-05 `claude-bypasses-coily-gh-wrapper` finding, which observed denials. Here the bypass succeeded silently because the allow-list rule was already in place at a scope that dominates per-repo lockdown.

Three compounding gaps:

1. **The session-start cwd was not a git repo, so no per-repo lockdown could anchor.** The agent's effective project root was the multi-repo parent, where lockdown does not live and is not expected to live.
2. **A parent-dir `Bash(gh issue *)` allow rule exists.** Whatever its original purpose (likely convenience for org-wide issue triage), it now silently shadows every child repo's `Bash(gh:*)` deny for any session that starts at the parent.
3. **The bypass produces no audit signal.** The wrapper-existence-doesn't-enforce-use anti-signal from the prior finding assumed denial-as-feedback. Here there's no denial and no row - the only evidence is downstream (issues filed on github.com without matching coily audit rows). A sweep over `~/.claude/projects/**/*.jsonl` for "denied" entries will not surface this class.

The May 5 finding's forward shape ("make lockdown the default-on state for any session that has coily installed") would not have caught this case either, because lockdown *was* default-on for the child repos. The shadowing happens above lockdown.

## Rule it produced

Anti-signal: **"per-repo lockdown is sufficient when sessions start above the repo."** False. A parent-dir `.claude/settings.local.json` allow overrides every child-dir deny for the session, and the multi-repo parent is a common session-start cwd for cross-repo work. Lockdown discipline has to extend to the session-root cwd actually used, not just the per-repo roots.

Forward shape candidates (file under coily#61):

- Have `coily lockdown` warn or refuse when applied to a child repo whose ancestor `.claude/settings*.json` carries broader permissions.
- Add a `coily lockdown --recursive` mode that walks up from any locked-down repo and either (a) refuses to leave broad allows above, or (b) installs a sibling `.claude/settings.json` at the session-root cwd that re-asserts the deny.
- Treat absence-of-audit-rows as a first-class signal: a periodic check that compares `gh issue create` events on github.com against `coilysiren-<repo>.jsonl` rows for the same repo, flagging the gap.

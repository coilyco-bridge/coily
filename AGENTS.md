# Agent instructions

Workspace-level conventions (git workflow, test/lint autonomy, readonly ops, writing voice, deploy knowledge) are loaded globally via `~/.claude/CLAUDE.md` -> `coilyco-ai/AGENTS.md`. This file covers only what's specific to this repo.

---

## Editing coily from other repos: forbidden, file an issue

When the session's primary cwd is **not** `/Users/kai/projects/coilysiren/coily` (check the env block, not live cwd), do not edit any file in this repo. Not source, not config, not docs, not AGENTS.md, not even a typo fix.

**Instead:** open a GitHub issue with `coily gh issue create --repo coilysiren/coily ...` (or `gh issue create --repo coilysiren/coily ...` if coily isn't installed). Include enough detail to act on cold. Cross-link the originating session/PR if relevant.

**Why:** coily is the wrapper that enforces the lockdown deny list (`aws`, `kubectl`, `ssh`, `scp`). A drive-by edit from a sibling-repo session can silently weaken the wrapper - add a passthrough subcommand, relax a filter, ship it, and the next coily run passes the deny list with the new behavior. The cwd gate forces explicit intent: a coily change must happen in a session Kai consciously started for coily work.

This rule is stricter than the general workspace git policy. It overrides "commit directly to main" for this one repo.

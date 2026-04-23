# Agent instructions

See `../AGENTS.md` for workspace-level conventions (git workflow, test/lint autonomy, readonly ops, writing voice, deploy knowledge). This file covers only what's specific to this repo.

**Exception:** auto-push to main only when this repo is the session's primary cwd (check env block, not live cwd). If a session started in a sibling and cd'd in, do not auto-push.

---

## Auto-commit policy

When this repo is your primary working directory, `git add`, `git commit`, and `git push` to `main` automatically after each successful change. No confirmation needed.

**Why:** coily is Kai's solo operator CLI. PR review overhead is pure friction here.

**How to apply:**

- Only when the session's primary working directory is `/Users/kai/projects/coilysiren/coily` (check the env block at session start). If you were started in a sibling repo and later `cd`'d into coily, do not auto-push - defer to the parent repo's policy.
- Group related edits into one commit. Don't push half-finished work.
- Skip the auto-push if the change touches release workflows, secrets, or anything Kai flagged as needing review.

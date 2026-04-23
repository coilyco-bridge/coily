# Agent instructions

## Auto-commit policy

When this repo is your primary working directory, `git add`, `git commit`, and `git push` to `main` automatically after each successful change. No confirmation needed.

**Why:** coily is Kai's solo operator CLI. PR review overhead is pure friction here.

**How to apply:**

- Only when the session's primary working directory is `/Users/kai/projects/coilysiren/coily` (check the env block at session start). If you were started in a sibling repo and later `cd`'d into coily, do not auto-push - defer to the parent repo's policy.
- Group related edits into one commit. Don't push half-finished work.
- Run the project's checks (build, test, lint) before pushing. If they fail, fix and re-stage rather than pushing red.
- Never `--no-verify`, never force-push.
- Skip the auto-push if the change touches release workflows, secrets, or anything Kai flagged as needing review.

## Prose style

Kai's writing voice guide applies to all prose (commit messages, PR bodies, READMEs, docs):

- `/Users/kai/projects/coilysiren/coilyco-vault/Obsidian Vault/Self/writing-voice.md`

High-frequency rules:

- No em-dashes. Use a period, comma, parens, or ` - ` (hyphen with spaces).
- No italics, ever.
- No semicolons in prose. (Code is fine.)
- Bold only for structural anchors, not mid-sentence emphasis.
- Decision or answer in sentence one.
- One emoji per message max.
- Fragments are complete sentences.

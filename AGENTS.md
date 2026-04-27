# Agent instructions

Workspace-level conventions (git workflow, test/lint autonomy, readonly ops, writing voice, deploy knowledge) are loaded globally via `~/.claude/CLAUDE.md` -> `coilyco-ai/AGENTS.md`. This file covers only what's specific to this repo.

---

## Editing coily from other repos: forbidden, file an issue

When the session's primary cwd is **not** `/Users/kai/projects/coilysiren/coily` (check the env block, not live cwd), do not edit any file in this repo. Not source, not config, not docs, not AGENTS.md, not even a typo fix.

**Instead:** open a GitHub issue with `coily gh issue create --repo coilysiren/coily ...` (or `gh issue create --repo coilysiren/coily ...` if coily isn't installed). Include enough detail to act on cold. Cross-link the originating session/PR if relevant.

**Why:** coily is the wrapper that enforces the lockdown deny list (`aws`, `kubectl`, `ssh`, `scp`). A drive-by edit from a sibling-repo session can silently weaken the wrapper - add a passthrough subcommand, relax a filter, ship it, and the next coily run passes the deny list with the new behavior. The cwd gate forces explicit intent: a coily change must happen in a session Kai consciously started for coily work.

This rule is stricter than the general workspace git policy. It overrides "commit directly to main" for this one repo.

---

## Release framework

Every push to `main` triggers `.github/workflows/release.yml`, which fully automates versioning and Homebrew distribution. No manual `make release`, no manual tag, no manual PR.

**Per-push flow:**

1. `mathieudutour/github-tag-action` computes the next semver from commits since the last tag. `default_bump: patch` means *every* push releases at least a patch.
   - plain commit -> patch bump
   - `feat: ...` -> minor bump
   - `feat!: ...` or body containing `BREAKING CHANGE:` -> major bump
2. The new tag is created and pushed. Unlike repo-recall, there is no source-tree version bump: `main.Version` is set by the Formula at build time via ldflags.
3. A GitHub Release is cut with the auto-generated changelog.
4. The `bump-tap` job downloads the new tarball, computes its sha256, and pushes the updated Formula directly to `main` on `coilysiren/homebrew-tap`. No PR.

**Loop safety:** the tag is created by an action using `GITHUB_TOKEN`, which by GitHub policy doesn't re-trigger workflows. So the release job won't recurse on its own tag.

**Secret required:** `HOMEBREW_TAP_TOKEN` - fine-grained PAT scoped to `coilysiren/homebrew-tap` with `Repository permissions -> Contents: Read and write`. Set via `gh secret set HOMEBREW_TAP_TOKEN --repo coilysiren/coily`.

**Formula source of truth:** `Formula/coily.rb` in `coilysiren/homebrew-tap`. The bump-tap job edits `url` + `sha256` only - any structural change to the formula (new dependency, install steps, test block) is a hand-edit on the tap and is not overwritten on release.

**Brew install caveat:** the Homebrew install path produces a *user-writable* `/opt/homebrew/bin/coily`, which loses the root-owned-binary property that `make install` preserves on unix. Brew is for fresh-machine bootstrap; the canonical install remains `make install` from a checkout.

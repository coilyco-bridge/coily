# 21. Skill install chain

**What it is**: The generated skill at `<repo>/skill/` is symlinked into `~/.claude/skills/coily/`. Regenerations via `make skill` propagate automatically through the symlink.

**How to invoke**: `make install-skill` once. After that, `make skill` updates the live skill.

**Expected shape**: `~/.claude/skills/coily` is a symlink. `~/.claude/skills/coily/SKILL.md` has valid frontmatter.

**Test prompt**:
> Verify `~/.claude/skills/coily` is a symlink to `<coily-repo>/skill`. Cat its SKILL.md and assert the frontmatter has `name: coily` and a non-empty `description:`. Then `make skill` in the repo, re-cat the skill, assert the file is still readable and the content may have changed.

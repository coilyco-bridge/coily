# 10. `coily skill-gen` (dev only)

**What it is**: Regenerates `skill/SKILL.md` and `skill/reference/*.md` from `configs/commands/*.yaml`.

**How to invoke**: `./bin/coily-dev skill-gen` from repo root. `--commands-dir` and `--out` flags for custom paths.

**Expected shape**: SKILL.md with frontmatter (`name: coily`, `description: ...`). One reference file per manifest. Deterministic output across runs.

**Test prompt**:
> In the coily repo, run `make skill` and assert `skill/SKILL.md` and `skill/reference/{aws,gh,kubectl}.md` exist. Save the md5 of each. Run `make skill` again and assert the md5s are unchanged (determinism). Assert `coily skill-gen` is NOT in the prod binary (`./bin/coily skill-gen` should fail with "No help topic").

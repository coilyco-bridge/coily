# 11. `coily install-skill` (dev only)

**What it is**: Symlinks `./skill/` into `~/.claude/skills/coily/`.

**How to invoke**: `./bin/coily-dev install-skill [--force]` from repo root.

**Expected shape**: Creates a symlink. Refuses to clobber existing without `--force`.

**Test prompt**:
> Run `make install-skill` from the repo. Assert `~/.claude/skills/coily` is a symlink pointing at `<repo>/skill`. Assert `~/.claude/skills/coily/SKILL.md` is readable through the symlink. Assert `./bin/coily install-skill` (prod binary) fails with "No help topic".

# 3. `coily lockdown`

**What it is**: Writes or merges `.claude/settings.json` with coily's canonical allowlist/denylist.

**How to invoke**:
- `coily lockdown` - dry-run, print plan
- `coily lockdown --apply` - write to `.claude/settings.json`
- `coily lockdown --local --apply` - write to `.claude/settings.local.json`
- `coily lockdown --replace --apply` - clobber existing allow/deny instead of merging

**Expected shape**: Dry-run prints JSON to stdout. Apply writes a settings.json with `permissions.allow`, `permissions.deny`, `deniedMcpServers`. Existing top-level keys preserved. Existing allow/deny merged unless `--replace`.

**Test prompt**:
> In a temp directory, verify `coily lockdown` without flags prints a valid JSON plan. With `--apply` it creates `.claude/settings.json` with 0600 perms. Running it twice does not duplicate entries. Running it against a pre-existing settings.json with a custom allow rule preserves the custom rule and unrelated top-level keys. Running with `--replace --apply` removes the custom rule. Clean up the temp dir when done.

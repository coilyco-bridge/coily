# 3. `coily lockdown`

**What it is**: Writes or merges `.claude/settings.json` with coily's canonical allowlist/denylist.

**How to invoke**:
- `coily lockdown` - dry-run, print plan
- `coily lockdown --apply` - write to `.claude/settings.json`
- `coily lockdown --local --apply` - write to `.claude/settings.local.json`
- `coily lockdown --replace --apply` - clobber existing allow/deny instead of merging

**Expected shape**: Dry-run prints JSON to stdout. Apply writes a `settings.json` with `permissions.allow`, `permissions.deny`, and `hooks.PreToolUse` (Bash matcher pointing at the generated `lockdown-deny.sh`). Existing top-level keys preserved. Existing allow/deny merged unless `--replace`.

**Scope**: Bash-only. MCP-server allowlisting is intentionally not in scope. The Bash deny list gates shell-level blast radius (cluster mutations, secret reads, package installs). MCP-server gating answers a different question - "is this MCP server trustworthy" - which is per-user / per-machine, not per-repo. Manage MCP servers at the user-settings level instead.

**Two enforcement layers**:

1. **`permissions.deny`** — the built-in Bash command-prefix matcher. Primary gate on the CLI. Silently bypassed in Claude Desktop agent mode.
2. **`hooks.PreToolUse` → `.claude/lockdown-deny.sh`** — a generated POSIX sh script that mirrors the same deny list as a `case` statement. Reads tool-input JSON on stdin, splits the command at shell separators, strips leading `env VAR=val` tokens, and exits 2 with a stderr message if any segment's leading token matches a deny prefix. Hooks fire under both the CLI and Desktop, so this is the only enforcement path on Desktop and defense-in-depth on the CLI. Rerun `coily lockdown --apply --replace` to regenerate after a coily upgrade if the deny list grew.

**Non-repo parent dirs supported**: lockdown does not require the target directory to be a git repo. Run it in a workspace root that contains many repos and Claude Code's normal config-resolution will apply the rules to every descendant.

**Test prompt**:
> In a temp directory, verify `coily lockdown` without flags prints a valid JSON plan. With `--apply` it creates `.claude/settings.json` with 0600 perms. Running it twice does not duplicate entries. Running it against a pre-existing settings.json with a custom allow rule preserves the custom rule and unrelated top-level keys. Running with `--replace --apply` removes the custom rule. Clean up the temp dir when done.

**Known limitation: Claude Desktop agent mode on Windows does not enforce the Bash deny list.** Verified 2026-04-23 on Claude Code v2.1.119. Identical repo and `.claude/settings.json` produce different behavior depending on host:

- `claude` CLI in Git Bash: `Bash(python:*)` fires, command blocked.
- Claude Code inside Claude Desktop (MSIX-packaged agent mode): `/permissions` shows the deny rule loaded, but the Bash tool runs `python --version` without consulting it.

The `PowerShell` / `PowerShell(*)` denies still fire in both hosts because they go through a different matcher. The implication is that `coily lockdown` is **CLI-only enforcement** for Bash rules. Agent sessions run from Claude Desktop effectively run with Bash permissions wide open, regardless of what `lockdown` wrote into the project. Prefer the CLI for any agent work that relies on lockdown for safety.

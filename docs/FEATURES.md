# Coily features

Baseline inventory of what coily does today. The point is to make scope changes
visible. When a feature is added, removed, or materially reshaped, update the
matching bullet in the same commit so this file stays a faithful mirror of the
binary.

Coily is a single-binary CLI security boundary. It wraps privileged ops (aws,
gh, kubectl, ssh, docker, tailscale, package managers, game-server systemd) in
named verbs, validates argv, and writes a JSONL audit row for every
invocation. There is no `coily shell`, `coily run`, or `coily ssh exec` escape
hatch by design.

## Architecture

- **Single binary** - One Go binary, one entry in the Claude Code allowlist.
- **Trust inversion** - Narrow allowlist of named verbs instead of broad denylist.
- **Escape-hatch resistant** - No `shell`, `run`, or free-form `exec` verbs.
- **Argv validation** - Every string arg checked for shell metacharacters at load and at invocation.
- **Scope binding** - Every audit row bound to a git toplevel (`--commit-scope` or `$COILY_COMMIT_SCOPE`). No opt-out.
- **Verb.Spec abstraction** - Uniform validation, action, audit pipeline for every command.

## Verb surface

### Built-in top level

- `coily whoami` - Unified identity check across aws, kubectl, gh.
- `coily version` - Print build version.
- `coily --list` - Enumerate all verbs (built-in plus repo).
- `coily --tree` - Print full command tree recursively.
- `coily setup` - Idempotent post-upgrade rituals (completion refresh, skill symlinks, lockdown rebaseline, user hook install).
- `coily install-completion` - Bash, zsh, fish completion install with `--dry-run`.

### CLI passthroughs (`coily ops`)

- `coily ops aws` - Verbatim forward to aws CLI.
- `coily ops gh` - Verbatim forward to GitHub CLI.
- `coily ops kubectl` - Verbatim forward to kubectl with lockdown readonly/write gating.
- `coily docker` - Docker passthrough at top level.
- `coily tailscale` - Tailscale passthrough at top level.

### Package managers (`coily pkg`)

- `coily pkg pnpm | npm | yarn | bun` - JavaScript package managers.
- `coily pkg uv | pip | pipx | poetry` - Python package managers.
- `coily pkg cargo` - Rust package manager.
- `coily pkg gem | bundle` - Ruby package managers.
- `coily pkg brew` - Read-only brew verbs (search, info, list).
- `coily brew install | uninstall | upgrade | reinstall` - Scoped to coilysiren/tap/* unless `--allow-untapped`.
- `coily pkg glama` - Glama MCP server directory plus telemetry.
- `coily pkg skillsmp` - skillsmp.com v1 skill discovery.

### SSH (free-form passthrough)

- `coily ssh <alias> -- coily <subcommand> <args>` - Resolves the host alias from `.coily/coily.yaml` `ssh.targets`, ssh's in, runs the supplied coily argv on the remote. Remote coily's lockdown is the security boundary, not this client side. Audit rows chain across hosts via `--audit-parent` (local row id is pre-allocated and shipped to the remote; remote row records it). See [coily#187](https://github.com/coilysiren/coily/issues/187) for the design.
- The previous per-verb wrappers (`coily ssh systemctl`, `copy`, `deploy`, `git`, `journalctl`, `kubectl`, `fs`, `rm-unit`) got deleted in step 8 of #187 once the passthrough was proven end-to-end on the systemctl call site ([coily#191](https://github.com/coilysiren/coily/issues/191)). Replacement form: `coily ssh kai-server -- coily <equivalent>`.

### Game-server ops (`coily gaming`)

- `coily gaming eco` - Status, tail, start, stop, restart, world, mod.
- `coily gaming core-keeper` - Status, tail, start, stop, restart.
- `coily gaming icarus` - Status, tail, start, stop, restart.
- `coily gaming factorio` - Status, tail, start, stop, restart, update, saves, mods, players.
- `coily eco world {get-seed, set-seed, randomize, snapshot}` - Local WorldGenerator.eco helpers.
- `coily eco mod {list, push}` - Mod listing and push to mod.io.
- `coily factorio saves {list, backup-now}` - Save file management.
- `coily factorio mods {list, sync}` - Mod sync via Factorio mod portal API.
- `coily factorio players {list}` - Whitelist plus bans.
- `coily factorio update` - steamcmd update for app 427520.

### REST API wrappers (`coily ops`)

- `coily ops modio` - mod.io v1 API for Eco mods.
- `coily ops discord` - Discord HTTP API with bot auth (generated from OpenAPI).
- `coily ops sentry` - Sentry Public API, read-only.
- `coily ops trello` - Trello REST API with key plus token (generated from OpenAPI).
- `coily ops forgejo` - Forgejo admin API (user list, doctor checks).

### Repo-defined commands (`coily exec`)

- `coily exec <cmd> [-- extra-args]` - Per-repo commands loaded from `.coily/coily.yaml`.
- Discovery walks cwd up to filesystem root, override with `$COILY_REPO_CONFIG`.
- Gated on clean and synced tree (no uncommitted, no untracked, branch has upstream, not behind).
- `--audit-override-dirty` bypasses the gate, tags audit row `audit_override: true`, snapshots porcelain status.
- Audit verb prefix `repo.<cmd>`.

## Audit and logging

- `coily audit path` - Print resolved audit log path.
- `coily audit tail [--follow] [--since <duration|unix-seconds>]` - Stream audit JSONL.
- `coily audit finding` - Parse and display findings and anomalies.
- `coily git trailer` - Emit `Audit-log:` trailers for git commits.
- `coily git trailer-hook` - prepare-commit-msg hook integration.
- `coily git audit-show` - Resolve `Audit-log:` trailer back to its record.
- Append-only JSONL at `~/.local/state/coily/audit.jsonl` by default.
- Per-host log files, host captured in row but not indexed on.
- Row schema includes timestamp, argv, decision, exit code, scope, verb, repo binding.
- Rotation by max size, backups, and age.
- Exit code classification (upstream_failed, policy_rejected, generic, etc.).
- Structured YAML error envelopes on stderr for programmatic consumers.

## Security and lockdown

- `coily lockdown --recursive [--path <dir>]` - Baseline `.claude/settings.json` rules across a workspace.
- `coily lockdown --apply [--replace]` - Apply or update one repo's lockdown.
- `coily lockdown skill [--format markdown|yaml]` - Regenerate the coily-passthroughs skill from live verb tree.
- Shell metacharacter validator rejects `$`, backticks, `;`, `&&`, `||`, `|`, `>`, `<`, `$(`, `${`, `\`.
- `pkg/policy.ValidateArg` runs on every string argument.
- `SkipPolicy` for tools that bypass shell (aws, gh, tailscale).
- `SkipScope` for meta-verbs (lockdown, setup, version).
- Lockdown defaults embedded in the binary, allowlist plus denylist baked at build.
- PreToolUse hook at `~/.claude/coily-binary-gate.sh` blocks non-homebrew coily binaries.

## Configuration

- Three layers, lowest to highest precedence: Go literal defaults, `~/.coily/config.yaml`, `./.coily/config.yaml`.
- `~/.coily/coily.yaml` repo allowlist, walked from cwd to root.
- Sections: `kai_server` (tailscale_host, ssh_user, ssh_key_path), `audit` (log_path, max_size, max_backups, max_age, compress), `aws` (profile), `eco` (configs_dir, server_dir), `factorio` (server_dir).
- Env overrides: `$COILY_AUDIT_LOG`, `$COILY_COMMIT_SCOPE`, `$COILY_REPO_CONFIG`, `$COILY_CACHE_DIR`.

## Secrets and auth

- AWS credentials from `~/.aws/config` plus `$AWS_PROFILE`.
- kubectl credentials from `~/.kube/config`.
- gh credentials from `~/.config/gh/hosts.yml`.
- Discord bot token from SSM.
- Sentry org slug plus auth token from config and SSM.
- Trello key plus token from SSM or config.
- mod.io API key from SSM.

## Distribution and release

- No prebuilt binaries. Every install is a source build.
- Push to `main` triggers `.github/workflows/release.yml`.
- Semver auto-bump (patch default, minor on `feat:`, major on `feat!:`).
- Auto-tag plus GitHub Release.
- Homebrew tap formula update via direct push to coilysiren/homebrew-tap.
- `make install` on darwin-arm64 to `/usr/local/bin/coily` (root-owned).
- `make install-windows` to `C:\Program Files\coily\coily.exe` (admin write only).
- `make deploy-server` cross-compiles for linux-arm64 and scp-installs to kai-server.
- `make dev` builds `./bin/coily-dev` (off-PATH, distinct name).
- Bootstrap `brew install coilysiren/tap/coily`.

## Cross-cutting infrastructure

- `pkg/shell` - Capture, Run, Stream subprocess abstraction.
- `pkg/gittree` - Clean, dirty, ahead, behind, upstream detection.
- `pkg/ttlcache` - Memoized cwd-to-toplevel resolution with 5-minute TTL.
- `pkg/ssh` - golang.org/x/crypto/ssh client, host key verification against `~/.ssh/known_hosts`, ssh-agent or PEM auth, SFTP support.
- Per-binary egress allowlisting (brew has entries, others pending).

## Testing and dev

- `make test` - Unit tests via `gotest`, raw `go test` in CI.
- `make vet` - go vet.
- `make build` / `make dev` - Prod and dev tags.
- `make clean` - Remove build outputs.
- Security claims test verifies prose against runtime (`cmd/coily/security_claims_test.go`).
- Coverage on passthroughs, policy, audit format, git tree state, scope resolution, repo config loading.
- Pre-commit hook integration via the pre-commit framework.

## Documentation surface

- `README.md` - Architecture overview, threat model summary, install paths, repo commands.
- `AGENTS.md` - Git workflow, fix-coily-first discipline, brew pipeline, release framework.
- `SECURITY.md` - Threat model, allowlist inversion, design guardrails, open questions, anti-signals.
- `docs/unresolved.md` - Known issues, incomplete features, next priorities.
- `docs/FEATURES.md` - This file. Baseline scope inventory.
- `coily --list`, `coily --tree`, `coily <verb> --help` - Runtime discovery.

## Known limitations

- No `coily self-update` in v1. v2 will add adversarial-reviewed CI plus binary signing.
- Upstream tool binaries resolved via `$PATH`, not pinned by SHA256.
- Claude Desktop on Windows does not enforce Bash deny list. Lockdown is CLI-only enforcement.
- Completion scripts use standard urfave/cli v3 patterns, not end-to-end tested.
- No confirmation tokens. Removed 2026-04-24 because they provided false security with no real fence.

## See also

- [README.md](../README.md) - human-facing intro.
- [AGENTS.md](../AGENTS.md) - agent-facing operating rules.
- [.coily/coily.yaml](../.coily/coily.yaml) - allowlisted commands.

Cross-reference convention from [coilysiren/agentic-os-kai#313](https://github.com/coilysiren/agentic-os-kai/issues/313).

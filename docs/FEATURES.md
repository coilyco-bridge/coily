# Coily features

Baseline of what coily ships. Update bullets in the same commit as the change so this file mirrors the binary.

Coily is a single-binary CLI security boundary. It wraps privileged ops (aws, gh, kubectl, ssh, docker, tailscale, package managers, game-server systemd) in named verbs, validates argv, writes a JSONL audit row per invocation. No `coily shell` / `coily run` / `coily ssh exec` escape hatch.

## Architecture

- Single Go binary, one entry in the Claude Code allowlist.
- Trust inversion: narrow allowlist of named verbs, not broad denylist.
- Escape-hatch resistant: no `shell`, `run`, free-form `exec`.
- Argv validation: every string arg checked for shell metacharacters at load + invocation.
- Scope binding: every audit row bound to a git toplevel (`--commit-scope` or `$COILY_COMMIT_SCOPE`). No opt-out.
- `Verb.Spec` abstraction: uniform validation/action/audit pipeline per command.

## Verb surface

**Built-in top level**: `coily whoami`, `coily version`, `coily --list` / `--tree`, `coily setup`, `coily install-completion`.

**CLI passthroughs**: `coily ops aws|gh|kubectl`, `coily docker`, `coily tailscale`. kubectl has readonly/write gating via lockdown.

**Package managers**: `coily pkg pnpm|npm|yarn|bun|uv|pip|pipx|poetry|cargo|gem|bundle|brew|glama|skillsmp`. `coily brew install|uninstall|upgrade|reinstall` is a separate top-level scoped to `coilysiren/tap/*` by default.

**SSH**: `coily ssh <alias> -- coily <argv>`. Free-form passthrough; remote coily's lockdown is the security boundary. Audit rows chain across hosts via `--audit-parent`. See [coily#187](https://github.com/coilysiren/coily/issues/187).

**Game-server ops**: `coily gaming {eco,core-keeper,icarus,factorio}` (status/tail/start/stop/restart common to all). Eco adds `world {get-seed,set-seed,randomize,snapshot}` + `mod {list,push}`. Factorio adds `update` (steamcmd), `saves`, `mods`, `players`.

**REST API wrappers**: `coily ops {modio,discord,sentry,trello,forgejo}`.

**Repo-defined**: `coily exec <cmd> [-- extra-args]`. Loaded from `.coily/coily.yaml`. Gated on clean+synced tree; `--audit-override-dirty` bypasses with audit tag. Verb prefix `repo.<cmd>`.

## Audit and logging

- `coily audit {path,tail}` + `coily git {trailer,trailer-hook,audit-show}`.
- Append-only per-scope JSONL under `~/.coily/audit/`. Schema in [docs/audit.md](audit.md): argv, decision, exit code, scope, verb, `profile_decision` (static labels), `egress[]` (runtime).
- Rotation by size, backups, age. Exit-code classification (upstream_failed / policy_rejected / generic).
- Structured YAML error envelopes on stderr.

## Security and lockdown

- `coily lockdown {--recursive,--apply,--replace,skill}`. Baselines `.claude/settings.json` across a workspace.
- Metacharacter validator rejects `$`, backticks, `;`, `&&`, `||`, `|`, `>`, `<`, `$(`, `${`, `\`.
- `policy.ValidateArg` on every string arg. `SkipPolicy` for SDK-routed tools; `SkipScope` for meta-verbs.
- Lockdown defaults embedded at build. PreToolUse hook at `~/.claude/coily-binary-gate.sh` blocks non-homebrew coily binaries.

## Configuration + secrets

Three-layer precedence: Go defaults < `~/.coily/config.yaml` < `./.coily/config.yaml`. Sections: `kai_server`, `audit`, `aws`, `eco`, `factorio`. Env: `$COILY_AUDIT_LOG`, `$COILY_COMMIT_SCOPE`, `$COILY_REPO_CONFIG`, `$COILY_CACHE_DIR`. AWS / kubectl / gh creds from their canonical files; Discord / Sentry / Trello / mod.io from SSM.

## Distribution

No prebuilt binaries. Push to `main` triggers semver bump + tag + GH Release + in-repo `Formula/coily.rb` `url` rewrite via Contents API. `make install` (root-owned `/usr/local/bin`), `make install-windows` (admin `C:\Program Files\coily\`), `make deploy-server`. Bootstrap: `brew tap coilysiren/coily https://github.com/coilysiren/coily; brew install coilysiren/coily/coily`.

## Cross-cutting infrastructure

All cross-cutting infrastructure lives in the sibling `cli-guard` module: `cli-guard/shell` (subprocess), `cli-guard/gittree` (clean/dirty/ahead/behind), `cli-guard/ttlcache` (cwd-to-toplevel memo), `cli-guard/ssh` (`golang.org/x/crypto/ssh` + known_hosts + ssh-agent + SFTP). coily itself has no `pkg/` tree.

## Testing + docs

`make test` (gotest) / `vet` / `build` / `dev` / `clean`. Security claims test verifies prose against runtime. Docs: README, AGENTS, SECURITY, docs/unresolved.md, and `coily --list` / `--tree` / `<verb> --help`.

## Known limitations

No `coily self-update` in v1. Upstream binaries resolved via `$PATH`, not SHA256-pinned. Claude Desktop on Windows doesn't enforce Bash deny list. No confirmation tokens (removed 2026-04-24: false security).

## See also

- [README.md](../README.md) - human-facing intro.
- [AGENTS.md](../AGENTS.md) - agent-facing operating rules.
- [.coily/coily.yaml](../.coily/coily.yaml) - allowlisted commands.

Cross-reference convention from [coilysiren/agentic-os#59](https://github.com/coilysiren/agentic-os/issues/59).

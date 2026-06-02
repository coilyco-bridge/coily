# Coily features

Baseline of what coily ships. Update in the same commit as the change so this file mirrors the binary.

Coily is a single-binary CLI security boundary. It wraps privileged ops (aws, gh, kubectl, ssh, docker, tailscale, package managers, game-server systemd) in named verbs, validates argv, and writes a JSONL audit row per invocation. No escape hatch.

## Architecture

- Single Go binary, one entry in the Claude Code allowlist.
- Trust inversion: narrow allowlist of named verbs, not broad denylist.
- Escape-hatch resistant: no `shell`, `run`, free-form `exec`.
- Argv validation: every string arg checked for shell metacharacters at load + invocation.
- RepoRoot stamping: every audit row records cwd's git toplevel (empty outside any repo), best-effort and forensic. `coily git audit-show --scope <repo>` filters by it.
- `Verb.Spec` abstraction: uniform validation/action/audit pipeline per command.

## Verb surface

**Built-in top level**: `coily whoami`, `coily version`, `coily --list` / `--tree`, `coily setup`, `coily install-completion`.

**CLI passthroughs**: `coily ops aws|gh|kubectl`, `coily docker`, `coily tailscale`. kubectl has readonly/write gating via lockdown.

**Package managers**: `coily pkg {pnpm,npm,yarn,bun,uv,pip,pipx,poetry,cargo,gem,bundle,brew,glama,skillsmp}`. `coily brew {install,uninstall,upgrade,reinstall}` is a separate top-level scoped to `coilysiren/tap/*`.

**SSH**: `coily ssh <alias> -- coily <argv>`. Free-form passthrough; remote coily's lockdown is the security boundary. Audit rows chain across hosts via `--audit-parent`. See [coily#187](https://github.com/coilysiren/coily/issues/187).

**Session**: `coily session {use,show,clear,end}`. Per-session lockdown-profile sentinel. `end` self-terminates a finished sidequest, SIGTERM to claude ([coily#309](https://github.com/coilysiren/coily/issues/309)).

**Game-server ops**: `coily gaming {eco,core-keeper,icarus,factorio}` (status/tail/start/stop/restart common). Eco adds `world` + `mod` subverbs; Factorio adds `update`, `saves`, `mods`, `players`.

**REST API wrappers**: `coily ops {modio,discord,sentry,trello,forgejo}`.

**Agent Channel**: `coily channel {create,post,read,state,spec,events,close}` wraps the v2 protocol from coilysiren/backend ([coily#330](https://github.com/coilysiren/coily/issues/330)).

**Repo-defined**: `coily exec <cmd> [-- extra-args]`. Loaded from `.coily/coily.yaml`. Gated on clean+synced tree; `--audit-override-dirty` bypasses with audit tag. Verb prefix `repo.<cmd>`.

## Audit and logging

- `coily audit {path,tail}` + `coily git audit-show`.
- Append-only per-scope JSONL under `~/.coily/audit/`. Schema in [docs/audit.md](audit.md): argv, decision, exit code, scope, verb, `profile_decision` (static labels), `egress[]` (runtime).
- Rotation by size, backups, age. Exit-code classification (upstream_failed / policy_rejected / generic).
- Structured YAML error envelopes on stderr.

## Security and lockdown

- `coily lockdown {--recursive,--apply,--replace,skill}`. Baselines `.claude/settings.json` across a workspace.
- Metacharacter validator rejects `$`, backticks, `;`, `&&`, `||`, `|`, `>`, `<`, `$(`, `${`, `\`.
- `policy.ValidateArg` on every string arg. `SkipPolicy` for SDK-routed tools; `SkipScope` for meta-verbs.
- Lockdown defaults embedded at build. PreToolUse hook at `~/.claude/coily-binary-gate.sh` blocks non-homebrew coily binaries.

## Configuration + secrets

Three-layer precedence: Go defaults < `~/.coily/config.yaml` < `./.coily/config.yaml`. Sections: `kai_server`, `audit`, `aws`, `eco`, `factorio`, `channel`. Env: `$COILY_AUDIT_LOG`, `$COILY_REPO_CONFIG`, `$COILY_CACHE_DIR`. AWS / kubectl / gh creds from canonical files, the REST APIs from SSM.

## Distribution

No prebuilt binaries. Push to `main` triggers semver bump + tag + GH Release + `Formula/coily.rb` rewrite. `make install`, `make install-windows`, `make deploy-server`. Bootstrap via `brew tap coilysiren/coily`.

## Cross-cutting infrastructure

All cross-cutting infrastructure lives in the sibling `cli-guard` module: `cli-guard/shell` (subprocess), `cli-guard/gittree` (git state), `cli-guard/ttlcache` (cwd-to-toplevel memo), `cli-guard/ssh` (x/crypto/ssh + known_hosts + ssh-agent + SFTP). coily has no `pkg/`.

## Testing + docs

`make test` / `vet` / `build` / `dev` / `clean`. Security claims test verifies prose against runtime. Docs: README, AGENTS, SECURITY, docs/unresolved.md, and `coily --list` / `--tree` / `<verb> --help`.

## Known limitations

No `coily self-update` in v1. Upstream binaries resolved via `$PATH`, not SHA256-pinned. Claude Desktop on Windows doesn't enforce Bash deny list.

## See also

- [README.md](../README.md) - human-facing intro.
- [AGENTS.md](../AGENTS.md) - agent-facing operating rules.
- [.coily/coily.yaml](../.coily/coily.yaml) - allowlisted commands.

Cross-reference convention from [coilysiren/agentic-os#59](https://github.com/coilysiren/agentic-os/issues/59).

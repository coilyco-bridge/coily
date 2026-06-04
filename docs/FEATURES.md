# Coily features

Baseline of what coily ships. Update in the same commit as the change so this file mirrors the binary.

Coily is a single-binary CLI security boundary. It wraps privileged ops (aws, gh, kubectl, docker, tailscale, package managers, game-server systemd) in named verbs, validates argv, and writes a JSONL audit row per invocation. No escape hatch.

## Architecture

- Single Go binary, one entry in the Claude Code allowlist.
- Trust inversion: narrow allowlist of named verbs, not broad denylist.
- Escape-hatch resistant: no `shell`, `run`, free-form `exec`.
- Argv validation: every string arg checked for shell metacharacters at load + invocation.
- RepoRoot stamping: every audit row records cwd's git toplevel (empty outside any repo), best-effort and forensic. `coily git audit-show --scope <repo>` filters by it.
- `Verb.Spec` abstraction: uniform validation/action/audit pipeline per command.

## Verb surface

**Built-in top level**: `coily whoami`, `coily version`, `coily --list` / `--tree`, `coily setup`, `coily install-completion`.

**CLI passthroughs**: `coily ops aws|gh|kubectl`, `coily docker`, `coily tailscale`. kubectl has readonly/write gating via lockdown. `ops aws` also runs a read-only argv gate ([coily#54](https://forgejo.coilysiren.me/coilyco-bridge/coily/issues/54)): read-only verbs (`s3 ls`, `describe-*`, `list-*`, ...) touching a sensitive resource pattern (secrets / state / backup buckets, admin role ARNs) are denied pre-send, default-deny with explicit allow via `aws.allow_sensitive_reads` config or the `COILY_AWS_ALLOW_SENSITIVE_READ` env var. Denied and allowed reads each land their own audit row (`ops.aws.read.denied` / `ops.aws.read.allowed`).

**git**: `coily git {status,log,diff,show,add,fetch,pull,push,branch,checkout,stash,restore}` are audited passthroughs. `coily git commit` is not - it is a dedicated, concurrency-safe verb ([coily#7](https://forgejo.coilysiren.me/coilyco-bridge/coily/issues/7)): it requires `coily git commit -m "msg" -- <path>...`, commits the named paths from the worktree against a private `GIT_INDEX_FILE` seeded from HEAD, and forbids the editor - so two sessions sharing one working tree cannot cross commit content or messages through the shared index / `COMMIT_EDITMSG`.

**Package managers**: `coily pkg {pnpm,npm,yarn,bun,uv,pip,pipx,poetry,cargo,gem,bundle,brew,glama,skillsmp}`. `coily brew {install,uninstall,upgrade,reinstall}` is a separate top-level scoped to the `coilysiren/<repo>/<formula>` per-repo taps.

**Session**: `coily session {use,show,clear,end}`. Per-session lockdown-profile sentinel. `end` self-terminates a finished sidequest, SIGTERM to claude ([coily#309](https://github.com/coilysiren/coily/issues/309)).

**Game-server ops**: `coily gaming {eco,core-keeper,icarus,factorio}` (status/tail/start/stop/restart common). Eco adds `world` + `mod` subverbs; Factorio adds `update`, `saves`, `mods`, `players`.

**REST API wrappers**: `coily ops {modio,discord,sentry,trello,forgejo}`.

**Multi-org owner resolution**: post org-split the fleet spans `coilysiren`, `coilyco-bridge`, and `coilyco-flight-deck` (config `primary_orgs`, [coily#162](https://forgejo.coilysiren.me/coilyco-bridge/coily/issues/162)). Dispatch trusts any ref in that set, and forgejo verbs accept either the historical `coilysiren` alias or the canonical owner: a mutating call that 301s from an alias owner is transparently re-issued (method + body preserved) against the canonical URL the redirect names, instead of silently no-opping.

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

Three-layer precedence: Go defaults < `~/.coily/config.yaml` < `./.coily/config.yaml`. Sections: `kai_server`, `audit`, `aws`, `eco`, `factorio`, `forgejo`. Env: `$COILY_AUDIT_LOG`, `$COILY_REPO_CONFIG`, `$COILY_CACHE_DIR`. AWS / kubectl / gh creds from canonical files, the REST APIs from SSM.

## Distribution

No prebuilt binaries. Push to `main` triggers semver bump + tag + GH Release + `Formula/coily.rb` rewrite. `make install`, `make install-windows`, `make deploy-server`. Bootstrap via `brew tap coilysiren/coily`.

## Cross-cutting infrastructure

All cross-cutting infrastructure lives in the sibling `cli-guard` module: `cli-guard/shell` (subprocess), `cli-guard/gittree` (git state), `cli-guard/ttlcache` (cwd-to-toplevel memo). coily has no `pkg/`.

## Testing + docs

`make test` / `vet` / `build` / `dev` / `clean`. Security claims test verifies prose against runtime. Docs: README, AGENTS, SECURITY, docs/unresolved.md, and `coily --list` / `--tree` / `<verb> --help`.

## Known limitations

No `coily self-update` in v1. Upstream binaries resolved via `$PATH`, not SHA256-pinned. Claude Desktop on Windows doesn't enforce Bash deny list.

## See also

- [README.md](../README.md) - human-facing intro.
- [AGENTS.md](../AGENTS.md) - agent-facing operating rules.
- [.coily/coily.yaml](../.coily/coily.yaml) - allowlisted commands.

Cross-reference convention from [coilysiren/agentic-os#59](https://github.com/coilysiren/agentic-os/issues/59).

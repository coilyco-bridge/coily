# Repo config schema (`.coily/coily.yaml`)

Per-repo command allowlist loaded by `pkg/repocfg` when coily is invoked anywhere inside a checkout. Each entry becomes a top-level verb invokable as `coily exec <name>`. The schema is intentionally tiny so the security boundary stays auditable in a single read of the loader.

This doc is the canonical reference for what may appear in a `.coily/coily.yaml`. It pairs with `pkg/repocfg/repocfg.go` (loader) and `SECURITY.md` (security story).

## Allowed shape

```yaml
commands:
  <name>:
    run: <command line>
    description: <free-form string, optional>
  <name>: <command line>   # shorthand for {run: ...}
```

That is the entire schema. Two top-level forms (`commands` only), two command-form variants (string scalar, or `{run, description}` mapping), three command keys total (`run`, `description`, plus the implicit map key as the verb name).

## Top-level keys

* `commands` - **required** - map of `<name> -> <command-spec>`. The only allowed top-level key. Empty (`commands: {}`) is valid for repos that exist only so the lockdown discipline applies uniformly across `coilysiren/*` (e.g., `homebrew-tap`, `coilysiren`).

Anything else at the top level is currently ignored silently by `yaml.v3`. Schema enforcement that rejects unknown keys is tracked in [#105](https://github.com/coilysiren/coily/issues/105).

## Command names (map keys)

Validated by `validateName` in `repocfg.go`:

* Must be non-empty.
* Allowed characters: `a-z`, `0-9`, `-`.
* Cannot start or end with `-`.
* No uppercase, underscores, dots, slashes, or unicode.

Names collide-check against built-in coily verbs at registration time (`coily exec` namespacing prevents shadowing built-ins).

## Command spec

A command value may be either a string scalar or a mapping.

### String form

```yaml
commands:
  test: go test ./...
```

The whole string is the `run` line. No description. Used by `coily/.coily/coily.yaml` for the simplest cases.

### Mapping form

```yaml
commands:
  test:
    run: go test ./...
    description: Run the unit test suite.
```

Allowed keys, both decoded through a typed struct:

* `run` - **required** - the command line. Parsed via `strings.Fields` (whitespace-split, no shell parsing). Empty `run` rejects the load.
* `description` - **optional** - free-form string shown in `coily --list` and help output.

Anything else inside the mapping is currently ignored silently by `yaml.v3`. See [#105](https://github.com/coilysiren/coily/issues/105).

## Argv validation

The `run` line is split by whitespace, never by a shell. Every resulting token (and every user-supplied extra appended at invocation time) is checked against `policy.ValidateArg`, which rejects shell metacharacters: `$ ; & | < > ( ) { } \` plus newline, carriage return, tab. Pipes, redirects, and `$(subshells)` fail at load time, not at invocation.

If a repo command needs a shell pipeline, the answer is a wrapper script committed under the repo, not an escape hatch in the schema. See `agentic-os-kai/.coily/coily.yaml` for the pattern: a `daily-*-auth` entry runs `bash scripts/refresh-daily-auth.sh <name>` rather than encoding the shell logic inline.

## Discovery

* `Filename` = `coily.yaml`
* `LocalDirName` = `.coily`
* Canonical location is `<repo>/.coily/coily.yaml`. The legacy `<repo>/coily.yaml` form is rejected with `ErrLegacyLocation` pointing at the new home.
* `Discover` walks up from the cwd looking for `.coily/coily.yaml`.
* `DiscoverChildren` (used by `coily exec` when no ancestor config exists) scans direct child directories one level down.
* `$COILY_REPO_CONFIG` overrides discovery with an absolute path. Test-only escape hatch.

## Survey results (2026-05-08)

Walked all `.coily/coily.yaml` files under `~/projects/coilysiren/`:

```
backend, coily, agentic-os-kai, coilysiren, eco-cycle-prep, eco-jobs-tracker,
eco-mcp-app, eco-mods-public, eco-telemetry, factorio-mods, galaxy-gen,
gauntlet, homebrew-tap, infrastructure, message-ops, repo-recall,
session-lattice, sirens-discord-ops, website
```

19 files total. **No drift observed:**

* Every file uses only `commands` at the top level.
* Every command entry uses only `run` and `description` (or the string-scalar shorthand).
* Every command name conforms to `[a-z0-9-]`.
* Two repos (`coilysiren`) have empty `commands: {}` by design.

The de facto schema and the loader's accepted schema agree. No legacy keys, no typos, no aspirational fields.

## Audit trail asymmetry

`coily exec <cmd>` writes a `repo.<cmd>` audit row with argv, exit code, commit scope, and working-tree status. By default the row has no `egress` field; built-in passthrough verbs (`ops.aws`, `ops.gh`, `pkg.uv`, etc.) wrap their underlying binary in a CONNECT proxy and record every outbound HTTPS host in the row's `egress` array. A plain `coily exec` spawns the declared `run` line directly with no proxy attached, so its outbound traffic is unobservable from the audit trail.

For repo commands that call LLM APIs, cloud SDKs not under `ops.aws`, or tunneled services, this is a forensics hole. A `repo.<cmd>` row that ran 150 outbound HTTPS calls looks identical in the audit log to one that ran zero.

Opt in per command with `audit.egress: true` in the mapping form:

```yaml
commands:
  play:
    run: uv run python bot.py
    description: Workshop battleships bot
    audit:
      egress: true
```

The opt-in starts the same per-invocation CONNECT proxy that passthrough verbs use, injects `HTTPS_PROXY` / `HTTP_PROXY` into the child's environment for the duration of that one call, and stamps the collected rows onto the audit record. Mode is observe (every host forwarded and logged; no allowlist). Works only for children that honor `HTTPS_PROXY`, so `docker` / `tailscale` / arbitrary tunneled protocols stay invisible; the field is best-effort by design.

Default stays off so the historical "argv + exit code only" shape survives for commands that genuinely don't care. Read "no `egress` field" on an opted-out command as "not observed", not "no traffic". See [coilysiren/coily#281](https://github.com/coilysiren/coily/issues/281) and [coilysiren/cli-guard#82](https://github.com/coilysiren/cli-guard/issues/82).

## Out of scope for this doc

* Runtime enforcement of unknown-key rejection. Tracked separately in [#105](https://github.com/coilysiren/coily/issues/105).
* Schema versioning. The schema is small enough that a flag day works if it ever needs to grow.

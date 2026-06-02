# Audit log format

Every coily invocation appends one JSONL row to `~/.coily/audit/<owner>-<repo>.jsonl` (or `~/.coily/audit/_unrooted.jsonl` when no commit scope can be resolved). One row per process, written after the wrapped command exits.

This doc is the canonical reference for the row schema. It pairs with the `audit` package in `coilysiren/cli-guard` (writer) and `SECURITY.md` (why the file exists).

## Row shape

```json
{
  "id": "019e45ec-e901-7563-9a93-4ddd1fc76ba5",
  "ts": 1779289614,
  "decision": "accept",
  "verb": "ops.gh",
  "argv": ["coily", "ops", "gh", "issue", "view", "..."],
  "exit_code": 0,
  "duration_ms": 1090,
  "repo_root": "/Users/kai/projects/coilyco-flight-deck/ward",
  "cwd_subprocess": "/Users/kai/projects/coilyco-flight-deck/ward",
  "cwd_at_invocation": "/Users/kai/projects/coilyco-flight-deck/ward",
  "commit_scope": "/Users/kai/projects/coilyco-flight-deck/ward",
  "profile_decision": { ... },
  "egress": [ ... ]
}
```

Field reference below.

## Always present

* `id` - UUIDv7. Sortable by creation time.
* `ts` - Unix epoch seconds at row write.
* `decision` - `accept` | `deny`. Gate outcome before any wrapped command runs.
* `verb` - Canonical verb name (e.g. `ops.gh`, `dispatch.interactive`, `repo.<cmd>` for `coily exec` user verbs).
* `argv` - Argv as coily received it. Validated against the verb's allowlist before any subprocess starts.
* `exit_code` - Wrapped command's exit code, or coily's own exit code if the gate denied.
* `repo_root` - Git toplevel resolved at invocation.
* `commit_scope` - The scope the audit row is bound to (`--commit-scope` or `$COILY_COMMIT_SCOPE`). Determines which `<owner>-<repo>.jsonl` file the row lands in.

## Sometimes present

* `duration_ms` - Wall time the wrapped command ran. Absent for deny decisions (no subprocess) and for verbs that exit before timing starts.
* `cwd_subprocess` - cwd handed to the wrapped command.
* `cwd_at_invocation` - cwd of the coily process itself. Distinct from `cwd_subprocess` when coily rehomes for `--commit-scope`.
* `profile_decision` - See below. Present when a session profile evaluation ran for this verb.
* `egress` - See below. Present when the verb is one of coily's built-in network-aware wrappers (e.g. `ops.gh`, `ops.aws`) that records outbound HTTPS connections.

## `profile_decision` is a static label, not a runtime observation

This is the most common source of misreading. Read it carefully.

```json
"profile_decision": {
  "allowed": true,
  "source": "unset",
  "coordinate": {
    "data_security": "max",
    "blast_radius": "low",
    "network_egress": "air-gapped",
    "filesystem_reach": "repo-only"
  },
  "reason": "no profile selected for this session"
}
```

* `allowed` - Whether the active session profile (or the default when none is selected) permits the verb shape to run. The gate writes this before the subprocess starts.
* `source` - How the profile was resolved: `override` (explicit `coily session use`), `unset` (no sentinel), `missing_file`, etc.
* `coordinate` - The axes the **active profile asserts about the verb shape**. These are declared labels from `profiles.Resolve`, not measurements of what the process did.
* `reason` - Human-readable note tied to `source`.

### `coordinate.network_egress` does not mean "this row made / did not make network calls"

The four `coordinate.*` axes (`data_security`, `blast_radius`, `network_egress`, `filesystem_reach`) come from the static profile definition. They describe what a future gate evaluation **would label** a verb of this shape under the active profile. They are not derived from what the process actually did.

The most common trap: `coordinate.network_egress: "air-gapped"` on a row where the wrapped command made hundreds of outbound HTTPS calls. Both are technically correct under the model. The profile labels the verb shape `air-gapped`, the process opened sockets anyway. The label was never a measurement.

### Where to look for runtime egress

The `egress` array, when present, is the runtime observation:

```json
"egress": [
  {"host": "api.github.com", "decision": "allow", "bytes_up": 4053, "bytes_down": 7657, "duration_ms": 1054}
]
```

* Built-in network-aware verbs populate `egress` from their own HTTPS-recording shim.
* User-defined `coily exec <cmd>` verbs do **not** currently populate `egress`. The wrapped subprocess runs outside the shim. Absence of `egress` on a `repo.*` row is "not observed", not "no egress occurred."

When reading an audit row to answer "what did this verb touch on the network", trust `egress[]`. Treat `coordinate.network_egress` as the static policy label that the row was evaluated against, not as a description of what happened.

## File layout

* `~/.coily/audit/<owner>-<repo>.jsonl` - Per-commit-scope log. One file per repo, one row per invocation, append-only.
* `~/.coily/audit/_unrooted.jsonl` - Catch-all for invocations with no resolvable commit scope.
* `~/.coily/audit/sessions/<session_id>/profile` - Session-profile sentinel file. Not a log row, named here so the layout is documented in one place.

The path is owned by `pkg/config` (`SessionProfilePath`, audit dir defaults). Override via `$COILY_AUDIT_LOG` (full path) for tests and ephemeral hosts.

## Reading the log

* One row per line, NDJSON. `jq -c` over the relevant `<owner>-<repo>.jsonl` is the canonical reader.
* `coily audit path` prints the file path for the current commit scope.
* `coily git audit-show --scope <repo> --since <unix-seconds>` prints the rows bound to a repo over a time window as yaml.

## See also

* [SECURITY.md](../SECURITY.md) - why audit rows exist and what they prove.
* [docs/FEATURES.md](FEATURES.md) - inventory of verbs that produce these rows.
* [docs/repocfg-schema.md](repocfg-schema.md) - schema for `coily exec` user verbs (`repo.*` rows).
* [coily#282](https://github.com/coilysiren/coily/issues/282) - the static-label-vs-runtime distinction this doc encodes.

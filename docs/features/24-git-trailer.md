# 24. `coily git trailer` and the prepare-commit-msg hook

**What it is**: Append `Audit-log:` trailers to every commit so the audit log is cross-referenced from the git history.

**Trailer shape**: `Audit-log: coily://<unix-ts>/<base32-short-id>`. The unix timestamp is the audit row's `ts`, the short ID is the first 8 chars of base32-encoded raw UUID v7 bytes from the row's `id`. URL-shaped so future tooling can render it as a link.

**Three new subcommands**:

- `coily git trailer` - emit `Audit-log:` lines for the current repo. Filter is `commit_scope == <repo toplevel>` AND `ts >= <since>`. Output goes to stdout, one trailer per audit row, capped at 20 by default. If zero rows match, emits `Audit-log: none` so consumers can distinguish "explicitly nothing happened" from "tool failed."
- `coily git audit-show <coily://...>` - resolve a trailer back to its full audit record, printed as yaml. Pass `--scope=<path> --since=<ts>` instead of a trailer to re-run the same query a truncated trailer summary refers to.
- `coily git trailer-hook <commit-msg-file> [<source>]` - prepare-commit-msg implementation. Skips merge/squash, otherwise rewrites the commit message file in place with trailers appended. Exits non-zero on any error so the commit is blocked.

**Audit row binding**: every audit row carries a `commit_scope` field set from the `--commit-scope` global flag (or `$COILY_COMMIT_SCOPE`). Default is `auto`, which resolves to `git rev-parse --show-toplevel` of cwd at invocation time. If cwd is not inside a git repo, `auto` is an error - operators are forced to either pass `--commit-scope=<path>` explicitly or `--commit-scope=-` to opt out of trailer binding entirely. This is the load-bearing provenance contract: a row without a `commit_scope` will not appear in any commit's trailer.

Read-only verbs (version, whoami, audit, lockdown, setup, install-completion, the git subcommands themselves) set `SkipScope: true` on their `verb.Spec` so they don't fail when run from outside a repo.

**Repo opt-in via pre-commit framework**:

```yaml
- repo: local
  hooks:
    - id: coily-trailer
      name: coily audit-log trailer
      entry: coily git trailer-hook
      language: system
      stages: [prepare-commit-msg]
      always_run: true
      pass_filenames: false
      require_serial: true
```

`coily git trailer-hook` exits non-zero when it can't resolve the repo, can't read the audit log, or can't write the message file. That blocks the commit, matching the design choice to make the trailer presence enforceable at the repo level.

**Window selection**: `coily git trailer` defaults `--since` to HEAD's committer timestamp (`git log -1 --format=%ct HEAD`). On a fresh branch with no HEAD commit, falls back to the last hour.

**Truncation**: when more than `--max` rows match, the most recent N rows are emitted as `Audit-log:` trailers and a summary line at the end records the truncation: `Audit-log: <k> earlier rows truncated; run \`coily git audit-show --since=<ts> --scope=<repo>\` for full list`.

**Why blocking, not soft-warn**: the chain-of-trust story only holds if every commit either has the trailer or is explicitly opting out. Soft-warn would mean half the commits silently lose provenance, which defeats the purpose of having the trailer at all.

**Why no GitHub Actions step**: deliberately out of scope. The hook fires locally, every commit landing on `main` already went through it. CI validation of the trailer presence would be belt-and-suspenders without adding signal.

**Cryptographic integrity is out of scope.** The audit row itself is unsigned. The `Signed-off-by` + GPG/SSH commit signature on the commit cover commit-level integrity; signing the audit row is a separate (cosign-shaped) lift filed in the issue tracker.

**Cross-machine audit aggregation is out of scope.** Each host's audit DB is local. A trailer authored on the Windows host references a row that doesn't exist in the Mac's audit log, and vice versa. Future kai-server sync is a separate issue.

**Test prompt**:
> Bring up a fresh git repo, run `coily lockdown --apply` from inside it, then run `coily git trailer`. Verify the output contains exactly one `Audit-log:` line. Run `coily git audit-show <trailer>` against that output and verify it produces a yaml block whose `record.verb` is `lockdown`. From outside any git repo, verify `coily lockdown` fails with `scope_unresolved` and a hint pointing at `--commit-scope`.

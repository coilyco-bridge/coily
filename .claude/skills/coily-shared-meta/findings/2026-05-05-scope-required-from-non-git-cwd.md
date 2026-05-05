---
date: 2026-05-05
slug: scope-required-from-non-git-cwd
promoted_to:
  - anti-signal: "scope auto-detect is transparent to the operator"
  - issue: https://github.com/coilysiren/coily/issues/59
---

# 2026-05-05 - many verb invocations fail with `scope: cwd is not inside a git repo` from non-tracked CWDs

## What was observed

Across the 35-day audit window, `_unrooted.jsonl` accumulated 485 rows. The dominant failure pattern across multiple verbs is `error: "scope: cwd is not inside a git repo; pass --commit-scope=<repo-path> explicitly"`. Affected verbs include:

- `core-keeper.stop` 4/4 fail (all scope-rejected)
- `ssh.systemctl.status` 7 of 8 failures are scope-rejected
- `aws ssm put-parameter` (multiple) - 7+ failures from non-tracked CWDs
- `aws ssm get-parameter` - same
- `ssh.kubectl get nodes` - 1 of 4 failures

Operators (Kai and Claude) invoke coily verbs from `~`, `/tmp`, or other non-git-toplevel directories. The `--commit-scope auto` default rejects them with an error that names the fix but does not apply it.

## Why it slipped

The audit-row contract requires every invocation to bind to a real repo. Auto-detect via `git toplevel` is the default. The default fails closed when cwd is not inside a tracked tree. This is correct from the audit-trail integrity standpoint. It is friction-loaded for ops verbs that have nothing to do with the cwd's repo identity (rotating an SSM parameter, restarting a game server) and where the operator naturally invokes from a scratch directory.

The error message names the fix (`pass --commit-scope=<repo-path> explicitly`) but does not surface what the operator actually wants: a sensible default for ops-style verbs that do not depend on cwd-as-repo. The fail-closed default is correct; the lack of a per-verb override or sensible fallback is the gap.

## Rule it produced

Anti-signal candidate for `coily-shared-meta`: **"scope auto-detect is transparent to the operator."** False. Auto-detect is transparent only inside tracked trees. Outside, every verb fails closed and the operator has to know to pass `--commit-scope=<path>` or run from a tracked dir.

The forward shape is not yet decided. Possible directions: (a) per-verb `cwd_scope_optional` annotation for verbs whose audit row does not need a repo binding; (b) `_unrooted` becoming a real first-class scope rather than an error; (c) operator-side discipline (always invoke from inside a repo). The decision needs a separate issue.

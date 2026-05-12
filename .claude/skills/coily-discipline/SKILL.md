---
name: coily-discipline
description: Agent-behavior rules for `coily`, the CLI security boundary. Bare-command denials route through coily wrappers, then `coily audit finding`. On coily ambiguity decrease your own permissions, don't loosen them. `coily lockdown` is permission-reducing by design. Triggers - coily, coily ops, coily lockdown, coily audit, permission denied, harness denial, settings.json, .claude/settings, allow rule, deny rule, audit finding, lockdown, escape hatch, route through coily.
---

# coily-discipline

Agent-behavior rules for the coily security boundary. Four rules apply when an agent works in or around a coily-managed environment.

Composes with: `coily-security-boundary-discipline` (the design surface of the boundary itself), `coily-meta-improvement` (the loop that produces and maintains coily rules), `coily-lockdown-meta` (lockdown-specific semantics).

## Bare-command denials route through coily, then file an audit finding

When the harness denies a bare external command (`gh`, `aws`, `kubectl`, `docker`, `tailscale`, etc.), retry through the coily wrapper (`coily ops gh ...`, `coily ops aws ...`, `coily ops kubectl ...`, `coily docker ...`, `coily tailscale ...`). The wrapper is the audited path the harness wants the agent on. Whenever a denial fires in any context, also run `coily audit finding` so the meta-improvement loop sees the friction.

This rule is the exception to the "denial means stop" instinct below: routing through the audited wrapper is the authorized recovery, not an escape hatch. Hand-edits of `.claude/settings*.json` and lateral workarounds (different flag, `go run`, etc.) are still off-limits.

## Default to decreasing your own permissions on coily ambiguity

When a coily-related action is ambiguous and the agent suspects the operator may not be actively steering (long auto-mode runs, scheduled wakeups, agent-spawned work), pick the option that tightens the boundary, not the one that loosens it. This is the inverse of the usual auto-mode "lean toward action" bias. For coily specifically, the safe direction is more restrictive.

If a coily invocation is denied by the harness, STOP and surface the question. Do not try alternative paths to accomplish the same effect (different flag, hand-edit, `go run`). The denial may be the harness correctly catching something the operator did not authorize.

Never reach for hand-edits of `.claude/settings*.json` with agent-constructed content as a workaround. That is self-modification with agent-inferred parameters. Lockdown writes canonical content baked into the binary; hand-edits do not.

## `coily lockdown` is permission-reducing by design

Every form (bare, `--apply`, `--apply --replace`, `--recursive`, `--recursive --apply`, `--recursive --apply --replace`) writes deny rules that constrain the agent. It is the canonical de-escalation tool.

When the operator authorizes a `coily lockdown` invocation, run it without re-asking permission to overwrite per-repo files or merge ancestor settings. That is what lockdown does, by design.

The paradox: "lockdown decreases permissions" coexists with "the harness may deny it because it touches the permission config." Resolve in favor of running lockdown when authorized. Resolve in favor of stopping when not. Loosening a deny rule, removing audit, or adding an allow rule almost never has a default answer. Always ask.

If a lockdown run in auto-mode locks the agent out of a path the operator later needs, the recovery is `coily lockdown --apply --replace` with a different `--path`, or hand-rollback from the operator's side. Do not silently un-lockdown to recover.

## Use prod coily to test local coily

When working inside the `coily` repo checkout, run the test suite via the brew-installed coily against the local checkout, not bare `go test`. Each repo declares its commands in `.coily/coily.yaml`. For coily itself: `cd <coily-checkout> && coily exec test [args...]`. Same for `vet`, `lint`, `lint-fix`, `cover`. The wrapper is audit-logged, obeys the lockdown deny list, and is the path the harness allows. Bare `go test` is denied by the deny-list pattern around the go toolchain. The `cd` is required because `coily exec` from the repo-parent cwd hits `exec_ambiguous_children`.

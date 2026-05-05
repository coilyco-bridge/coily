---
date: 2026-05-05
slug: kubectl-error-lost-in-exit-status-1
promoted_to:
  - anti-signal: "audit row's error field captures the underlying tool's error"
  - issue: https://github.com/coilysiren/coily/issues/63
---

# 2026-05-05 - `kubectl.get` failures land in audit with `error: "exit status 1"`, losing kubectl's stderr

## What was observed

`kubectl.get` recorded 5 invocations in the 35-day window. All 5 failed (100% rate). The audit rows carry `error: "exit status 1"` - the Go process-exit string, with no kubectl-side stderr captured. Sample rows:

- `coily kubectl get svc observability` → `error: "exit status 1"`
- `coily kubectl get pods --all-namespaces` → `error: "exit status 1"`

The bare top-level `kubectl` verb has 17 rows with 7 failures, and the same pattern: `error: "exit status 1"` for verbs accepted by the gate but failing downstream. Whatever kubectl said (no current context, no permission, server unreachable, resource not found) is gone by the time the audit row lands.

By contrast, the `gh` and `aws` verbs sometimes carry richer error strings (`scope: cwd is not inside a git repo`, `policy: shell metacharacter rejected`, `eco: remote sudo: ssh: ...`). Those are coily-side errors. The kubectl-side errors specifically are reduced to "exit status 1."

## Why it slipped

Coily's pass-through path executes `kubectl ...` and waits on the process. When the process exits non-zero, the Go layer captures the exit code but not the stderr. The audit row is built from the coily-side view: argv accepted, exit code 1. The kubectl stderr that explained why is on the operator's terminal (probably) but not in the durable record.

This is a different opaqueness than the eco-status one. There the message is verbose but unactionable. Here the message is missing entirely. For an audit log to be the trail, the trail has to carry enough to reconstruct both what was attempted and what came back. "Exit status 1" reconstructs the attempt but not the response.

## Rule it produced

Anti-signal: **"the audit row's `error` field captures the underlying tool's error."** False. For pass-through verbs that do not surface their own coily-side error, the field captures only the process exit. The downstream tool's stderr is not durable.

Forward shape: the pass-through layer captures stderr (or a tail of it) into the audit row. Unbounded stderr capture is a size risk. First-N-bytes or last-N-bytes is the typical cap. This applies to every pass-through, not just kubectl - aws and gh likely have the same gap when their failures don't trigger coily-side errors.

This finding rhymes with the eco-status one but at a different layer: there the error is too verbatim, here it is too summarized. The boundary between them is where the failure originates (coily layer vs. tool layer).

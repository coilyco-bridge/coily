---
name: coily-ops-kubectl-meta
description: Meta-improvement skill for the `coily ops kubectl` pass-through. Encodes anti-signals, sequencing rules, and references for the kubectl wrapper. Write-once observations live in `findings/YYYY-MM-DD-<slug>.md` siblings. Followup state lives on the GitHub issues those findings cite. Distinct from `coily-ops-kubectl-usage`, the flat lookup of how to invoke the verb today. Use this when adding or removing a kubectl sub-verb, when changing kubectl pass-through behavior (context routing, error capture, stderr handling), when an audit-log review surfaces a pattern of failures across kubectl verbs, or when an incident reveals lost downstream stderr in pass-through executions. Triggers - coily ops kubectl, kubectl passthrough, kubectl wrapper, kubectl audit row, kubectl get, kubectl apply, kubectl context, kubectl error capture, exit status 1, pass-through stderr, kubectl skill, kubectl meta-improvement.
---

# coily-ops-kubectl-meta

Meta-improvement layer for `coily ops kubectl`. The kubectl pass-through is lower-volume than gh or aws but failure-dense (100% failure rate on `kubectl.get` in the 35-day window) and exposes the pass-through-stderr-loss pattern that likely affects every pass-through verb.

Composes with: `coily-security-boundary-discipline` (audit trail load-bearing property), `coily-shared-meta`, `coily-ops-investigation` (stderr-loss is an opaqueness-vs-bug case), `coily-ops-kubectl-usage`.

## 1. Anti-signals

- **"the audit row's `error` field captures the underlying tool's error."** False for pass-through verbs. For verbs accepted by the gate but failing downstream, the field captures only the Go process exit (`error: "exit status 1"`). The downstream tool's stderr is not durable - it printed to the operator's terminal but did not land in the audit log. This affects every pass-through, not just kubectl, but kubectl shows it cleanest because its failures rarely produce coily-side errors.
  **Pin:** [findings/2026-05-05-kubectl-error-lost-in-exit-status-1.md](findings/2026-05-05-kubectl-error-lost-in-exit-status-1.md), [coily#63](https://github.com/coilysiren/coily/issues/63).

## 2. Sequencing rules

Generic ops sequencing rules live in `coily-shared-meta` and apply by inheritance.

No kubectl-specific sequencing rules seeded yet.

## 3. References

- `cmd/coily/ops_kubectl.go` - cli surface for `coily ops kubectl`.
- `pkg/policy` - argv-validation gate.
- `pkg/audit` - audit-row writer. Kubectl verbs land as `kubectl.*` (currently) or `ops.kubectl.*` (post-#50). The stderr-tail extension proposed in #63 lives here.
- `~/.coily/audit/*.jsonl` - filter `verb` prefix `kubectl.` for rows.
- `findings/` - dated write-once observations.

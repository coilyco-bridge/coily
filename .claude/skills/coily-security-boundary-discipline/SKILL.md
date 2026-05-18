---
name: coily-security-boundary-discipline
description: Discipline for designing, evaluating, and maintaining the coily CLI security boundary. Encodes the load-bearing properties (privileged-ops scope, escape-hatch resistance, audit trail), the anti-signals that survived prior design rounds (plumbed-through-makes-it-the-boundary, summary-as-shadow, drop-before-replace, doc-runtime drift), and the doc-runtime sync practice (`TestSecurityClaim_*` in `cmd/coily/security_claims_test.go`). Use when reviewing a security-boundary feature drop / add / refactor, when prose claims and runtime behavior may have drifted, when reasoning about whether a new feature actually expands the boundary or just uses it, or when sequencing a boundary change. Triggers - security boundary, threat model, escape hatch, audit trail, privileged op, off-host shadow, prompt injection, lockdown, deny list, allowlist inversion, claim vs runtime, doc-runtime drift, SECURITY.md, anti-signal, gate verification, plumbed through, replace before drop, verify the gate.
---

# coily-security-boundary-discipline

The practices and anti-signals that came out of coily's security-boundary work. Use this to keep prose, runtime, and design moves aligned when modifying any `coily-ops-*` or other privileged-ops surface.

Composes with: `coily-meta-improvement` (the loop), `coily-skill-authoring` (rule shape), `coily-shared-meta` (where generic ops rules live), `coily-ops-investigation` (when a boundary failure crosses components).

## 1. Load-bearing properties

The boundary is real if and only if all three hold. Drop any one and the boundary becomes a hopeful gesture, not a security artifact.

- **Privileged-ops scope.** The set of operations that must route through coily is enumerable, documented in `SECURITY.md`, and enforced at runtime by `cli-guard/policy`. An op outside the documented scope that still mutates is a gap.
- **Escape-hatch resistance.** No `SkipPolicy: true`, no `--bypass`, no environment-variable backdoor that lets the operator route around the gate. If escape hatches exist, they are themselves enumerated and audited.
- **Audit trail.** Every invocation lands a row in `~/.coily/audit/<owner>-<repo>.jsonl`, regardless of whether the gate denied or allowed. The row carries enough to reconstruct what was attempted.

## 2. Anti-signals

Phrases or framings that survived previous design rounds because nobody tested them.

- **"Plumbed through the gate makes it part of the boundary."** False. A verb that calls into `cli-guard/policy` is using the gate. The boundary includes only verbs whose policy actually constrains them. A pass-through that delegates 100% to argv-noop is plumbed but not gated.
- **"A summary stream is an off-host shadow of the audit log."** False. A summary loses the row-level fidelity required to reconstruct what happened. A real shadow preserves rows (rsync, S3 with object-lock, an append-only HTTP endpoint), not summaries.
  **Pin:** [coily#51](https://github.com/coilysiren/coily/issues/51), [coily#55](https://github.com/coilysiren/coily/issues/55).
- **"Drop the feature, then build the replacement."** False. Replace-before-drop preserves the boundary mid-flight. Drop-then-replace creates a window where the boundary is degraded and any in-flight op uses the unguarded path.
- **"Prose in `SECURITY.md` reflects current runtime."** Not unless a `TestSecurityClaim_*` test pins it. Doc-runtime drift is the default. The test is what holds them together.
- **"Context-free shell-metachar policy is the right default."** False given coily's direct-exec model. The metachar gate's threat model is "what if argv is shell-evaluated downstream"; coily executes via direct exec, no shell. For known content-flag values (jq expressions, markdown bodies, JSON literals) the metachars are inert in the actual execution path. Cost in the 35-day window: `gh.run.list` 96.6% rejected on jq's `|`, `gh issue` body args rejected on markdown `>`, `aws route53 change-resource-record-sets` rejected on JSON `{`. The gate is rejecting a threat that does not exist for these values.
  **Pin:** [findings/2026-05-05-metachar-gate-context-blind.md](findings/2026-05-05-metachar-gate-context-blind.md), [coily#60](https://github.com/coilysiren/coily/issues/60).

## 3. Sequencing rules for boundary changes

Generic boundary mechanics (policy-before-verb, deny-by-default for destructive, remove-policy-in-same-commit) live in `coily-shared-meta`. Boundary-specific sequencing:

- **Adding a load-bearing claim to `SECURITY.md` requires adding a corresponding `TestSecurityClaim_*` test in the same commit.**
  **Why:** doc-runtime sync practice. A claim without a test drifts. A test without a claim is unreadable.
  **How to apply:** any `SECURITY.md` edit that asserts runtime behavior.
- **Removing a load-bearing property requires the replacement to ship in a prior or same commit.**
  **Why:** the replace-before-drop principle at the boundary level. Properties are not removed unilaterally.
  **How to apply:** any change that drops audit, gate, or off-host-shadow capability.
- **A new feature that "uses the boundary" does not automatically expand it.**
  **Why:** plumbed-through is not gated. Distinguish use from extension at design time.
  **How to apply:** any feature ask phrased as "we route this through coily" - check whether the routing carries policy that constrains the verb, or whether it is a pass-through.

## 4. Decision template: is this on the boundary?

Three questions. All three must be yes for the feature to be on the boundary.

1. Does the verb mutate state outside the operator's local scope (cloud, repo, cluster, remote service)?
2. Does `cli-guard/policy` reject some non-empty subset of valid argv? (If policy is allow-all, the gate is plumbed but not gated.)
3. Does the audit row carry enough to reconstruct what was attempted?

If any answer is no, the feature uses the boundary but is not part of it. Document accordingly.

## 5. References

- `SECURITY.md` (in coily root) - the prose surface. Load-bearing claims live here.
- `cmd/coily/security_claims_test.go` (`TestSecurityClaim_*`) - the runtime pin for prose claims.
- `cli-guard/policy` - argv-validation gate.
- `cli-guard/audit` - audit-row writer.
- `cli-guard/verb`, `cli-guard/scope` - the verb-to-policy binding.
- `findings/` (in this directory) - dated write-once observations that produced the entries above.
- Composes-with: `coily-meta-improvement`, `coily-shared-meta`, `coily-ops-investigation`, every `coily-ops-*-meta` skill.

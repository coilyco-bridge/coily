---
name: coily-ops-aws-meta
description: Meta-improvement skill for the `coily ops aws` pass-through. Encodes anti-signals, sequencing rules, and references. Write-once observations live in `findings/YYYY-MM-DD-<slug>.md` siblings; followup state lives on the GitHub issues those findings cite. Distinct from `coily-ops-aws-usage`, the flat lookup of how to invoke the verb today. Use this when adding or removing an aws sub-verb, when changing argv-validation behavior, when reviewing a security-boundary claim about aws, when an audit-log review surfaces a pattern of denied or near-miss aws invocations, or when an incident reveals the aws gate did not catch what it claimed to catch. Triggers - coily ops aws, aws passthrough, aws gate, aws argv validation, aws audit row, aws lockdown, iam policy gap, aws deny list, aws read-only verb, aws destructive verb, sts get-caller-identity, aws drift, aws skill, aws meta-improvement, aws design round.
---

# coily-ops-aws-meta

Meta-improvement layer for `coily ops aws`. The aws pass-through is the canonical instance of the security-boundary discipline. When that discipline collides with aws-specific reality (read-only verbs that still leak, iam policies wider than the gate, sub-verbs that a friend's account does not have), the rules and counter-rules land here.

Composes with: `coily-security-boundary-discipline`, `coily-shared-meta`, `coily-ops-investigation`, `coily-ops-aws-usage`.

Conventions for `coily-*-meta` skills (catalogue vs. rule shape, write-once findings, when to reach for a Python helper) are in `coily-skill-authoring`.

## 1. Anti-signals

Phrases or framings that survived a previous design round because nobody tested them. Each entry pairs the bad shape with the actual property. Entries grounded by a finding, issue, or test carry a `**Pin:**` line. Absence of Pin means the entry is seeded, not yet grounded.

- **"argv validation is the boundary."** False. Argv validation is one layer. The boundary is the composition of argv validation + the audit row + the off-host shadow + the verification that the gate code is correct (`pkg/policy` tests). A regression in any one layer degrades the boundary even if the others still pass.
- **"iam allow is sufficient, the coily gate is belt-and-suspenders."** False. iam policies drift wider than the runtime needs (lazy scoping, role reuse). The coily gate is the layer that enforces the *intended* surface, narrower than what iam permits. Drop coily and the effective surface jumps to iam-wide.
- **"read-only aws verbs do not need an audit row."** False. Read-only verbs still exfiltrate (a `s3 ls` on a sensitive bucket, an `sts get-caller-identity` that confirms a role assumption). The audit row is the trail, not the gate. Trails apply to reads.
- **"audit row is sufficient for read-only verbs."** False. The trail documents the leak. It does not prevent it.
  **Pin:** [findings/2026-05-05-read-only-audit-without-gate.md](findings/2026-05-05-read-only-audit-without-gate.md), [coily#58](https://github.com/coilysiren/coily/issues/58).
- **"if it's denied at the iam edge, coily does not need to deny it."** False. iam denials happen after the request is sent. Coily denials happen before. The pre-send denial is what keeps an opaque-but-rejected attempt out of CloudTrail and out of the threat model "what was tried."

## 2. Sequencing rules

Aws-specific sequencing. Generic ops sequencing rules (policy-before-verb, deny-by-default for destructive, remove-policy-in-same-commit) are hoisted to `coily-shared-meta` and apply to all `coily-ops-*` skills by inheritance.

No aws-specific sequencing rules seeded yet.

## 3. References

- `cmd/coily/ops_aws.go` - cli surface for `coily ops aws`.
- `pkg/policy` - argv-validation gate.
- `pkg/audit` - audit-row writer. Aws verbs land as `ops.aws.*`.
- `cmd/coily/security_claims_test.go` (`TestSecurityClaim_*`) - prose-vs-runtime gate. Any aws claim added to `SECURITY.md` requires a matching test.
- `~/.coily/audit/*.jsonl` - per-repo audit trail. Filter `verb` prefix `ops.aws.` for aws rows.
- `findings/` - dated write-once observations that produced the entries above.

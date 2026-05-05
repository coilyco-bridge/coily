---
date: 2026-05-05
slug: aws-tool-binary-sha256-mismatch
promoted_to: []
---

# 2026-05-05 - embedded aws binary sha256 mismatch surfaced during a route53 invocation

## What was observed

One audit row in the window carries: `error: "shell: sha256 mismatch for https://github.com/coilysiren/coily/releases/download/tools-latest/aws-darwin-arm64.tar.gz: g..."` (truncated). Argv: `coily aws route53 change-resource-record-sets --hosted-zone-id Z06714552N3MO04UB ...`.

The release-pipeline auto-update of embedded tool versions (one of the design invariants captured in `coily-shared-meta` section 4) is verifying sha256 on download and rejecting on mismatch. That is the intended behavior: a sha256 mismatch is the supply-chain integrity check working.

What is unclear from the single row alone: whether this was a transient release-glitch (asset published while sha was being computed), a CDN cache-poisoning anomaly, or a legitimate signal that the tools-latest tag's published checksum drifted from the artifact at the GitHub release URL. Single occurrence, no other rows in the window with the same shape.

## Why it slipped

The supply-chain check is the boundary working. The slip, if any, is in observability: a sha256 mismatch is a load-bearing security event but it landed in the audit log as one row alongside argv-rejection rows of the same shape. There is no escalation, no notification, no separate sink. The signal lands in the same noise.

If a real supply-chain compromise had produced this row, the operator would only see it on a manual audit-log review. By design, every coily invocation produces a row, so the signal is buried in volume (~330 rows/day in this window).

## Rule it produced

This finding does not yet promote to an anti-signal or sequencing rule because n=1. It is recorded as evidence for the next time a sha256 mismatch is observed. If a second occurrence lands in the next 30 days, the pair promotes the pattern to a sequencing rule (sha256 mismatch routes to a separate sink with operator notification, not to the general audit log alone).

If no recurrence in 90 days, the row is treated as transient and this finding stays as the closest thing to a postmortem the event got.

No forward action filed. The single occurrence does not yet warrant an issue. If a second occurrence lands, the pair becomes the trigger.

---
date: 2026-05-05
slug: read-only-audit-without-gate
promoted_to:
  - anti-signal: "audit row is sufficient for read-only verbs"
  - issue: https://github.com/coilysiren/coily/issues/58
---

# 2026-05-05 - read-only aws verbs land an audit row but pass argv-gate-free

## What was observed

While seeding the anti-signal catalogue in `coily-ops-aws-meta`, the entry "read-only aws verbs do not need an audit row" was added (false: trails apply to reads). On review, the implicit inverse - "audit row is sufficient for read-only" - was identified as the actual runtime gap. Today `coily ops aws` for read-only sub-verbs writes an audit row and passes through without argv validation against sensitive resource patterns.

## Why it slipped

The trail-vs-gate distinction was clear in the security-boundary discipline at the destructive-verb layer (the gate denies, then the audit row records). At the read-only layer, the same distinction was implicit but never made into a runtime claim. The boundary code grew the destructive-verb gate without ever asking the symmetric question for reads. Doc and runtime both treated "read-only is low-blast-radius" as a tacit reason to skip the gate. Read-only is not low-blast-radius for exfiltration or state-confirmation classes of attack. It is just low-blast-radius for mutation.

## Rule it produced

Anti-signal catalogue entry: "audit row is sufficient for read-only verbs." False. The audit row is the trail. The trail documents the leak. It does not prevent it.

Forward action filed at [coily#58](https://github.com/coilysiren/coily/issues/58). Followup state lives there.

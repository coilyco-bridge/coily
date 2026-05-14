---
date: 2026-05-14
slug: trailer-hook-leaks-securestring-values
promoted_to:
  - issue: https://github.com/coilysiren/coily/issues/157
---

# 2026-05-14 - `coily git trailer-hook` embeds full audit argv (including SecureString `--value`) into commit message bodies

## What was observed

While committing in `coilysiren/repo-recall` (commit `757d1be`, since reset, never pushed), the pre-commit `coily git trailer-hook` appended four `Audit-log:` trailers to the commit body. Two of those rows were `ops.aws.ssm.put-parameter` invocations against `/tailscale/oauth/backend/*` with `--type SecureString`:

- ts `1778744466`, verb `ops.aws.ssm.put-parameter`, accept, exit 0 - argv contained `--name /tailscale/oauth/backend/client-id --value <plaintext-client-id> --type SecureString --overwrite`.
- ts `1778744472`, verb `ops.aws.ssm.put-parameter`, accept, exit 0 - argv contained `--name /tailscale/oauth/backend/secret --value <plaintext-tskey-client-secret> --type SecureString --overwrite`.

The trailer-hook emitted both rows verbatim, including the full `--value` argument. The plaintext secrets landed in the git commit message body. The commit was local-only and reset before push, so blast radius stayed on host. Kai revoked the affected tailscale OAuth credentials.

The audit log itself stores argv verbatim by design - that is the point of an audit trail. The violation is that the trailer-hook re-publishes argv into a less-restricted surface (commit messages, future PR bodies, anywhere the commit is mirrored) without scrubbing high-risk fields.

## Why it slipped

The audit-row format is one surface (operator-only, on-host, gated). Commit message trailers are a wholly different surface (replicated to every clone, surfaceable in PRs, attachable to public mirrors). The trailer-hook treats the audit log as the authoritative narrative source and re-emits rows as-is. There is no field-level scrub between the two surfaces.

The threat model that authorized SecureString puts via coily (per AGENTS.md "Stashing new secrets via coily ops aws ssm put-parameter --type SecureString is pre-authorized") assumed the value passed via argv would be captured in the audit log only, not promoted onto other surfaces. The trailer-hook silently widened the audit surface beyond that assumption.

Adjacent gap: the per-row capture itself includes `--value <secret>` because argv is captured before any sensitivity classification. Even a stricter trailer scrub leaves the on-host audit log holding plaintext for any verb whose secret lives in argv, not stdin or env. That is a separate but related boundary question.

## Rule it produced

Anti-signal: **"the audit log is the canonical narrative, just surface it wherever audit context is useful."** False. The audit log is the most-permissive surface coily controls. Re-emitting it onto any other surface (git trailers, JSON exports, dashboards rendered into public artifacts) requires a field-level scrub of value-bearing flags for verbs that handle secrets.

The forward shape: a deny-list of `(verb, flag)` pairs whose values must be redacted before any cross-surface emission. At minimum: `ops.aws.ssm.put-parameter --value`, `ops.aws.secretsmanager.* --secret-string`, anything labeled `SecureString`. The trailer-hook is the immediate fix surface; the same scrub should gate `coily audit dashboard` and any future export verb.

Independent rule: argv-as-secret-channel is the deeper smell. For verbs that exist to write a secret, the secret should arrive via stdin or `@/path/to/file` so the on-host audit log never holds the plaintext at rest either. That is a larger redesign and out of scope for this finding.

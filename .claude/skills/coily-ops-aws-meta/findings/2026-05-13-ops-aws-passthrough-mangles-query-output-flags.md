---
date: 2026-05-13
slug: ops-aws-passthrough-mangles-query-output-flags
promoted_to:
  - issue: https://github.com/coilysiren/coily/issues/147
---

# 2026-05-13 - coily ops aws drops trailing --query and --output flags despite the `--` separator

## What was observed

While fetching three SSM parameters during webhook setup (coilyco-ai#405), three consecutive `coily ops aws` invocations of the shape

```
coily --commit-scope=/home/kai/projects/coilysiren/coilyco-ai ops aws -- ssm get-parameter --name /coily/discord-webhook-url --with-decryption --query Parameter.Value --output text
```

failed with `aws` rejecting `--query`, `Parameter.Value`, `--output`, and `text` as four separate "Unknown options." Audit rows: `019e23b4-73d9-7829-be9f-dd2529090b90`, `019e23b4-7545-75a7-8dcd-110e7ed0c7b3`, `019e23b4-76a0-7e75-8be4-04ffb5cc3539` (all ts 1778715489, verb `ops.aws`, decision accept, exit_code 1). Each audit row recorded the full argv including `--query`/`--output`, so the mangling happens between coily-side audit capture and the actual exec of `aws`.

Workaround that worked first try: drop the `--query`/`--output` flags and parse the JSON response with `jq -r '.Parameter.Value'`. The rest of the argv (`--name`, `--with-decryption`, the SSM path) passed through unchanged.

## Why it slipped

The contract for `coily ops <tool> -- <args>` is that argv after `--` is forwarded byte-for-byte to the underlying binary. That contract is implicit in the verb's name ("passthrough"), not enforced by a test against the most common scripting flags of each wrapped tool. `--query` and `--output` are the two flags any aws scripting use will reach for first, so this fails on contact with normal use, not at some exotic edge.

The boundary itself accepted the calls and wrote correct audit rows. The friction is in the passthrough, not in the gate. But friction in a passthrough erodes the boundary the gate protects: an agent who hits "Unknown options" three times in a row is tempted to drop back to bare `aws`, which is exactly what the lockdown is supposed to prevent. A passthrough that is not a faithful proxy is a tax on doing the right thing.

## Rule it produced

Anti-signal candidate (not promoted, just data for now): "the `--` separator is sufficient to forward argv verbatim through a coily passthrough." Today it is not, at least for `ops.aws` with `--query`/`--output`. Forward action filed at [coily#147](https://github.com/coilysiren/coily/issues/147).

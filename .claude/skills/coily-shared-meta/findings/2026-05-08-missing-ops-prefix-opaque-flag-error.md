---
date: 2026-05-08
slug: missing-ops-prefix-opaque-flag-error
promoted_to:
  - issue: https://github.com/coilysiren/coily/issues/87
---

# 2026-05-08 - missing `ops` prefix on aws/gh/kubectl invocations surfaces as "flag provided but not defined"

## What was observed

ts 1778244954, verb captured as bare-root invocation, audit_log_path `/Users/kai/.coily/audit/coilysiren-factorio-mods.jsonl`. The agent invoked:

    coily aws ssm put-parameter --name /factorio/mod-portal/username --type String --value '<redacted>'
    coily aws ssm put-parameter --name /factorio/mod-portal/token    --type SecureString --value '<redacted>'

Both rejected with `flag provided but not defined: -name` plus the full top-level help dump. The actual fault is the missing `ops` prefix - the correct invocation is `coily ops aws ssm put-parameter ...`. urfave/cli walked `aws ssm put-parameter` past the root command (none of those are top-level commands), then tried to parse `--name` against the root flag set, where it does not exist. The error message names the symptom (`-name` unknown at root) rather than the cause (`aws` is not a root command; you meant `ops aws`).

## Why it slipped

Two gaps stack:

- urfave/cli's default behavior on unknown positional args is to keep walking until it hits a flag, then complain about the flag. There is no "did you mean `coily ops aws`?" hint and no "unknown command `aws`" error. The agent (and a sleepy human) reads the flag error and starts inspecting flag spelling, not command structure.
- The skill surface (`coily-passthroughs`, AGENTS.md "coily ops aws ...") names the correct invocation, but a model running on prior-knowledge muscle memory ("aws cli is `aws ssm put-parameter`") will drop the `ops` prefix and mirror the bare AWS CLI shape. The gate catches it (good) but the diagnostic does not point at the structural fix.

This is the on-the-train test failing in miniature: the recovery message does not name the next dictation-friendly command.

## Rule it produced

Candidate anti-signal for `coily-shared-meta`: "a `flag provided but not defined` error at the root command means a missing subcommand prefix, not a flag typo. Re-check whether `ops` (or `gaming`, `pkg`, `ssh`) is missing before inspecting flag spelling."

Candidate forward action: teach the root command to detect `aws | gh | kubectl | docker | tailscale` as a positional and emit `did you mean: coily ops <arg> ...?` before urfave/cli's flag parser runs. Same shape for `eco | factorio | icarus | core-keeper` -> `coily gaming ...`. Cheap, high-leverage, narrows the opaque-error class.

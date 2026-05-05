---
date: 2026-05-05
slug: metachar-gate-context-blind
promoted_to:
  - anti-signal: "shell-metachar policy is context-free and that is fine"
  - issue: https://github.com/coilysiren/coily/issues/60
---

# 2026-05-05 - argv shell-metachar gate rejects legitimate non-shell uses (jq, markdown, JSON)

## What was observed

Three repeated failure shapes in the 35-day window, all rooted in the same context-blind `policy.ValidateArg` rejection:

1. **`gh.run.list` 563 of 583 invocations rejected (96.6%).** Sample error: `policy: shell metacharacter rejected: arg --jq contains '|' at index 5`. The flag value is a jq expression `[.[] | select(.status != "completed")] | length`. The `|` is jq's pipe operator, not a shell pipe. The gate rejects it because it is `|` in argv, not because it would be interpreted by a shell.

2. **`gh.issue.comment` and `gh.issue.create` body args rejected.** Sample error: `policy: shell metacharacter rejected: arg positional[6] contains '>' at index 0`. The body content begins with `> 🤖 Filed by Claude Code` - a markdown blockquote. The leading `>` is markdown syntax, not shell redirection. Gate rejects.

3. **`aws route53 change-resource-record-sets` rejected.** Sample error: `policy: shell metacharacter rejected: arg --change-batch contains '{' at index 0`. The flag value is a JSON object literal. Gate rejects on `{`.

The metachar gate is doing exactly what it was designed to do: refuse argv values that contain characters which would be shell-meaningful if the argv were ever passed through a shell. The boundary is real. The gap is that coily executes argv via `exec`/`os.exec`, not via a shell. There is no shell to be meaningful in. The metachars are inert in the actual execution path.

## Why it slipped

The metachar gate inherits the threat model of "what if argv is ever shell-evaluated downstream." That threat exists for any verb that pipes through `bash -c` or similar. For the direct-exec path coily actually uses, the threat is theoretical. The gate's context-free design treats every argv value as if it might hit a shell. It does not.

The cost: high-volume legitimate use cases (jq, gh markdown bodies, aws JSON flag values) are 96.6% blocked. Operators (and Claude) hit the wall constantly. The audit log shows the same `--jq` argv rejected hundreds of times by the same operator - the wall is not an occasional speedbump but a daily blocker.

## Rule it produced

Anti-signal: **"context-free shell-metachar policy is the right default."** False given coily's direct-exec model. The right default is structural: known-multiline content (markdown bodies, jq expressions, JSON literals) gets passed via stdin or via an explicit `--from-file` shape, not as inline argv values that the gate then has to character-class-permit.

The forward shape: either (a) per-flag whitelist of known content types (`--jq`, `--body`, `--change-batch` get a context-aware exemption), or (b) a structured-input convention (`@-` for stdin, `@/path` for file) that bypasses metachar checks because the value never lives in argv. Option (b) matches the gh-cli, aws-cli, and curl conventions. Option (a) is friction-lower but maintains a longer exemption list. Decision is forward action.

This generalizes beyond aws and gh. Any pass-through whose underlying tool accepts a content-type flag (jq expression, JSON, YAML, markdown) will hit the same wall.

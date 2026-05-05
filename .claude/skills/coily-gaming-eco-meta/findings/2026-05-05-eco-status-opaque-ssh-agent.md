---
date: 2026-05-05
slug: eco-status-opaque-ssh-agent
promoted_to:
  - anti-signal: "remote-side errors surface usefully through coily"
  - issue: https://github.com/coilysiren/coily/issues/62
---

# 2026-05-05 - `coily eco status` is 100% failure with opaque ssh-agent error

## What was observed

`eco.status` recorded 6 invocations in the 35-day window. All 6 failed (100% rate). Sample error: `eco: remote sudo: ssh: no authentication method available (ssh-agent unreachable and no key path)`.

The error correctly identifies the immediate cause (ssh-agent unreachable, no key path configured). It does not name what the operator should do. The next reader (operator or agent) sees a verb that always fails and a string they cannot act on without external knowledge.

This is the opaqueness-vs-bug triage from `coily-ops-investigation`. The bug is environmental: ssh-agent was not loaded in the session that invoked `coily eco status`. The bug is fixable by the operator (`ssh-add ~/.ssh/<key>`). The opaqueness is in coily: the error does not say "load ssh-agent" or "configure a key path in `coily.yaml`." It states the symptom in transport-layer language.

## Why it slipped

The error string is a faithful rendering of what `pkg/ssh` saw. Coily's transport layer surfaces the underlying ssh failure verbatim. That preserves fidelity but loses actionability. The operator sees the SSH library's idiom instead of an instruction.

A connected gap: the verb itself is a thin wrapper around `ssh systemctl status eco-server`. When ssh-agent is unreachable, the verb cannot complete. There is no fallback or pre-check. The 100% failure rate is genuine - not a coily bug, an environmental precondition that coily does not surface as a precondition.

## Rule it produced

Anti-signal: **"remote-side transport errors surface usefully through coily."** False. They surface verbatim. Verbatim is fidelity, not actionability.

Forward shape (per the opaqueness-vs-bug rule in `coily-ops-investigation`):

- **Low-priority, opaque error → fix opaqueness first.** Add a precheck or a translated-error layer that converts common transport failures (no ssh-agent, no key path, host unreachable, sudo password required) into instruction-shaped messages.
- The verb itself is fine. The transport-error rendering is the gap.

This pattern applies beyond `eco status`. Every `ssh.*` verb has the same surface and the same exposure to the same transport-layer error idioms.

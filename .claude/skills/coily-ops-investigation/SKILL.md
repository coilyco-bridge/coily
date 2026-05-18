---
name: coily-ops-investigation
description: ALWAYS active on the words "investigate", "investigation", "investigative", "triage", "debug", "root cause", or any close variant - no exceptions, regardless of domain. Meta-skill for ops investigation work involving coily and the systems it fronts. Encodes the cross-cutting discipline that applies before any domain-specific reasoning - version-pin first, articulate the failure mechanism, codify anti-signals, route privileged writes through coily not improvisation, triage opaqueness vs bug. Also fires whenever an error, bug, exception, stack trace, broken MCP, weird production signal, or "what version are we on" request lands. Triggers - investigate, investigation, investigative, triage, debug, root cause, agentic ops, agentic SRE, error, exception, stack trace, bug, NRE, null reference, broken MCP, MCP failing, 403, 401, oncall, ops, SRE, reliability, postmortem, version check, what version, is this a known bug, why is X failing, weird signal, runbook, distributed system investigation.
---

# coily-ops-investigation

Cross-cutting discipline for any investigation involving coily or the hosts and services coily fronts. This skill does not investigate. It enforces what runs before any domain reasoning.

Composes with: `coily-meta-improvement` (findings produced by an investigation land in the appropriate `coily-*-meta`), `coily-shared-meta` (host fleet context), `coily-security-boundary-discipline` (when a privileged op is involved in or implicated by the failure).

## Always-active trigger: "investigate"

When the user says "investigate", "investigation", "investigative", "triage", "debug", "root cause", or any close variant, **this skill fires unconditionally**. Do not skip the universal first moves below.

If a stricter trigger names both this meta-shaped action and a domain (e.g. "investigate this aws gate failure"), fire both: this meta first (universal first moves), then the domain skill (`coily-ops-aws-meta` or whichever applies).

## Core principle: skills are reference + routing, not orchestration

Routed skills are reference docs plus simple routing. They do not embed orchestration, do not wrap APIs, do not multi-step tool sequences. Anything more elaborate belongs in coily itself (which is the wrapper) or a Python helper the skill calls.

The cleverness is in deciding when a skill fires and what signal belongs in its reference. Not in what the skill itself does.

## Universal first moves

These run before reaching for any domain skill, regardless of context.

### 1. Version-pin every implicated component

For any error from third-party code (mods, libraries, MCP servers, plugins, language runtimes), the first three actions are:

1. Identify the package and version.
2. Look up the latest release.
3. Scan recent changelog entries for the symptom.

Only then read code or theorize. A patched-upstream bug needs no investigation. It needs an upgrade. Skipping the version-pin step routinely wastes the entire investigation.

### 2. Articulate the failure mechanism in plain language with `file:line` causality

If you cannot say in one sentence what physically goes wrong - which line dereferences what null, which branch races which - you do not understand the bug yet. Stop generating fixes. Keep reading.

### 3. Enumerate input partitions that reach the failing code path

Which partitions fail. Which partitions pass. The shape of the partition is usually the bug.

### 4. Check for case-library precedent before generating

If the appropriate `coily-*-meta` skill has anti-signals or findings, read them before proposing a new theory. Most bugs in a domain rhyme.

### 5. Adversarial self-review post-fix

Try to construct an input that would still break the proposed fix. If you cannot, you have not stress-tested it.

### 6. Stop if the mechanism cannot be articulated

Do not open a PR, do not push a fix, do not declare the investigation closed. Bounce back to step 2.

### 7. Triage opaqueness vs. bug, by priority

Before fixing the bug, decide whether the **error itself** carried enough context to debug. If not, the opaqueness is its own bug and gets prioritized against the original by severity:

- **Low-priority, opaque error:** fix the opaqueness *first*. Better error message, structured fields, full component chain (every actor the call passed through), trigger phrases that route to the right skill, correlation IDs across components if the system supports it. Then fix the underlying bug. Rationale: opaqueness fixes pay back on every future failure of this shape.
- **Medium-priority:** fix both in parallel. Same commit or adjacent commits. Opaqueness work is not optional.
- **High-priority:** fix the bug now, then immediately file a follow-up issue for the opaqueness work. Do not let urgency bury the meta layer.

**Anti-signal:** "the error is annoying but I know what it means" is the most common excuse to skip the opaqueness fix. The point is not what *you* know - it is what the error tells the next reader, which is often agent-you with no context.

## Privileged ops route through coily, not improvisation

This skill does not document deploy / rollback / prod-DB-write / infra-mutation procedures. Those route through coily (argv validation, allow-list enforcement, audit-log writes, human gate) or are explicitly out of scope.

If an investigation reaches a point where the next action is a privileged write and there is no coily verb for it, **stop and tell the operator**. Do not improvise the write.

A skill that documents a high-blast-radius write workflow is a security asset only if a human gate is baked into the procedure. Better to have no skill than a skill that sands the friction off a privileged write.

## Anti-signal codification (the highest-leverage rule)

Positive instructions ("do X") are easy to write and low-value. Negative instructions ("do NOT waste time on X, even though it looks right") encode expensive-earned knowledge.

When extending any `coily-*-meta` skill from an investigation, the anti-signals section is mandatory and goes first. A finding that does not produce an anti-signal or sequencing rule is data, not a rule - file it as a finding without promotion. See `coily-meta-improvement`.

## What a coily-implicated investigation looks like

Failures involving coily typically cross at least two components: the laptop running coily, the remote target (kai-server, AWS, GitHub, a friend's host), and often a service in between (ssh, tailscale, kubectl context, iam). The investigation has to identify which layer failed before reaching for a domain skill.

Routing heuristic:

- **Argv rejected by the gate** → `coily-security-boundary-discipline` first (was the rejection correct?), then the per-verb `coily-*-meta` if the gate is wrong.
- **Argv accepted but the underlying call failed** → the per-verb `coily-*-meta` (aws / gh / kubectl / gaming-eco / etc.).
- **Audit row missing or malformed** → `coily-shared-meta` (audit architecture) and `coily-security-boundary-discipline` (audit trail is load-bearing).
- **Failure spans hosts** → `coily-shared-meta` (host fleet) for inventory, then per-verb meta.

## References

- `~/.coily/audit/*.jsonl` - audit trail. First place to look for what was attempted.
- `cli-guard/policy` - argv-validation gate. Second place to look for why an attempt was rejected.
- `cli-guard/audit` - audit-row writer.
- `findings/` (in this directory) - dated write-once observations of cross-cutting investigation patterns.
- Composes-with: `coily-meta-improvement`, `coily-shared-meta`, `coily-security-boundary-discipline`, every `coily-*-meta` skill.

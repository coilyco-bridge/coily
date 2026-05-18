---
name: coily-shared-meta
description: Inventory and shared rules that apply across all `coily-*-meta` skills. The host fleet that coily operationally fronts (Mac laptop, Windows host, kai-server, friends' machines), the audit log architecture (per-host JSONL with host captured but not indexed on), and generic ops sequencing rules hoisted out of per-verb meta skills. Distinct from per-verb skills like `coily-ops-aws-meta`; this skill carries the cross-cutting facts and rules. Use when reasoning about which host a coily verb reaches, where its audit row lands, what is destructive on which host, when designing a new coily verb that spans hosts, or when a sequencing rule applies to multiple `coily-*-meta` skills and should be hoisted here. Triggers - host fleet, kai-server, friends machine, friend host, multi-host, cross-host, audit log architecture, audit row, audit JSONL, per-host audit, host destructive, host idempotent, generic ops sequencing, hoisted rule, shared meta, coily inventory.
---

# coily-shared-meta

Cross-cutting inventory and rules for coily as a system. Anything that applies to more than one `coily-*-meta` skill lives here, so per-area skills stay focused on what is specific to that area.

Composes with: `coily-meta-improvement` (the loop), `coily-skill-authoring` (the convention that hoists generic rules here), every `coily-*-meta` skill (which inherits the rules below).

## 1. Host fleet

The hosts coily operationally fronts. Each host has an owner, a destructive surface, an idempotent surface, and an audit-row destination.

| Host class | Owner | Destructive surface | Idempotent surface | Audit row lands |
|---|---|---|---|---|
| Operator laptop (Mac) | Operator | Verbs invoked here that mutate remote state. | `coily ops aws sts get-caller-identity`, `coily ops kubectl get`, `coily audit *`, `coily ops gh` reads. | Local `~/.coily/audit/<owner>-<repo>.jsonl` on the laptop. |
| Operator laptop (Windows) | Operator | Same as Mac. | Same as Mac. | Local `%USERPROFILE%\.coily\audit\...` on the Windows host. |
| kai-server (homelab) | Kai (or matching ssh user) | `coily ssh deploy`, `coily gaming * restart/stop`, `coily ssh systemctl restart <unit>`. Service-impacting for whoever is using the server. | `coily gaming * status`, `coily ssh kubectl get`, journalctl tails. | On the originating laptop, not on kai-server. The verb runs on kai-server via the ssh transport but is initiated from a laptop. |
| Friend's machine | The friend | Whatever coily verb runs there from the friend's own laptop. | Whatever the friend's coily reads. | On the friend's laptop. Not on Kai's. |
| AWS / GitHub / mod.io / Trello / Discord | The respective service | `coily ops aws delete-*`, `coily ops gh repo delete`, etc. | Reads, list operations. | On the laptop that initiated. |

Friends' machines are not inspectable from Kai's laptop. A coily verb running on a friend's host produces an audit row on their own JSONL, not Kai's. Cross-host audit correlation is open at [coily#55](https://github.com/coilysiren/coily/issues/55). See the `coily-meta-improvement` rule that the meta-improvement loop does not index on host.

## 2. Audit log architecture

Per-host, per-repo JSONL at `~/.coily/audit/<owner>-<repo>.jsonl`. One row per coily invocation, regardless of whether the gate denied or allowed. Verb names are stable strings (`ops.aws.*`, `gaming.eco.*`, `audit.*`).

- **Host is captured in each row but is not the index key.** **Why:** meta-improvement does not need cross-host correlation as a primary axis - the patterns we care about are verb-level and shape-level, not who-ran-it. Capturing host preserves the option to notice host-specific patterns (a friend's host hitting deny rates Kai's does not) without making host the organizing dimension. **How to apply:** any audit-row schema change should keep host as a field, not promote it to a path component or a directory split.
- **Audit rows are append-only locally.** The off-host shadow that would make rows survivable beyond the host is open at [coily#55](https://github.com/coilysiren/coily/issues/55).
- **The audit row is the trail, not the gate.** A row landing does not mean the action was authorized. It means the action was attempted. Anti-signal codified in `coily-ops-aws-meta` and applies generally.

## 3. Generic ops sequencing rules (hoisted)

Rules that apply to all `coily-*-meta` skills. Per-area skills do not duplicate these. They reference them.

- **Argv-validation policy lands before the verb that uses it.**
  **Why:** the gate must reject before the underlying call can succeed. A verb shipping ahead of its policy entry passes through unvalidated for the time-between.
  **How to apply:** any PR that adds a coily sub-verb wrapping an external tool.
- **Destructive verb defaults to deny + explicit gate (`--i-mean-it` or similar), not deny + remove later.**
  **Why:** replace-before-drop applied at sub-verb granularity. Removing the gate first creates a window where the destructive call ships unguarded.
  **How to apply:** any verb whose underlying call mutates remote state.
- **Removing a verb removes its policy entry and any composed-with skill references in the *same* commit, never a follow-up.**
  **Why:** orphan policy entries become stale documentation that lies about the surface. Orphan skill references send future readers to dead pointers.
  **How to apply:** any coily-verb deletion PR.
- **A new top-level verb requires (a) a `cli-guard/policy` entry, (b) a `TestSecurityClaim_*` test if it makes a security-boundary claim, and (c) a corresponding `coily-<area>-meta` skill once it earns a finding.** **Why:** the boundary is the composition of code + test + skill. Shipping any one without the others creates a degradation gap. **How to apply:** any PR that adds a new top-level coily verb or sub-verb group.

## 4. Cross-cutting anti-signals

Anti-signals that apply across multiple `coily-*-meta` areas because the underlying mechanism is shared.

- **"scope auto-detect is transparent to the operator."** False. The `--commit-scope auto` default fails closed when cwd is not inside a tracked tree. Many ops verbs (game-server restarts, SSM parameter rotations, k8s queries) have nothing to do with the cwd's repo identity but still get rejected by the auto-detect. The `_unrooted.jsonl` audit file accumulated 485 rows in 35 days. The error names the fix but does not apply it.
  **Pin:** [findings/2026-05-05-scope-required-from-non-git-cwd.md](findings/2026-05-05-scope-required-from-non-git-cwd.md), [coily#59](https://github.com/coilysiren/coily/issues/59).

## 5. Coily design invariants

Decisions that are settled and apply across all `coily-*-meta` skills. Listed here so per-area skills do not re-derive them and so a friend onboarding to coily sees the shape of the cli before reaching for any verb.

- **Three-bucket token scoping, not per-verb.** Tokens are scoped `read` / `write` / `delete`. There is no `aws.route53:read` granularity. **Why:** per-verb token granularity multiplies the auth surface without a proportional reduction in blast radius. Three buckets capture the actual decision points operators care about. **How to apply:** when a new sub-verb lands, classify it into one of the three buckets. Do not invent a fourth.
- **Lockdown does not require a token.** Any operator can re-baseline the deny list. **Why:** token-gating the safety boundary is circular. The boundary exists to constrain operators who are mid-task, not to constrain the act of tightening it. **How to apply:** any feature that touches `coily lockdown` keeps the no-token requirement.
- **Mirror the underlying tool's subcommand structure.** `coily ops aws ssm get-parameter`, not `coily ops aws secret get`. Do not collapse verbs into a coily-native shorthand. **Why:** muscle memory and agent retraining cost both compound when the wrapper renames things. The cost of the mirror is one extra word per invocation. The cost of a custom shorthand is a decade of operator confusion. **How to apply:** when adding a new pass-through, the verb path matches the underlying tool's argv exactly.
- **Audit logs are global, not per-repo-only.** Rows live at `~/.coily/audit/<owner>-<repo>.jsonl`, but the log's durability outlives the repo. Deleting a repo locally does not delete its audit trail. **Why:** the audit log is the operator's trail, not the repo's metadata. Operator state outlives any individual repo. **How to apply:** never gate audit-log retention on repo state.
- **Release pipeline auto-updates embedded tool versions.** Each merge to main on coily checks for new aws-cli, kubectl, gh, tailscale releases and bumps the embed. **Why:** version drift between the operator's expectation and the wrapper's bundled binary is a silent failure mode. The pipeline keeps them aligned. **How to apply:** never pin an embedded version manually unless documenting why in the same commit.

## 6. References

- `cmd/coily/` - the cli surface.
- `cli-guard/policy` - argv-validation gate (the runtime layer of the sequencing rules above).
- `cli-guard/audit` - audit-row writer.
- `cmd/coily/security_claims_test.go` (`TestSecurityClaim_*`) - prose-vs-runtime gate.
- `~/.coily/audit/*.jsonl` - per-repo audit trails.
- [coily#55](https://github.com/coilysiren/coily/issues/55) - off-host audit shadow placeholder.
- `findings/` (in this directory) - dated write-once observations that produced the entries above.
- Composes-with: `coily-meta-improvement`, `coily-skill-authoring`, `coily-security-boundary-discipline`, every `coily-*-meta` skill.

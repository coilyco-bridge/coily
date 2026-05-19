---
name: coily-meta
description: Design surface for coily (the operator CLI). Meta-improvement loop, security-boundary discipline, agent discipline, per-verb anti-signals. Findings live as GitHub issues with label `finding`. Always fires on "investigate", "triage", "debug". Triggers - coily, coily audit, audit finding, investigate, triage, debug.
---

# coily-meta

The single skill for coily's design surface. Replaces the prior spread of `coily-*-meta`, `coily-*-usage`, and the cross-cutting discipline skills. Findings are GitHub issues on coilysiren/coily with label `finding`, not files on disk.

Composes with: the runtime layer in `cmd/coily/` and `cli-guard/`, the audit trail at `~/.coily/audit/*.jsonl`, the prose surface in `SECURITY.md`.

## 1. The meta-improvement loop

The loop that produces and maintains everything else in this skill. Five steps, each with a where-it-lives so the data does not pool in one place.

### 1.1 Observe

A concrete signal arrives. One of:

- **Audit-log sweep.** A scan of `~/.coily/audit/*.jsonl` (and Claude session-history denied-Bash entries) over a window finds a pattern - a verb invoked at unexpected scope, a deny rate that suggests a missing gate, a near-miss the gate caught but the next operator might not. Run `scripts/sweep.py` for the structured report.
- **Incident.** A coily verb did something the operator did not intend, or did not do something the operator did intend. Real or near-miss.
- **Catalogue review.** Reading existing anti-signals or sequencing rules below and noticing an implicit inverse, an unstated assumption, or a silent gap.
- **Friend report.** Someone running coily on their own host hit a shape that does not match Kai's experience.

The observation is a fact about the system, not yet a rule.

### 1.2 File the finding (as a GitHub issue)

A finding is a GitHub issue on coilysiren/coily with label `finding`. Body schema:

```markdown
## What was observed

<concrete, scoped to one shape; cite the audit row's ts, id, verb, argv>

## Why it slipped

<what gap in the gate, the audit, the docs, or the threat model let this through>

## Rule it produced

<one-line claim; may be empty if this finding is data, not a rule>
```

Title: `finding: <one-line summary>`. Label: `finding`.

The body should capture the observation as it was made: the audit row's facts, what was observed, why it slipped, the rule it implies. After that, edits and comments are both fine - GitHub issues are mutable and resolution updates (linked commits, "fixed in vX.Y.Z", reframing) are useful trail.

`coily audit finding` walks an agent through the file step from a single audit-row id.

### 1.3 Promote to a rule

When the finding produces or grounds a rule, edit this skill:

- **Anti-signal:** add a one-line entry to section 4 (shared) or 9 (per-verb), with `**Pin:**` linking the finding issue (and any follow-up issue).
- **Sequencing rule:** add a three-line entry (rule / Why / How to apply) to section 4 (shared) or 9 (per-verb), with the Why citing the finding.

Generic rules (apply to all `coily-ops-*`, not just one verb) live in section 4 (shared). Per-verb rules live in section 9. See section 2 (authoring) for the hoist test.

A finding can also fail to promote: the observation was real but did not generalize. That is fine. The finding stays as the closed issue. The next finding may rhyme with it and the pair may then warrant a rule.

### 1.4 File the forward action (when applicable)

If the finding implies a code change, a doc change, or a sequencing rule addition that is not yet in this file, the forward action is a separate GitHub issue on the appropriate repo (usually coilysiren/coily). Link both issues at each other. The forward-action issue is the source of truth for "what happens next." The finding remains the source of truth for "what was observed."

### 1.5 Verify (when applicable)

If the rule is verifiable at runtime - a `TestSecurityClaim_*` test, a CI check, a generated allowlist - the verification lives in the coily codebase. The Pin line points at the test. If the forward-action issue closes without a test, the rule does not promote to "validated." It stays "pinned to issue."

This is the only step that crosses out of this skill into code. Steps 1.1-1.4 are skill-resident. Step 1.5 is code-resident.

### 1.6 Vocabulary

- **Raw → rollup → consumer.** The data layering this loop follows. Finding issues are raw. This skill is the rollup. The agent invoking a coily verb is the consumer that inherits the rolled-up trust.
- **Continuous comprehension.** The property of always carrying an accurate, current model of the system in non-human substrate. Finding issues preserve raw data. The bounded sections below preserve the model. When they drift, comprehension has decayed.
- **Anti-signal codification.** The highest-leverage rule output. Negatively-framed instructions ("do not waste time on X, even though it looks right") encode expensive-earned knowledge.
- **Failure-shape as the unit of organization.** Section 9 is split by verb area because the area maps to a real component boundary in the cli, not because of finding-storage concerns.

## 2. Authoring conventions

Conventions for editing this skill.

### 2.1 Catalogue entries (anti-signals, references)

Two-line shape:

- Lead line: the claim or pointer.
- `**Pin:**` line: links to the finding issue, the forward-action issue, or the test that grounds the entry. Omit the Pin line when the entry is seeded but not yet grounded. Absence-of-Pin means seeded.

Do not write a status field. Finding issues describe followups in comments. If you want to know whether a Pin is still live, open the issue.

### 2.2 Rule entries (sequencing rules)

Three-line shape:

- Lead line: the rule, as an imperative or claim.
- `**Why:**` line: the originating finding, issue, or constraint. Where the why is empirical, link the finding issue rather than restating the evidence inline.
- `**How to apply:**` line: when the rule fires.

### 2.3 Hoist vs. stamp

Rules that apply across multiple verb areas live in section 4. Per-area sections only carry rules specific to that area.

Test: if a rule mentions only generic boundary mechanics (policy-before-verb, deny-by-default for destructive, remove-policy-in-same-commit), it is shared. If a rule mentions the area's specific resource (s3 buckets, k8s contexts, the eco systemd unit), it stays in the area section.

When in doubt, hoist. Per-area duplication is the worse failure mode.

### 2.4 Encode the why

Every rule and anti-signal exists because something happened that produced it. Capture the something, not just the rule. A rule with no why is harder to retire correctly when the situation changes. The Pin lines and Why lines are the structural enforcement.

### 2.5 Bias toward Python helpers

When the loop needs to parse files, walk a directory, query a JSONL or SQLite, or do any structured data manipulation, write a script in `scripts/` and have this skill call it. Pure prompt instructions are fine for narrative steps. Python is right for anything where determinism, speed, or testability matter. `scripts/sweep.py` is the canonical example.

### 2.6 Retire a rule

When the runtime fact that produced it changes (the verb is removed, the iam policy is restructured, the gate now catches what it did not before), delete the entry from this skill and file a new finding issue noting the retirement. The deleted entry's git history is the audit trail.

A rule pinned to a closed forward-action issue with a passing test is not retired. It is validated and stays in the catalogue.

## 3. Agent discipline

How agents behave when working in or around a coily-managed environment.

### 3.1 Bare-command denials route through coily, then file a finding

When the harness denies a bare external command (`gh`, `aws`, `kubectl`, `docker`, `tailscale`, etc.), retry through the coily wrapper (`coily ops gh ...`, `coily ops aws ...`, `coily ops kubectl ...`, `coily docker ...`, `coily tailscale ...`). The wrapper is the audited path the harness wants the agent on. Whenever a denial fires, also run `coily audit finding` so the loop sees the friction.

This is the exception to the "denial means stop" rule below: routing through the audited wrapper is the authorized recovery, not an escape hatch. Hand-edits of `.claude/settings*.json` and lateral workarounds (different flag, `go run`, etc.) are still off-limits.

### 3.2 Default to decreasing your own permissions on coily ambiguity

When a coily-related action is ambiguous and the agent suspects the operator may not be actively steering (long auto-mode runs, scheduled wakeups, agent-spawned work), pick the option that tightens the boundary, not the one that loosens it. This is the inverse of the usual auto-mode "lean toward action" bias. For coily specifically, the safe direction is more restrictive.

If a coily invocation is denied by the harness, STOP and surface the question. Do not try alternative paths to accomplish the same effect (different flag, hand-edit, `go run`). The denial may be the harness correctly catching something the operator did not authorize.

Never reach for hand-edits of `.claude/settings*.json` with agent-constructed content as a workaround. That is self-modification with agent-inferred parameters. Lockdown writes canonical content baked into the binary; hand-edits do not.

### 3.3 `coily lockdown` is permission-reducing by design

Every form (bare, `--apply`, `--apply --replace`, `--recursive`, `--recursive --apply`, `--recursive --apply --replace`) writes deny rules that constrain the agent. It is the canonical de-escalation tool.

When the operator authorizes a `coily lockdown` invocation, run it without re-asking permission to overwrite per-repo files or merge ancestor settings. That is what lockdown does, by design.

The paradox: "lockdown decreases permissions" coexists with "the harness may deny it because it touches the permission config." Resolve in favor of running lockdown when authorized. Resolve in favor of stopping when not. Loosening a deny rule, removing audit, or adding an allow rule almost never has a default answer. Always ask.

If a lockdown run in auto-mode locks the agent out of a path the operator later needs, the recovery is `coily lockdown --apply --replace` with a different `--path`, or hand-rollback from the operator's side. Do not silently un-lockdown to recover.

### 3.4 Use prod coily to test local coily

When working inside the `coily` repo checkout, run the test suite via the brew-installed coily against the local checkout, not bare `go test`. Each repo declares its commands in `.coily/coily.yaml`. For coily itself: `cd <coily-checkout> && coily exec test [args...]`. Same for `vet`, `lint`, `lint-fix`, `cover`. The wrapper is audit-logged, obeys the lockdown deny list, and is the path the harness allows. Bare `go test` is denied by the deny-list pattern around the go toolchain. The `cd` is required because `coily exec` from the repo-parent cwd hits `exec_ambiguous_children`.

### 3.5 Never prefix Bash with cd, export, or env when calling coily (or any allowlisted leading token)

The Claude Code harness allowlist (`Bash(coily:*)`, `Bash(gh:*)`, etc.) matches the **leading token** of the command. `cd ~/foo && coily ...` starts with `cd`, not `coily`, so `Bash(coily:*)` does not apply and the harness prompts on every call. With dozens of probes that is a wall of prompts. Same trap for `export PATH=... && coily`, `env X=Y coily`, and `MSYS_NO_PATHCONV=1 coily`.

Rules:

- Invoke binaries bare: `coily ops gh auth status`, never `cd somewhere && coily ops gh auth status`.
- To bind an audit row to a specific repo, use `coily --commit-scope=<repo-path> ...` instead of `cd <repo-path> && coily ...`. There is no opt-out by design (every audit row must bind to a real repo).
- If you genuinely need to change dirs (rare, e.g. `coily exec` from the repo-parent), accept the prompt or ask Kai to widen the allowlist explicitly. Do not paper over by chaining.
- The same rule applies to any other allowlisted leading token (`gh`, `aws`, `kubectl`, `docker`, etc.) - don't bury them behind `env` / `cd` / `PATH=`.

Cross-platform: this is a harness-level rule, not a host-level one. The Linux specifics (PATH injection, linuxbrew location) stay in `kai-linux-env`; the harness-allowlist discipline lives here so it loads on Mac, Windows, and Linux sessions alike.

**Pin:** [coily#180](https://github.com/coilysiren/coily/issues/180).

## 4. Shared rules and inventory

Cross-cutting facts and rules that apply across every verb area.

### 4.1 Host fleet

The hosts coily operationally fronts. Each host has an owner, a destructive surface, an idempotent surface, and an audit-row destination.

| Host class | Owner | Destructive surface | Idempotent surface | Audit row lands |
|---|---|---|---|---|
| Operator laptop (Mac) | Operator | Verbs invoked here that mutate remote state. | `coily ops aws sts get-caller-identity`, `coily ops kubectl get`, `coily audit *`, `coily ops gh` reads. | Local `~/.coily/audit/<owner>-<repo>.jsonl` on the laptop. |
| Operator laptop (Windows) | Operator | Same as Mac. | Same as Mac. | Local `%USERPROFILE%\.coily\audit\...` on the Windows host. |
| kai-server (homelab) | Kai (or matching ssh user) | `coily ssh deploy`, `coily gaming * restart/stop`, `coily ssh systemctl restart <unit>`. Service-impacting for whoever is using the server. | `coily gaming * status`, `coily ssh kubectl get`, journalctl tails. | On the originating laptop, not on kai-server. The verb runs on kai-server via the ssh transport but is initiated from a laptop. |
| Friend's machine | The friend | Whatever coily verb runs there from the friend's own laptop. | Whatever the friend's coily reads. | On the friend's laptop. Not on Kai's. |
| AWS / GitHub / mod.io / Trello / Discord | The respective service | `coily ops aws delete-*`, `coily ops gh repo delete`, etc. | Reads, list operations. | On the laptop that initiated. |

Friends' machines are not inspectable from Kai's laptop. A coily verb running on a friend's host produces an audit row on their own JSONL, not Kai's. Cross-host audit correlation is open at [coily#55](https://github.com/coilysiren/coily/issues/55).

### 4.2 Audit log architecture

Per-host, per-repo JSONL at `~/.coily/audit/<owner>-<repo>.jsonl`. One row per coily invocation, regardless of whether the gate denied or allowed. Verb names are stable strings (`ops.aws.*`, `gaming.eco.*`, `audit.*`).

- **Host is captured in each row but is not the index key.** **Why:** the loop does not need cross-host correlation as a primary axis. The patterns we care about are verb-level and shape-level. Capturing host preserves the option to notice host-specific patterns without making host the organizing dimension. **How to apply:** any audit-row schema change keeps host as a field, not a path component.
- **Audit rows are append-only locally.** The off-host shadow that would make rows survivable beyond the host is open at [coily#55](https://github.com/coilysiren/coily/issues/55).
- **The audit row is the trail, not the gate.** A row landing does not mean the action was authorized. It means the action was attempted.

### 4.3 Generic ops sequencing rules

- **Argv-validation policy lands before the verb that uses it.**
  **Why:** the gate must reject before the underlying call can succeed. A verb shipping ahead of its policy entry passes through unvalidated for the time-between.
  **How to apply:** any PR that adds a coily sub-verb wrapping an external tool.
- **Destructive verb defaults to deny + explicit gate (`--i-mean-it` or similar), not deny + remove later.**
  **Why:** replace-before-drop at sub-verb granularity. Removing the gate first creates a window where the destructive call ships unguarded.
  **How to apply:** any verb whose underlying call mutates remote state.
- **Removing a verb removes its policy entry and any cross-references in the same commit, never a follow-up.**
  **Why:** orphan policy entries become stale documentation that lies about the surface. Orphan references send future readers to dead pointers.
  **How to apply:** any coily-verb deletion PR.
- **A new top-level verb requires a `cli-guard/policy` entry, a `TestSecurityClaim_*` test if it makes a security-boundary claim, and an anti-signal entry in section 9 once it earns a finding.**
  **Why:** the boundary is the composition of code + test + skill. Shipping any one without the others creates a degradation gap.
  **How to apply:** any PR that adds a new top-level coily verb or sub-verb group.

### 4.4 Cross-cutting anti-signals

- **"scope auto-detect is transparent to the operator."** False. The `--commit-scope auto` default fails closed when cwd is not inside a tracked tree. Many ops verbs (game-server restarts, SSM parameter rotations, k8s queries) have nothing to do with the cwd's repo identity but still get rejected. The `_unrooted.jsonl` audit file accumulated 485 rows in 35 days. The error names the fix but does not apply it.
  **Pin:** finding [coily#229](https://github.com/coilysiren/coily/issues/229), forward [coily#59](https://github.com/coilysiren/coily/issues/59).

### 4.5 Coily design invariants

Settled decisions that apply across all verb areas. Listed here so per-area sections do not re-derive them and so a friend onboarding to coily sees the shape of the cli before reaching for any verb.

- **Three-bucket token scoping, not per-verb.** Tokens are scoped `read` / `write` / `delete`. There is no `aws.route53:read` granularity. **Why:** per-verb token granularity multiplies the auth surface without a proportional reduction in blast radius. **How to apply:** when a new sub-verb lands, classify it into one of the three buckets.
- **Lockdown does not require a token.** Any operator can re-baseline the deny list. **Why:** token-gating the safety boundary is circular. The boundary exists to constrain operators mid-task, not the act of tightening it. **How to apply:** any feature that touches `coily lockdown` keeps the no-token requirement.
- **Mirror the underlying tool's subcommand structure.** `coily ops aws ssm get-parameter`, not `coily ops aws secret get`. **Why:** muscle memory and agent retraining cost compound when the wrapper renames things. **How to apply:** when adding a new pass-through, the verb path matches the underlying tool's argv exactly.
- **Audit logs are global, not per-repo-only.** Rows live at `~/.coily/audit/<owner>-<repo>.jsonl`, but durability outlives the repo. **Why:** the audit log is the operator's trail, not the repo's metadata. **How to apply:** never gate audit-log retention on repo state.
- **Release pipeline auto-updates embedded tool versions.** Each merge to main on coily checks for new aws-cli, kubectl, gh, tailscale releases and bumps the embed. **Why:** version drift between the operator's expectation and the wrapper's bundled binary is a silent failure mode. **How to apply:** never pin an embedded version manually unless documenting why in the same commit.

## 5. Security boundary discipline

How to keep prose, runtime, and design moves aligned when modifying any `coily-ops-*` or other privileged-ops surface.

### 5.1 Load-bearing properties

The boundary is real if and only if all three hold. Drop any one and the boundary becomes a hopeful gesture.

- **Privileged-ops scope.** The set of operations that must route through coily is enumerable, documented in `SECURITY.md`, and enforced at runtime by `cli-guard/policy`. An op outside the documented scope that still mutates is a gap.
- **Escape-hatch resistance.** No `SkipPolicy: true`, no `--bypass`, no environment-variable backdoor that lets the operator route around the gate. If escape hatches exist, they are themselves enumerated and audited.
- **Audit trail.** Every invocation lands a row in `~/.coily/audit/<owner>-<repo>.jsonl`, regardless of whether the gate denied or allowed. The row carries enough to reconstruct what was attempted.

### 5.2 Anti-signals

- **"Plumbed through the gate makes it part of the boundary."** False. A verb that calls into `cli-guard/policy` is using the gate. The boundary includes only verbs whose policy actually constrains them. A pass-through that delegates 100% to argv-noop is plumbed but not gated.
- **"A summary stream is an off-host shadow of the audit log."** False. A summary loses the row-level fidelity required to reconstruct what happened. A real shadow preserves rows (rsync, S3 with object-lock, an append-only HTTP endpoint).
  **Pin:** [coily#51](https://github.com/coilysiren/coily/issues/51), [coily#55](https://github.com/coilysiren/coily/issues/55).
- **"Drop the feature, then build the replacement."** False. Replace-before-drop preserves the boundary mid-flight. Drop-then-replace creates a window where the boundary is degraded.
- **"Prose in `SECURITY.md` reflects current runtime."** Not unless a `TestSecurityClaim_*` test pins it. Doc-runtime drift is the default.
- **"Context-free shell-metachar policy is the right default."** False given coily's direct-exec model. The metachar gate's threat model is "what if argv is shell-evaluated downstream"; coily executes via direct exec, no shell. For known content-flag values (jq expressions, markdown bodies, JSON literals) the metachars are inert in the actual execution path. Cost in the 35-day window: `gh.run.list` 96.6% rejected on jq's `|`, `gh issue` body args rejected on markdown `>`, `aws route53 change-resource-record-sets` rejected on JSON `{`.
  **Pin:** finding [coily#227](https://github.com/coilysiren/coily/issues/227), forward [coily#60](https://github.com/coilysiren/coily/issues/60).

### 5.3 Sequencing rules for boundary changes

- **Adding a load-bearing claim to `SECURITY.md` requires adding a corresponding `TestSecurityClaim_*` test in the same commit.**
  **Why:** doc-runtime sync practice. A claim without a test drifts. A test without a claim is unreadable.
  **How to apply:** any `SECURITY.md` edit that asserts runtime behavior.
- **Removing a load-bearing property requires the replacement to ship in a prior or same commit.**
  **Why:** replace-before-drop at the boundary level.
  **How to apply:** any change that drops audit, gate, or off-host-shadow capability.
- **A new feature that "uses the boundary" does not automatically expand it.**
  **Why:** plumbed-through is not gated. Distinguish use from extension at design time.
  **How to apply:** any feature ask phrased as "we route this through coily" - check whether the routing carries policy that constrains the verb.

### 5.4 Decision template: is this on the boundary?

Three questions. All three must be yes for the feature to be on the boundary.

1. Does the verb mutate state outside the operator's local scope (cloud, repo, cluster, remote service)?
2. Does `cli-guard/policy` reject some non-empty subset of valid argv? (If policy is allow-all, the gate is plumbed but not gated.)
3. Does the audit row carry enough to reconstruct what was attempted?

If any answer is no, the feature uses the boundary but is not part of it.

## 6. Investigation discipline

Cross-cutting discipline for any investigation involving coily or the hosts and services coily fronts.

### 6.1 Always-active trigger

When the user says "investigate", "investigation", "investigative", "triage", "debug", "root cause", or any close variant, this section fires unconditionally. Do not skip the universal first moves below regardless of domain.

### 6.2 Universal first moves

1. **Version-pin every implicated component.** For any error from third-party code (mods, libraries, MCP servers, plugins, language runtimes), identify the package + version, look up the latest release, scan recent changelog entries for the symptom. A patched-upstream bug needs an upgrade, not an investigation.
2. **Articulate the failure mechanism in plain language with `file:line` causality.** If you cannot say in one sentence what physically goes wrong, you do not understand the bug yet. Stop generating fixes.
3. **Enumerate input partitions that reach the failing code path.** Which fail, which pass. The shape of the partition is usually the bug.
4. **Check for case-library precedent before generating.** Read section 9 (per-verb anti-signals) and the `finding`-labeled issues for the implicated area before proposing a new theory.
5. **Adversarial self-review post-fix.** Try to construct an input that would still break the proposed fix.
6. **Stop if the mechanism cannot be articulated.** Do not open a PR, do not push a fix, do not declare the investigation closed. Bounce back to step 2.
7. **Triage opaqueness vs. bug, by priority.** Decide whether the error itself carried enough context. If not, the opaqueness is its own bug. Low-priority + opaque error: fix the opaqueness first (better message, structured fields, component chain). Medium: fix both in parallel. High: fix the bug now, file a follow-up issue for the opaqueness work immediately. Anti-signal: "the error is annoying but I know what it means" is the most common excuse to skip the opaqueness fix.

### 6.3 Privileged ops route through coily, not improvisation

If an investigation reaches a point where the next action is a privileged write and there is no coily verb for it, stop and tell the operator. Do not improvise the write. A skill that documents a high-blast-radius write workflow is a security asset only if a human gate is baked into the procedure.

### 6.4 Routing heuristic

- **Argv rejected by the gate** → section 5 first (was the rejection correct?), then section 9 (per-verb) if the gate is wrong.
- **Argv accepted but the underlying call failed** → section 9 (the per-verb anti-signals).
- **Audit row missing or malformed** → section 4.2 (audit architecture) and section 5 (audit trail is load-bearing).
- **Failure spans hosts** → section 4.1 (host fleet) for inventory, then per-verb.

## 7. Skills are reference plus routing, not orchestration

This skill is a reference doc and a routing surface. It does not embed orchestration, does not wrap APIs, does not multi-step tool sequences. Anything more elaborate belongs in coily itself (which is the wrapper) or a Python helper this skill calls. The cleverness is in deciding when this skill fires and what signal belongs in its reference. Not in what the skill itself does.

## 8. Where the data lives

- **Findings.** GitHub issues on coilysiren/coily with label `finding`. Index: `gh issue list --repo coilysiren/coily --label finding --state all`. Body captures the observation; resolution and followup land in comments or body edits as the trail accumulates.
- **Audit rows.** `~/.coily/audit/<owner>-<repo>.jsonl`. Per-host, append-only. Not copied into this skill or its issues.
- **Aggregations across findings.** Write a Python helper in `scripts/`. Do not hand-curate aggregated views into this skill.
- **This skill.** Bounded. Anti-signals, sequencing rules, references. No raw data, no per-day logs, no aggregated reports.

## 9. Per-verb anti-signals

Per-area entries. Generic boundary mechanics live in section 4.3. Only verb-specific shapes go here.

### 9.1 `coily ops aws`

Canonical instance of the security-boundary discipline. When that discipline collides with aws-specific reality (read-only verbs that still leak, iam policies wider than the gate, sub-verbs a friend's account does not have), the rules and counter-rules land here.

- **"argv validation is the boundary."** False. Argv validation is one layer. The boundary is argv validation + audit row + off-host shadow + verification that the gate code is correct (`cli-guard/policy` tests). A regression in any one layer degrades the boundary.
- **"iam allow is sufficient, the coily gate is belt-and-suspenders."** False. iam policies drift wider than the runtime needs. The coily gate enforces the intended surface, narrower than what iam permits. Drop coily and the effective surface jumps to iam-wide.
- **"read-only aws verbs do not need an audit row."** False. Read-only verbs still exfiltrate. The audit row is the trail, not the gate. Trails apply to reads.
- **"audit row is sufficient for read-only verbs."** False. The trail documents the leak. It does not prevent it.
  **Pin:** finding [coily#219](https://github.com/coilysiren/coily/issues/219), forward [coily#58](https://github.com/coilysiren/coily/issues/58).
- **"if it's denied at the iam edge, coily does not need to deny it."** False. iam denials happen after the request is sent. Coily denials happen before. The pre-send denial keeps an opaque-but-rejected attempt out of CloudTrail and out of the threat model "what was tried."

References: `cmd/coily/ops_aws.go`, audit-row verb prefix `ops.aws.`.

### 9.2 `coily ops gh`

High-volume (~1000 audit rows in a 35-day window), the single largest source of friction-driven workarounds. Anti-signals come from real audit-log evidence.

- **"the wrapper exists, therefore the agent uses it."** False. The agent uses the path of least denial. Raw `gh` denied by Claude Code's permission boundary, without the deny message naming the wrapper, is the path the agent learns. 113 raw `gh` denials in 35d while `coily ops gh` was actively exercised 1000+ times. Lockdown is the mechanism that closes this. Rollout is the gap.
  **Pin:** finding [coily#221](https://github.com/coilysiren/coily/issues/221), forward [coily#61](https://github.com/coilysiren/coily/issues/61).

References: `cmd/coily/ops_gh.go`, audit-row verb prefix `gh.` (currently) or `ops.gh.` (post-#50).

### 9.3 `coily ops kubectl`

Lower-volume than gh or aws but failure-dense (100% failure rate on `kubectl.get` in the 35-day window). Exposes the pass-through-stderr-loss pattern that likely affects every pass-through verb.

- **"the audit row's `error` field captures the underlying tool's error."** False for pass-through verbs. For verbs accepted by the gate but failing downstream, the field captures only the Go process exit (`error: "exit status 1"`). The downstream tool's stderr is not durable. This affects every pass-through, not just kubectl, but kubectl shows it cleanest.
  **Pin:** finding [coily#225](https://github.com/coilysiren/coily/issues/225), forward [coily#63](https://github.com/coilysiren/coily/issues/63).

References: `cmd/coily/ops_kubectl.go`, audit-row verb prefix `kubectl.` (currently) or `ops.kubectl.` (post-#50).

### 9.4 `coily systemctl`

Lower-volume than the `ops` passthroughs but failure-dense at hosts where the per-unit sudoers fragment does not strict-match every verb in the closed set. The cleanest demonstration of the boundary-vs-perimeter question.

- **"per-unit sudoers carveouts are the gate."** False. coily is the gate. Per-unit NOPASSWD lists in `/etc/sudoers.d/<repo>` duplicate the closed verb set already enforced inside coily, and drift via sudoers strict-match: a fragment that lists `restart` + `status` for one service silently fails to cover `stop` / `disable` / `daemon-reload`, or covers `<service>.service` but not the matching `.timer`. The duplication is the failure mode, not the safety.
  **Pin:** finding [coily#233](https://github.com/coilysiren/coily/issues/233), forward [coily#203](https://github.com/coilysiren/coily/issues/203).
- **"inner sudo works on every host."** False on non-tty sessions (Claude Code Bash tool, systemd-spawned shells, cron) when the host lacks a per-unit NOPASSWD rule that strict-matches the full argv. Sudo refuses to prompt without a tty and the verb fails with `sudo: a terminal is required to read the password`.
  **Pin:** finding [coily#233](https://github.com/coilysiren/coily/issues/233), forward [coily#203](https://github.com/coilysiren/coily/issues/203).

Friend-shippable host fleet rule: every coily-managed host carries `(ALL) NOPASSWD: /home/linuxbrew/.linuxbrew/bin/coily` (or the equivalent install path). Hosts without that grant fall back to per-unit sudoers and the strict-match failure mode is in play.

References: `cmd/coily/ops_systemctl.go`, `cmd/coily/ops_systemctl_test.go`, closed verb set in `cli-guard/policy` (`status` / `start` / `stop` / `restart` / `enable` / `disable` / `daemon-reload`), audit-row verb prefix `systemctl.`.

### 9.5 `coily gaming eco`

Operates the eco-server systemd unit on kai-server via the ssh transport. Failures typically split into transport-layer (ssh, sudo, key path) and game-server-state.

- **"remote-side transport errors surface usefully through coily."** False. They surface verbatim. Verbatim is fidelity, not actionability. `coily eco status` failed 6/6 with `ssh: no authentication method available (ssh-agent unreachable and no key path)` - correct, faithful, not actionable.
  **Pin:** finding [coily#218](https://github.com/coilysiren/coily/issues/218), forward [coily#62](https://github.com/coilysiren/coily/issues/62).

References: `cmd/coily/ops_eco.go`, `cmd/coily/ops_eco_mod.go`, `cli-guard/ssh` for the transport layer, audit-row verb prefix `eco.` or `gaming.eco.`.

## 10. References

- `cmd/coily/` - the cli surface.
- `cli-guard/policy` - argv-validation gate (the runtime layer of the sequencing rules).
- `cli-guard/audit` - audit-row writer.
- `cli-guard/verb`, `cli-guard/scope` - the verb-to-policy binding.
- `cli-guard/ssh` - transport layer for remote verbs.
- `cmd/coily/security_claims_test.go` (`TestSecurityClaim_*`) - prose-vs-runtime gate.
- `SECURITY.md` (in coily root) - the prose surface. Load-bearing claims live here.
- `~/.coily/audit/*.jsonl` - per-repo audit trails.
- `scripts/sweep.py` - structured-report generator for audit-log scans, called from step 1.1.
- [`gh issue list --repo coilysiren/coily --label finding`](https://github.com/coilysiren/coily/issues?q=label%3Afinding) - the finding index.
- [coily#55](https://github.com/coilysiren/coily/issues/55) - off-host audit shadow placeholder.
- [coily#215](https://github.com/coilysiren/coily/issues/215) - the consolidation that produced this skill shape.

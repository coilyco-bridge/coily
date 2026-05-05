---
name: coily
description: coily is Kai's operational meta-aggregator across distributed systems - her Mac laptop, her Windows host, kai-server (k3s + Eco game server + sirens-discord-ops + factorio + core-keeper + icarus + thermal-heartbeat), and friends' machines that run instances of the same Eco-community stack. This skill encodes that scope so coily verbs are understood as multi-host operations (verb X reaches host Y, audit row lands on host Z), the audit log is understood as a per-host artifact with deliberate cross-host gaps (see coily#55), and per-host operational expectations (who owns it, what's destructive there, what fails when host Y is offline) are visible by default rather than re-derived per session. Lives in coily/.claude/skills/coily/, the named exception to the "skills live in coilyco-ai" rule because coily IS the operational layer it would otherwise be a runbook for. Composes with security-boundary-discipline (general CLI-as-security-boundary discipline, in coilyco-ai) and coilyco-ops-investigation (when a coily failure crosses component boundaries). Use when running any coily verb that touches a non-laptop host, when reasoning about which host an op reaches, when designing a new coily verb that spans hosts, when investigating a coily failure that crosses host or component boundaries, when onboarding to coily as a system rather than as a CLI. Triggers - coily, kai-server, eco server, eco-server, factorio, factorio-server, core-keeper, icarus, sirens-discord-ops, sirens-echo, thermal-heartbeat, k3s, kubectl, ssh deploy, friend's eco server, friends' machines, friend's machine, eco community, multi-host, cross-host, host fleet, which host, where does this run, audit aggregation, audit log per host, audit shadow, coily verb, coily as a system, coily fleet, coily operator.
---

# coily

Coily is the operational meta-aggregator across the systems Kai (and the Eco-community friends) actually run. This file is the seed; the body gets filled in a separate session.

## Why this skill exists in the coily repo, not coilyco-ai

The general pattern (`kai-skill-authoring`) is that all skills, including investigation skills for tools that live in other repos, land in `coilyco-ai/.claude/skills/`. Centralizing optimizes for the consumer (the operator under partial-failure conditions) over the author.

Coily is the named exception. **Why:** the file you are reading is not a runbook *about* coily; it is the canonical reference for *what coily is operationally*. A new agent reading the coily codebase should find this skill alongside the code so the multi-host framing is part of the codebase's self-description, not a footnote in a different repo. The runbook-monorepo rationale (which keeps eco-investigation, sentry-backend-bugs, etc. in coilyco-ai) does not apply to coily itself, because coily is the system, not a system-being-investigated. **How to apply:** new skills *about* coily failures or specific coily features still go in coilyco-ai. New skills that encode coily's operational identity (what it acts on, what hosts it touches, what its cross-host story is) live here.

Flagged 2026-05-05 during coily#49 follow-up.

## Conventions for filling this out

Per `kai-skill-authoring`, each substantive section follows the same shape:

- **Lead with the rule.** One short imperative or claim.
- **`**Why:**` line.** The incident, constraint, or prior failure that produced the rule. Cite the originating commit / issue / dated finding.
- **`**How to apply:**` line.** When the rule fires.
- **Date-stamp** where the why is empirical.

Catalogue sections (the host fleet, the verb-to-host map) can be tables; rule sections use the three-part shape.

When the next fill needs structured data (parsing the cli command tree to enumerate verbs and their target hosts, parsing audit JSONLs to confirm cross-host expectations, walking SECURITY.md against actual host coverage), reach for a committed Python script in this directory rather than encoding the procedure as prompt. See the "Bias toward Python helpers" rule in `kai-skill-authoring`.

## What goes here (when this skill is filled out)

The fill is deferred. These are the seeds the next session should chew on, not topic headings to expand without thought.

### 1. The host fleet

Catalogue the hosts coily operationally fronts. At minimum:

- **Kai's Mac laptop.** Primary operator surface. Most coily invocations originate here. Audit log lives at `~/.coily/audit/<owner>-<repo>.jsonl`.
- **Kai's Windows host.** Second operator surface. Same coily binary (cross-compiled), separate audit log on its own filesystem. Cross-host correlation is open per coily#55.
- **kai-server (the homelab).** Operates the k3s cluster, plus systemd-managed game servers and bots: `eco-server`, `factorio-server`, `core-keeper-server`, `icarus-server`, `sirens-discord-ops`, `sirens-echo`, `thermal-heartbeat`. coily reaches it via the `pkg/ssh` SDK (not a subprocess) and via `kubectl` over tailscale. Some coily verbs (`coily eco`, `coily ssh deploy`) run *on* kai-server through the ssh transport but are *initiated* from a laptop, so the audit row lives on the laptop, not the server.
- **Friends' machines running the Eco-community stack.** Other operators run their own instances of the same systemd units (`eco-server`, possibly `sirens-discord-ops`). coily is the intended operator surface for them too, so the design questions ("which host, which audit log, which deny rule") are not Kai-only.
- **AWS, GitHub, mod.io, Trello, Discord (as services, not hosts).** coily verbs front these. They are not hosts, but they are surfaces where coily ops have effects, so the verb-to-surface map should include them.

### 2. Per-host operational expectations

For each host class, the next fill should encode:

- Who owns the host (Kai, kai-server-as-a-shared-resource, a friend, AWS).
- What's destructive there. `coily ssh deploy` overwrites files on kai-server. `coily ops aws ssm delete-parameter` is irrecoverable at the AWS edge. `coily gaming eco restart` is service-impacting for whoever's playing on kai-server right now.
- What's idempotent. `coily gaming eco status`, `coily ops aws sts get-caller-identity`, `coily ops kubectl get` are safe to repeat.
- Who is the operator on that host. For Kai's laptops, Kai. For kai-server, Kai or the matching ssh user. For friends' machines, the friend - and the agent running coily on Kai's behalf is *not* the friend.
- What audit context applies. Where the row lands, what `commit_scope` it binds to, whether the audit trailer is reachable from the host that owns the affected resource.

### 3. Cross-host coordination

Today the audit log is per-host. Cross-host correlation is open (coily#55). The next fill should make this gap explicit: when the question is "what did coily do across all hosts in window T," the answer today is "ssh into each host and read its JSONL." That is acceptable for current usage modes, not acceptable indefinitely.

Commit trailers reference the originating host's audit row. A trailer authored on the Mac references a row that does not exist on the Windows laptop's audit log, and vice versa. This is also documented in `pkg/audit` package comments.

### 4. Failure modes that span hosts

When the next fill enumerates coily failure modes, the multi-host axis matters: host offline, SSH key not loaded, kubeconfig pointing at a stale context, AWS creds missing, tailscale disconnected, the friend's host running an out-of-date coily binary. Each affects a different subset of the verb surface.

### 5. Composes with

- **`security-boundary-discipline`** (in coilyco-ai). General CLI-as-security-boundary discipline. Coily is the canonical instance, but the discipline applies to any future CLI gate.
- **`coilyco-ops-investigation`** (in coilyco-ai). When a coily failure crosses component boundaries (coily + ssh + kai-server + a game server, say), the investigation skill is the router; this skill is the operational map.
- **`kai-windows-env`** (in coilyco-ai). Windows-host specifics that affect coily on the second laptop.

## Status

**Shell only, deferred.** Originating thread is coily#49 (closed); this skill was extracted from that thread when it became clear coily's operational identity is bigger than the security-boundary-discipline framing alone. Fill happens in a separate session.

The `.gitignore` change that lets this skill be tracked at all is in the same commit. Setup-side question (how this skill gets symlinked into `~/.claude/skills/<name>` so the harness picks it up, given `coilyco-ai/setup.sh` only walks `coilyco-ai/.claude/skills/`) is deliberately not solved here. Kai will build to it.

Flagged 2026-05-05.

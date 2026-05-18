---
name: coily-gaming-eco-meta
description: Meta-improvement skill for the `coily gaming eco` verb group (status, start, stop, restart, tail, mod push, world snapshot/get-seed/set-seed/randomize). Encodes anti-signals, sequencing rules, and references for the eco game-server operator surface. Write-once observations live in `findings/YYYY-MM-DD-<slug>.md` siblings. Followup state lives on the GitHub issues those findings cite. Distinct from `coily-gaming-eco-usage`, the flat lookup of how to invoke the verbs today. Use this when adding or removing an eco sub-verb, when changing eco-server transport behavior (ssh, sudo, key path), when an audit-log review surfaces a pattern of failures across eco verbs, or when an incident reveals an opaque transport-layer error. Triggers - coily gaming eco, coily eco, eco-server, eco status, eco start, eco stop, eco restart, eco tail, eco mod push, eco world, ssh-agent error, remote sudo, eco skill, eco meta-improvement.
---

# coily-gaming-eco-meta

Meta-improvement layer for `coily gaming eco`. The eco verb group operates the eco-server systemd unit on kai-server via the ssh transport. Failures here typically split into transport-layer (ssh, sudo, key path) and game-server-state (eco-server is up but reporting trouble).

Composes with: `coily-shared-meta` (host fleet, audit architecture), `coily-ops-investigation` (transport failures route through the universal first moves), `coily-gaming-eco-usage`.

## 1. Anti-signals

- **"remote-side transport errors surface usefully through coily."** False. They surface verbatim. Verbatim is fidelity, not actionability. `coily eco status` failed 6/6 with `ssh: no authentication method available (ssh-agent unreachable and no key path)` - correct, faithful, not actionable. The next reader sees a verb that always fails and a string they cannot act on without external knowledge.
  **Pin:** [findings/2026-05-05-eco-status-opaque-ssh-agent.md](findings/2026-05-05-eco-status-opaque-ssh-agent.md), [coily#62](https://github.com/coilysiren/coily/issues/62).

## 2. Sequencing rules

Generic ops sequencing rules live in `coily-shared-meta` and apply by inheritance.

No eco-specific sequencing rules seeded yet.

## 3. References

- `cmd/coily/ops_eco.go` and `cmd/coily/ops_eco_mod.go` - cli surface for `coily gaming eco`.
- `cli-guard/ssh` - transport. The verbatim-error surface lives here.
- `cli-guard/audit` - audit-row writer. Eco verbs land as `eco.*` and `gaming.eco.*`.
- `~/.coily/audit/*.jsonl` - filter `verb` prefix `eco.` or `gaming.eco.` for rows.
- `findings/` - dated write-once observations.

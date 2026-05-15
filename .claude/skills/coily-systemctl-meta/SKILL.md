---
name: coily-systemctl-meta
description: Meta-improvement skill for the `coily systemctl` verb. Encodes anti-signals, sequencing rules, and references for the systemctl wrapper. Write-once observations live in `findings/YYYY-MM-DD-<slug>.md` siblings. Followup state lives on the GitHub issues those findings cite. Distinct from `coily-systemctl-usage`, the flat lookup of how to invoke the verb today. Use this when adding or removing a systemctl sub-verb, when changing how coily acquires root for mutating verbs (inner sudo vs self-elevate), when an audit-log review surfaces a pattern of password-prompt failures, when an incident reveals a per-unit sudoers carveout has drifted from the closed verb set, or when reasoning about whether coily-as-boundary obviates a per-unit sudoers fragment. Triggers - coily systemctl, systemctl passthrough, systemctl wrapper, systemctl audit row, systemctl stop, systemctl restart, systemctl daemon-reload, sudoers strict-match, NOPASSWD, per-unit sudoers, inner sudo, self-elevate, sudo coily, systemd unit, sirens-discord-ops-update.timer, password prompt non-tty, systemctl meta-improvement.
---

# coily-systemctl-meta

Meta-improvement layer for `coily systemctl`. The systemctl verb is lower-volume than the `ops` passthroughs but failure-dense at hosts where the per-unit sudoers fragment does not strict-match every verb in the closed set. The verb is also the cleanest demonstration of the boundary-vs-perimeter question: coily wants to be the gate, but the current shape delegates gate-keeping to sudoers strict-match in addition.

Composes with: `coily-security-boundary-discipline` (the "coily IS the boundary" property is what justifies self-elevation), `coily-shared-meta` (host fleet, audit architecture, generic ops sequencing), `coily-ops-investigation` (sudo-prompt-on-non-tty is an opaqueness-vs-bug case), `coily-systemctl-usage`.

## 1. Anti-signals

- **"per-unit sudoers carveouts are the gate."** False. coily is the gate. Per-unit NOPASSWD lists in `/etc/sudoers.d/<repo>` duplicate the closed verb set already enforced inside coily, and drift via sudoers strict-match: a fragment that lists `restart` + `status` for one service silently fails to cover `stop` / `disable` / `daemon-reload`, or covers `<service>.service` but not the matching `.timer`. The duplication is the failure mode, not the safety.
  **Pin:** [findings/2026-05-15-inner-sudo-non-tty-strict-match.md](findings/2026-05-15-inner-sudo-non-tty-strict-match.md), [coily#203](https://github.com/coilysiren/coily/issues/203).

- **"inner sudo works on every host."** False on non-tty sessions (Claude Code Bash tool, systemd-spawned shells, cron) when the host lacks a per-unit NOPASSWD rule that strict-matches the full argv. Sudo refuses to prompt without a tty and the verb fails with `sudo: a terminal is required to read the password`. The verb has no way to recover in that context.
  **Pin:** [findings/2026-05-15-inner-sudo-non-tty-strict-match.md](findings/2026-05-15-inner-sudo-non-tty-strict-match.md), [coily#203](https://github.com/coilysiren/coily/issues/203).

## 2. Sequencing rules

Generic ops sequencing rules live in `coily-shared-meta` and apply by inheritance.

No systemctl-specific sequencing rules seeded yet. Candidate (not yet promoted): when adding a new mutating systemctl sub-verb, the closed-set test in `cmd/coily/ops_systemctl_test.go` must list it before the cli registration ships, so the gate rejects unknown verbs by construction rather than by sudoers happenstance.

## 3. References

- `cmd/coily/ops_systemctl.go` - cli surface for `coily systemctl`.
- `cmd/coily/ops_systemctl_test.go` - closed-verb-set test.
- `pkg/policy` - argv-validation gate. Closed verb set: `status` / `start` / `stop` / `restart` / `enable` / `disable` / `daemon-reload`.
- `pkg/audit` - audit-row writer. Systemctl verbs land as `systemctl.<verb>`.
- `~/.coily/audit/*.jsonl` - filter `verb` prefix `systemctl.` for rows.
- `findings/` - dated write-once observations.
- Friend-shippable host fleet rule: every coily-managed host carries `(ALL) NOPASSWD: /home/linuxbrew/.linuxbrew/bin/coily` (or the equivalent install path). Hosts without that grant fall back to per-unit sudoers and the strict-match failure mode is in play. See `coily-shared-meta` for the host inventory.

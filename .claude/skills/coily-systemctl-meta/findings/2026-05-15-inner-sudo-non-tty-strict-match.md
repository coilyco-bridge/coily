---
date: 2026-05-15
slug: inner-sudo-non-tty-strict-match
promoted_to:
  - anti-signal: "per-unit sudoers carveouts are the gate"
  - anti-signal: "inner sudo works on every host"
  - issue: https://github.com/coilysiren/coily/issues/203
---

# 2026-05-15 - `coily systemctl stop sirens-discord-ops-update.timer` blocked by inner-sudo + strict-match sudoers on non-tty Claude session

## What was observed

While cleaning up the auto-update timer from `sirens-discord-ops` (closing coilysiren/sirens-discord-ops#22) on `kai-server` from a Claude Code Bash session, the three-step cleanup -

```
coily --commit-scope=/home/kai/projects/coilysiren/sirens-discord-ops systemctl stop    sirens-discord-ops-update.timer
coily --commit-scope=/home/kai/projects/coilysiren/sirens-discord-ops systemctl disable sirens-discord-ops-update.timer
coily --commit-scope=/home/kai/projects/coilysiren/sirens-discord-ops systemctl daemon-reload
```

failed on the first line with:

```
sudo: a terminal is required to read the password; either use the -S option to read from standard input or configure an askpass helper
sudo: a password is required
coily: exit status 1
error:
    kind: upstream_failed
    message: exit status 1
    exit_code: 3
```

Reading the live sudoers (`sudo -l`) showed that the `kai-sirens-discord-ops` fragment lists NOPASSWD entries for `sirens-discord-ops.service` (restart, status with and without `--no-pager`, both `/bin` and `/usr/bin` paths) but nothing for `sirens-discord-ops-update.timer`, and nothing for `stop` / `disable` / `daemon-reload` even on the service. So the inner `sudo systemctl stop sirens-discord-ops-update.timer` falls through to the catch-all `(ALL : ALL) ALL` rule, which requires a password.

Meanwhile, the same `sudo -l` showed `(ALL) NOPASSWD: /home/linuxbrew/.linuxbrew/bin/coily` - a broad grant on the coily binary itself. The verb that fails could succeed via `sudo coily systemctl stop ...` from the outside, because the broad grant matches coily-as-a-whole, but the current shape sudo-prefixes systemctl from the inside and the broad grant never gets a chance.

## Why it slipped

The verb's shape (`coily systemctl ...` runs as the invoking user, shells out to `sudo systemctl <verb> <unit>`) was designed when the assumption was "every privileged systemctl invocation has a per-unit NOPASSWD line in the relevant `/etc/sudoers.d/` fragment." That assumption is locally true for the units this fleet has hand-written fragments for (game servers, repo-recall, personal-dashboard, claude-remote-control, sirens-discord-ops) but the fragments cover the units' steady-state verbs (status, restart) not the full closed verb set (start/stop/restart/enable/disable/daemon-reload + status). Operations outside that steady state - decommissioning a timer, daemon-reloading after a unit file change, disabling a service before removal - hit verbs the fragment did not anticipate.

The design intent for coily is that coily itself is the security boundary: audit log, deny list, escape-hatch resistance, argv validation all live in coily. Anything routed through coily is, by definition, a validated invocation. A parallel per-unit sudoers allowlist is not adding a second layer of safety, it is creating a second gate that must agree with the first or the wrapper breaks. When the two disagree, the wrapper has no way to recover, because sudo-on-non-tty cannot prompt.

The failure mode is also tty-coupled: an operator on the Mac at a real terminal would hit a single password prompt and proceed. The same verb from a Claude session, a cron unit, or any systemd-spawned shell dead-ends. The two execution modes silently diverge on what the verb is capable of doing.

## Rule it produced

Two anti-signals:

- **"per-unit sudoers carveouts are the gate."** False. The closed verb set inside coily is the gate. Per-unit NOPASSWD lists are at best redundant, at worst the cause of strict-match drift.
- **"inner sudo works on every host."** False on non-tty sessions when per-unit NOPASSWD is missing.

Forward shape proposed in #203: coily detects non-root invocation of a mutating systemctl verb and re-execs itself via `sudo /home/linuxbrew/.linuxbrew/bin/coily systemctl <verb> <unit>`, relying on the broad `(ALL) NOPASSWD: .../coily` grant. The re-execed root call then runs `systemctl <verb> <unit>` directly with no inner sudo. Audit row is written once. Status verbs keep their existing unprivileged path. Hosts without the broad grant fall back to today's behavior with a clear error.

Boundary stays in coily: the closed verb set, argv validation, deny list, and audit-trail capture all continue to live where they live. The change is in how root is acquired, not in what is allowed.

This finding rhymes with the kubectl-error-lost finding (2026-05-05) and the aws-passthrough-mangles-flags finding (2026-05-13) at a common layer: each is a case where the passthrough shape erodes the boundary instead of strengthening it. A passthrough that prompts for a password on non-tty, mangles trailing flags, or drops downstream stderr is a tax on doing the right thing, and the tax always tempts the operator (or agent) back to the bare tool.

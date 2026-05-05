---
name: coily-ops-gh-meta
description: Meta-improvement skill for the `coily ops gh` pass-through. Encodes anti-signals, sequencing rules, and references for the GitHub CLI wrapper. Write-once observations live in `findings/YYYY-MM-DD-<slug>.md` siblings. Followup state lives on the GitHub issues those findings cite. Distinct from `coily-ops-gh-usage`, the flat lookup of how to invoke the verb today. Use this when adding or removing a gh sub-verb, when changing argv-validation behavior for gh, when reviewing a security-boundary claim about gh, when an audit-log review surfaces a pattern of denied or near-miss gh invocations, or when an incident reveals the gh gate or the gh-wrapper-bypass pattern. Triggers - coily ops gh, gh passthrough, gh wrapper, gh argv validation, gh audit row, gh denials, gh issue create, gh run list, gh api, gh body, gh markdown, jq pipe, claude bypasses gh, gh skill, gh meta-improvement.
---

# coily-ops-gh-meta

Meta-improvement layer for `coily ops gh`. The gh pass-through is high-volume (~1000 audit rows in a 35-day window) and is the single largest source of friction-driven workarounds. Anti-signals here come from real audit-log evidence, not seeded framings.

Composes with: `coily-security-boundary-discipline` (the metachar gate finding cuts across gh, aws, and any other content-flag pass-through), `coily-shared-meta`, `coily-ops-investigation`, `coily-ops-gh-usage`.

## 1. Anti-signals

- **"the wrapper exists, therefore the agent uses it."** False. The agent uses the path of least denial. Raw `gh` denied by Claude Code's permission boundary, without the deny message naming the wrapper, is the path the agent learns. 113 raw `gh` denials in 35d while `coily ops gh` was actively exercised 1000+ times. Lockdown is the mechanism that closes this. Rollout is the gap.
  **Pin:** [findings/2026-05-05-claude-bypasses-coily-gh-wrapper.md](findings/2026-05-05-claude-bypasses-coily-gh-wrapper.md), [coily#61](https://github.com/coilysiren/coily/issues/61).

## 2. Sequencing rules

Generic ops sequencing rules (policy-before-verb, deny-by-default for destructive, remove-policy-in-same-commit) live in `coily-shared-meta` and apply by inheritance.

No gh-specific sequencing rules seeded yet.

## 3. References

- `cmd/coily/ops_gh.go` - cli surface for `coily ops gh`.
- `pkg/policy` - argv-validation gate. Gh metachar rejections account for the dominant gh failure mode.
- `pkg/audit` - audit-row writer. Gh verbs land as `gh.*` (currently) or `ops.gh.*` (post-#50).
- `~/.coily/audit/*.jsonl` - filter `verb` prefix `gh.` for gh rows.
- `findings/` - dated write-once observations.

---
date: 2026-05-12
slug: bare-gh-npm-during-mcporter-install
promoted_to:
  # populated later when a rule lands or an issue is filed:
  # - anti-signal: "<entry text>"
  # - sequencing-rule: "<entry text>"
  # - issue: <url>
---

# 2026-05-12 - agent reached for bare gh and bare npm/npx five times while trying to install mcporter on Linux, instead of routing through coily

## What was observed

No coily audit rows exist for these denials - all five rejections happened upstream of the gate, at the Claude Code Bash deny layer. The agent was asked to install mcporter, emit `mcp-servers/repo-recall.d.ts`, commit, push, and test. Working cwd was `/home/kai/projects/coilysiren` on a Linux host (kai-server, not the canonical Mac workspace).

Sequence of bare-tool reaches in this session, in order:

1. `npm view mcporter` - denied with the generic "Permission to use Bash with command npm view mcporter has been denied" message.
2. `gh repo list coilysiren --limit 100 | grep -i mcp` and `gh search repos mcporter --limit 5` - denied. The agent then correctly switched to `coily ops gh search repos mcporter` (succeeded, found `openclaw/mcporter`).
3. `npm install -g mcporter` - denied (same generic message).
4. `npm config get prefix` - denied.
5. `npx --version; which npx` - denied with the *specific* coily-aware message: "Bash invocation of `npx` circumvents the user's deny rules covering npm/npx/node tooling and routes around the coily lockdown."

Only on the fifth attempt did the deny message explicitly name the coily lockdown. The first four were generic "permission denied" lines that did not name coily as the boundary being bypassed.

After the npx denial the agent finally stopped and reported the situation to Kai. Kai filed this finding.

## Why it slipped

Two distinct gaps.

First, the agent's discipline gap: after the gh denial routed cleanly through `coily ops gh`, the agent did not generalize. The lesson "bare-tool denials route through coily wrappers" was applied to gh and then forgotten one tool over. Bare `npm` was tried as if it were a separate question. The `coily-discipline` skill is explicit on this: "Bare-command denials route through coily wrappers, then `coily audit finding`." The agent satisfied neither half until prompted.

Second, the harness messaging gap. The first four denials returned a generic "permission denied" string with no mention of coily, no routing hint, no "did-you-mean coily ops <area>." Only the fifth (`npx`) carried the coily-aware error text. An agent reading the first four denials in isolation has no signal that coily is the boundary being crossed - it just looks like a permission failure that might be fixable by trying a sibling command (`npm view` -> `npm install` -> `npm config get` -> `npx`). The specific npx message is the right shape. The generic npm-* messages are not.

This is a sibling shape to `2026-05-09-skill-body-points-to-bare-python-not-coily-exec`. Both findings are about denials that happen upstream of the coily gate and therefore leave no audit-log trace. Both are about agent friction caused by opaque or absent routing hints. Difference: there, the skill body misled the agent; here, the harness deny messages did.

There is also a genuine capability gap underneath. `coily.yaml` has no npm / npx / mcporter verb, and mcporter ships only a Mac-arm64 binary plus an npm `.tgz` (no Linux native). Even if the agent had not bare-tool-reached, there was no coily-routed path to install mcporter on this host. The bare-tool reaches are the symptom; the missing coily verb is the underlying gap. Per `kai-brew-release` discipline, the correct default here was fix-coily-first: stop, report, propose the verb. The agent eventually did that but only after the fifth denial.

## Rule it produced

Candidate sequencing rule for `coily-shared-meta` or `coily-discipline`: "After one bare-tool denial routed successfully through a coily wrapper, treat *every* subsequent bare-tool reach in the same session as needing the same routing - do not test sibling commands (`npm view` after `npm install` was denied) to see if the rule is narrower than it looks. One denial generalizes."

Candidate forward action on coilysiren/coily: align the generic npm/npx/node deny message with the npx-specific one. The npx error already names coily as the boundary being bypassed; npm should too. Specifically, the harness deny rule covering `npm/npx/node` tooling should return the same coily-aware text for all four binaries, not just npx. This is one settings-file edit, mechanical.

Candidate forward action on coilyco-ai or coily: decide whether `coily ops mcporter` (or `coily ops npm` as a narrow allowlisted pass-through scoped to mcporter calls) should exist on Linux hosts. If yes, file an issue. If no, document the Mac-only constraint in `tooling-mcp-servers` so the next agent stops earlier.

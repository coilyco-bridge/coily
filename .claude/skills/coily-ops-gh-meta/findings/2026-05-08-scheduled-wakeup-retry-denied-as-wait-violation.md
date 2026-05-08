---
date: 2026-05-08
slug: scheduled-wakeup-retry-denied-as-wait-violation
promoted_to:
  # - issue: <url> (deferred - issue creation is the very thing being denied)
---

# 2026-05-08 - Scheduled-wakeup retry of rate-limited `coily ops gh issue create` denied as "wait boundary violation"

## What was observed

Earlier turn: `coily ops gh issue create` against `coilysiren/coily` failed upstream with a GitHub GraphQL rate-limit (exit_code=3, kind=upstream_failed). Kai said "In an hour, please." Agent registered a `ScheduleWakeup` for 3640s with a self-contained retry prompt.

On wakeup fire, the agent retried verbatim:

    coily --commit-scope=/Users/kai/projects/coilysiren/coily ops gh issue create -R coilysiren/coily -t "..." -b "..."

The harness denied the call with: "User said 'In an hour, please' - agent is retrying the rate-limited issue creation immediately, violating the explicit wait boundary."

Two possibilities, no audit row exists either way (denial is harness-side, before coily was invoked):

1. The wakeup fired earlier than the 1h delay (clock or scheduler bug).
2. The harness's wait-tracking does not treat scheduled wakeups as satisfying user-stated waits, and treats any post-deny retry as immediate regardless of elapsed wall time.

## Why it slipped

The `ScheduleWakeup` tool and the harness's wait-boundary tracker do not appear to share state. The agent's contract with the user ("wait an hour, then retry") was encoded as a 3640s delay on the wakeup, but the harness still classifies the post-wakeup retry as "immediate." From the agent's side this is opaque: the wakeup fired, the prompt asked for the retry, the retry was denied with language that suggests no time passed.

This is the same shape as 2026-05-08-three-bare-gh-denials-before-wrapper-reach (denial language mismatched the actual situation), but at a different layer: there it was "deny doesn't name the wrapper," here it is "deny doesn't acknowledge the wait the user authorized."

## Rule it produced

Open question, not yet a rule: when a user-stated wait is satisfied by a scheduled wakeup, the harness should either (a) treat the wakeup as the wait satisfaction and allow the post-wakeup retry, or (b) the agent's recovery on a "wait violation" deny needs a different shape than just retry-after-wait. Currently the agent has no tool to prove the wait elapsed.

Forward action belongs as an issue on `coilysiren/coily` (or wherever the harness wait-tracker lives), but issue creation itself is what is denied here. Recovery is hand-off to Kai.

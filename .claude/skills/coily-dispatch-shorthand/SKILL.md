---
name: coily-dispatch-shorthand
description: Normalize dictated `coily dispatch` requests into canonical `owner/repo#N` before invoking. Use when Kai dictates a dispatch phrase aimed at a coilysiren repo, especially shapes that mangle in voice ("coily dispatch coily-siren coily-issue 125", "dispatch repo recall issue 42", "fire dispatch on session lattice ticket 17"). Triggers - coily dispatch, dispatch issue, dispatch ticket, fire dispatch, run dispatch, claude -p on issue, fan out issue, voice-dictated GitHub ref aimed at coilysiren/*.
---

# Coily dispatch shorthand

`coily dispatch` is a privileged op (see [coilysiren/coily#136](https://github.com/coilysiren/coily/issues/136)). Mis-parsing a dictated ref silently spawns `claude -p` against the wrong issue. This skill normalizes dictation into canonical `owner/repo#N` before handing off.

## Assumptions

Multi-issue fan-out across a slice happens *before* this skill, not inside it. By the time dispatch-shorthand fires, the issues already exist - `writing-to-issues` or `tooling-sidequest` sliced the work and filed them. This skill's job is narrow: take one dictated reference to one already-open issue and resolve it to a canonical ref. It does not slice work, create issues, or decide what to fan out.

## When to fire

Any user phrase containing "dispatch" plus a numeric tail, especially when the surrounding tokens look like mangled dictation of a GitHub ref. Also fire on "fan out", "spawn", "run claude on" when paired with an issue number.

Do NOT fire when the user has already typed a clean `owner/repo#N` or a GitHub URL - pass straight through to `coily dispatch`.

## Step 1: refresh the registry

`data/repo-registry.md` is regenerated daily by `.github/workflows/sync-repo-registry.yml` (13:00 UTC). The local checkout may lag further. Before resolving, refresh:

```bash
gh api repos/coilysiren/agentic-os-kai/contents/data/repo-registry.md \
  --jq '.content' | base64 -d
```

Read the live copy, not the on-disk file. If `gh` is unreachable, fall back to local with a one-line caveat to Kai.

## Step 2: parse the dictation

Recognize the shape `[coily ]dispatch <owner-tokens> <repo-tokens> [issue|number|ticket|hash|pound|#] <N>`.

**Filler to drop:** "the", "issue", "number", "ticket", "hash", "pound", "on", "for", "please". Also drop a repeated "coily" used as a dictation discourse marker rather than the actual repo name - e.g. "coily-siren coily-issue 125" the second "coily" is filler.

**Owner resolution:** default to `coilysiren` for any of "coily-siren", "coilysiren", "coily siren", "my org", "the org", or omitted. Refuse if a different owner is named - dispatch boundary is `coilysiren/*` only.

**Repo resolution:** lowercase, strip hyphens/spaces, fuzzy-match against the registry's repo column.

## Step 3: known dictation collisions

Bake these in. Voice dictation produces them constantly:

* "coily" alone → `coily` (NOT `agentic-os-kai`). The bare word is the security CLI.
* "coily co ai" / "coily-co-ai" / "coilco ai" / "coily-coai" → `agentic-os-kai`
* "repo recall" / "recall" alone → `repo-recall`
* "eco mods" → `eco-mods` (private superset). Only resolve to `eco-mods-public` if Kai says "public".
* "eco cycle prep" / "cycle prep" → `eco-cycle-prep`
* "eco jobs" / "jobs tracker" → `eco-jobs-tracker`
* "eco MCP" / "eco mcp app" → `eco-mcp-app`
* "eco telemetry" → `eco-telemetry`
* "session lattice" / "lattice" → `session-lattice`
* "gauntlet" → `gauntlet`
* "galaxy gen" → `galaxy-gen`
* "sirens discord" / "discord ops" → `sirens-discord-ops`
* "infra" / "infrastructure" → `infrastructure`
* "website" / "the site" → `website`
* "tap" / "homebrew tap" → `homebrew-tap`

## Step 4: confirm before dispatch

Show the resolved canonical ref plus the issue title from `gh issue view <ref> --json title,state,url`. One line, one-shot confirmation:

> Resolved: `coilysiren/coily#125` - "<title>". Dispatch?

If the match was unique and unambiguous, you may dispatch directly without confirmation. If two repos fuzzy-match, ALWAYS confirm.

## Step 5: refuse and explain

Refuse if:

* Issue state is `CLOSED`.
* Owner is not `coilysiren`.
* Repo did not resolve against the registry (don't guess).
* `gh issue view` errors (issue does not exist).

Refusal should name the failing condition so Kai can re-dictate, not just "could not resolve".

## Step 6: hand off

`coily dispatch` requires an explicit mode (coilysiren/coily#270). Pick one:

```bash
coily dispatch headless    coilysiren/<repo>#<N>   # AFK / fan-out, claude -p, no human eyes
coily dispatch interactive coilysiren/<repo>#<N>   # new Warp tab, focused session, human supervises
```

Bare `coily dispatch <ref>` errors with "specify mode: interactive | headless". When in doubt for a dictated one-off, default to `interactive` (operator has eyes on the result). For a queued backlog sweep, `headless`.

That's the entire surface. Worktree placement, prompt seeding, audit row, ntfy notification - all owned by `coily dispatch` itself.

## Examples

* "coily dispatch coily-siren coily-issue 125" → `coilysiren/coily#125`
* "dispatch coily co ai 313" → `coilysiren/agentic-os-kai#313`
* "fire dispatch on repo recall ticket 108" → `coilysiren/repo-recall#108`
* "dispatch session lattice issue 94" → `coilysiren/session-lattice#94`
* "dispatch eco mods public 17" → `coilysiren/eco-mods-public#17`
* "dispatch the site issue 5" → `coilysiren/website#5`

## Out of scope

* Worktree placement, prompt seeding, audit (owned by `coily dispatch`).
* Authoring the issue body (Kai's job, or a separate scaffolding skill).

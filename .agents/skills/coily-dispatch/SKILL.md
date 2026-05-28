---
name: coily-dispatch
description: Normalize a dictated `coily dispatch` request into a canonical `owner/repo#N`, then pick headless or interactive mode. Use when Kai dictates a dispatch phrase aimed at a coilysiren repo, especially shapes that mangle in voice ("coily dispatch coily-siren coily-issue 125", "dispatch repo recall issue 42"). Triggers - coily dispatch, dispatch issue, dispatch ticket, fire dispatch, run dispatch, claude -p on issue, fan out issue, open one for me, spin this up, HITL this, let me iterate on this, voice-dictated GitHub ref aimed at coilysiren/*.
---

# Coily dispatch

`coily dispatch` is a privileged op (see [coilysiren/coily#136](https://github.com/coilysiren/coily/issues/136)). Mis-parsing a dictated ref silently spawns `claude` against the wrong issue. This skill does two jobs: normalize a dictated reference into a canonical `owner/repo#N`, and pick the dispatch mode (headless or interactive).

## Assumptions

Multi-issue fan-out across a slice happens *before* this skill, not inside it. By the time dispatch fires, the issues already exist - `writing-to-issues` or `tooling-sidequest` sliced the work and filed them. This skill's job is narrow: take one dictated reference to one already-open issue, resolve it to a canonical ref, pick a mode, hand off. It does not slice work, create issues, or decide what to fan out.

## When to fire

Any user phrase containing "dispatch" plus a numeric tail, especially when the surrounding tokens look like mangled dictation of a GitHub ref. Also fire on "fan out", "spawn", "run claude on", and the interactive-intent phrasing ("open one for me", "spin this up", "let me iterate on this", "HITL this") when paired with an issue.

Do NOT fire when the user has already typed a clean `owner/repo#N` or a GitHub URL with no dictation noise - pass straight through to `coily dispatch`, picking the mode per the rules below.

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

## Step 6: pick the mode

`coily dispatch` requires an explicit mode (coilysiren/coily#270): `headless` or `interactive`. Bare `coily dispatch <ref>` errors.

**Default is headless.** By the time Kai dispatches an issue, the design is already done - it happened at the top of the session chain that produced the issue. The dispatched task is pre-decided work: "go execute the thing we already figured out." That does not need her in the loop. The PR is the review gate.

```bash
coily dispatch headless coilysiren/<repo>#<N>   # claude -p, fire-and-forget, no human eyes
```

**Headless detaches by design.** `coily dispatch headless` spawns the child `claude -p` as a detached process and returns within a second - the work survives the terminal closing and runs lights-out. The dispatch call returning is not the work finishing. The dispatch output names the child pid and a log path. If the caller needs to report back when the work actually lands, do not assume a completion signal arrives on its own - watch the pid (`while ps -p <pid> >/dev/null 2>&1; do sleep 15; done`) and read the log tail once it exits.

Pick **interactive** only when one of these holds:

* **Supervision phrasing.** Kai says "open one for me", "let me iterate on this", "spin this up", "HITL this", "give me a session on it" - she is signalling she will have eyes on. Interactive dispatch collapses ~20 seconds of mechanical friction (open URL, new terminal, paste issue) into a new Warp tab cwd'd into the repo with claude pre-submitted and context collection already running.
* **Live decisions remain in the issue.** Thin spec, open design questions, exploratory work - something Kai will want to weigh in on mid-flight. A locked design doc with bounded open questions is headless. A one-line "figure out X" is interactive.

```bash
coily dispatch interactive coilysiren/<repo>#<N>   # new Warp tab, focused session, human supervises
```

**Explicit mode words always win.** If Kai says "headless", "AFK", "interactive", or "supervised", use that and skip the heuristic.

## Step 6b: pick the consult posture (interactive only)

Surface (headless vs interactive) is *where* the run lives. **Consult posture** is *how readily it pauses to involve Kai* - a separate axis selected with `--posture` on `interactive` (coilysiren/coily#130). It is a prompt preamble, not a permission mode: no hard read-only stop like plan mode.

* `--posture watch` (default) - auto mode, Kai may watch but is not consulted. The PR / merge is the review gate. This is the historical interactive behavior.
* `--posture consult` - auto mode with a raised interruption budget. The dispatched agent is encouraged to surface real judgment calls (a naming choice, an irreversible decision, two viable designs) and wait for Kai rather than guess. Still moves by default on everything that does not need her.

```bash
coily dispatch interactive --posture consult coilysiren/<repo>#<N>   # live tab, encouraged to pause and ask
```

Pick `consult` when Kai signals she wants a say mid-flight ("let me weigh in", "ask me before you decide", "check with me on the design") without wanting a hard plan-mode gate. headless never consults by design; the flag is interactive-only.

Worktree placement, prompt seeding, the audit row, the ntfy notification, and (for headless) detaching the child process are all owned by `coily dispatch` itself - not this skill.

## Examples

* "coily dispatch coily-siren coily-issue 125" → `coily dispatch headless coilysiren/coily#125`
* "dispatch coily co ai 313" → `coily dispatch headless coilysiren/agentic-os-kai#313`
* "fire dispatch on repo recall ticket 108" → `coily dispatch headless coilysiren/repo-recall#108`
* "open one for me on session lattice 94" → `coily dispatch interactive coilysiren/session-lattice#94`
* "let me iterate on the site issue 5" → `coily dispatch interactive coilysiren/website#5`
* "dispatch eco mods public 17" → `coily dispatch headless coilysiren/eco-mods-public#17`

## Out of scope

* Worktree placement, prompt seeding, audit, child-process detaching (owned by `coily dispatch`).
* Authoring the issue body (Kai's job, or a separate scaffolding skill).
* Slicing work into multiple issues (`writing-to-issues`, `tooling-sidequest`).

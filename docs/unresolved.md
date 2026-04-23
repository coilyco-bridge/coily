# Unresolved problems and unclear paths

End-of-session scan. Things that are either broken, incomplete, uncertain,
or worth deciding on before the next chunk of work. Each numbered item
lives in its own file under `unresolved/` so a sub-agent array can fan
out one agent per item, each touching a single file.

## Known bugs and rough edges

(none open)

## Incomplete features

(none open)

## Open questions

(none open)

## What I would build next, in order

1. Add a docs/audit.md explaining the log format and a `coily audit tail`
   verb so Kai can review it easily.
2. **Per-directory env var injection in `.coily/coily.yaml`** (direnv slice
   of mise). An `env:` block alongside `commands:` that gets merged into
   the process env when running repo commands. Values go through
   `policy.ValidateArg` so no shell metacharacters sneak in. Scope is the
   repo root, discovered by the same walk that finds `.coily/coily.yaml`.
3. **Dev tool version pinning for repo-command binaries** (asdf slice of
   mise). Repo commands currently resolve binaries via `$PATH` with no
   authenticity check. Add an optional `tools:` block pinning name +
   version + sha256, with a `coily tools sync` verb that fetches into a
   per-user cache (same pattern as the embedded aws/kubectl/gh extraction)
   and prepends the cache to PATH only for the duration of that command.
4. **Task dependencies and pre/post hooks in `.coily/coily.yaml`** (make /
   mise task-runner slice). Today commands are flat. Extend the schema so
   a command can declare `deps: [other-cmd]` and optional `pre:` / `post:`
   steps. Cycle-detect at load time. Audit each step as a separate
   `repo.<cmd>` row so the JSONL still tells the full story.

## Things that are done but deserve skepticism

- **Classifier heuristic**. High-confidence on common cases. Low-
  confidence on the long tail. The per-tool classification snapshot at
  `cmd/subcli-scope/testdata/<tool>.classified.txt` makes a wrong label
  show up in the diff on every fixture refresh, so misses surface during
  review instead of silently dropping the policy gate. See features.md
  test #13 for the full coverage plan.
- **Completion scripts**. The bash/zsh/fish scripts I wrote are standard
  patterns for urfave/cli v3, but I did not verify any of them work
  end-to-end in a live shell. Sub-agent test #9 should catch regressions.
- **HMAC token key lifecycle**. First-use key creation works and perms are
  tight. But key rotation is not built. If Kai wants to invalidate all
  outstanding tokens, they delete the key file, which invalidates
  everything indiscriminately. Finer rotation would need a key version
  field in the token.
- **lockdown defaults.yaml**. I wrote ~80 rules mostly by thinking through
  the threat model. I did not run `coily lockdown --apply` against my
  laptop's real `~/.claude/settings.json` and audit the merged result. It
  might over-deny something Kai needs every day. Sub-agent test #3 covers
  the mechanics but not "are these the right rules".

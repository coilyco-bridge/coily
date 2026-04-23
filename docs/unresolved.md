# Unresolved problems and unclear paths

End-of-session scan. Things that are either broken, incomplete, uncertain,
or worth deciding on before the next chunk of work. Each numbered item
lives in its own file under `unresolved/` so a sub-agent array can fan
out one agent per item, each touching a single file.

## Known bugs and rough edges

1. [Pass-through flag types are all strings](unresolved/01-flag-types.md)
2. [Mutating-verb classifier is prefix-based and will misclassify](unresolved/02-classifier-prefixes.md)
3. [aws s3api help field is groff garbage](unresolved/03-s3api-groff.md)
4. [gh api's --method flag clashes with coily's structure](unresolved/04-gh-api-method.md)
5. [kubectl context "help" is real](unresolved/05-kubectl-help-context.md)
6. [Audit log perms are 0600 but the default path is ~/.local/state](unresolved/06-audit-log-perms.md)
7. [`runtime` is init-time. Test isolation is awkward](unresolved/07-runtime-singleton.md)

## Incomplete features

8. [Embedded sub-binaries](unresolved/08-embedded-binaries.md)
9. [SDK-native ssh/scp/tailscale](unresolved/09-sdk-ssh.md)
10. [`coily eco world`](unresolved/10-eco-world.md)
11. [Self-update v2 + adversarial review](unresolved/11-self-update.md)
12. [Layer 3 integration tests (end-to-end)](unresolved/12-layer3-tests.md)

## Open questions

13. [Should `coily lockdown` require a token?](unresolved/13-lockdown-token.md)
14. [Token scoping granularity](unresolved/14-token-scoping.md)
15. [Where does the coily config actually live?](unresolved/15-config-location.md)
16. [The generated pass-through surface is opinionated by scope](unresolved/17-passthrough-scope.md)

## What I would build next, in order

1. Fix flag types in gen-passthrough (bug #1 above). Highest-utility fix.
2. Extend summary() to handle s3api-style aws help (bug #3).
3. Add a docs/audit.md explaining the log format and a `coily audit tail`
   verb so Kai can review it easily.
4. Pass runtime into verbs explicitly so cmd/coily/ becomes testable (#7).
5. Then: embedded binaries (#8). Big lift but closes the threat model.

## Things that are done but deserve skepticism

- **Classifier heuristic**. High-confidence on common cases. Low-
  confidence on the long tail. See features.md test #13.
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

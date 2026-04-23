# 13. Should `coily lockdown` require a token?

Category: Resolved

Resolution: no token on the automatic path, token only on the destructive
one. The command has three tiers mapped to blast radius:

- `coily lockdown`: ReadOnly. Prints the plan and exits. No token.
- `coily lockdown --apply`: writes only if `.claude/settings.json` does not
  already exist. Refuses on an existing file. No token. This is the
  fresh-repo bootstrap path and should be frictionless.
- `coily lockdown --apply --replace`: overwrites an existing file. Elevated,
  token required. This is the one that can clobber custom allow/deny
  entries the user added by hand.

Why this shape:

- `lockdown` turns on the safety boundary. Putting a token in front of
  "make things safer" is backward. Friction on enabling safety means
  people skip it.
- The current `--apply` behavior silently merges into an existing file,
  which hides changes. Refusing instead makes the user pick a lane:
  bootstrap a fresh repo, or explicitly `--replace` an existing one.
- `--replace` is the only variant that can destroy user config, so it
  gets the Elevated gate.

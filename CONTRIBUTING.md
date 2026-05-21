# Contributing to coily

**coily does not accept external pull requests.** This is deliberate, and it is not about you.

coily is a CLI security boundary. Its whole job is to be a trustworthy gate in front of privileged operations - AWS, kubectl, GitHub, the homelab. Every line of it is load-bearing for that trust. An external pull request to coily is, structurally, a request to modify a security boundary from outside it. Even a well-meant one widens the set of people and processes that can change the gate. The threat model in [SECURITY.md](SECURITY.md) only holds while the answer to "who can change coily" stays small and known.

So the contribution model here is closed on purpose. Forking is encouraged - read it, learn from it, build your own. Changes to this repo come from its maintainer.

## Where to contribute instead

If you want to contribute to this kind of tooling, **[coilysiren/agent-guard](https://github.com/coilysiren/agent-guard)** is the place. agent-guard is the generalizable, contribution-friendly version of the same idea: a generic CLI guard for repos with external contributors. coily delegates its PreToolUse hook to agent-guard. Pull requests are welcome there.

## What is welcome here

- **Bug reports** - open an issue. If coily's boundary has a hole, the maintainer wants to know.
- **Ideas and design feedback** - open an issue.
- **Security findings** - see the reporting section in [SECURITY.md](SECURITY.md).

Issues are always open. It is only pull requests that are closed.

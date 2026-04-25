# coily features inventory

This document enumerates every user-visible feature in coily along with a
testable assertion and a one-line sub-agent test prompt. Structured so that
a sub-agent array can fan out, one agent per feature, each running
independently against a single file.

Every feature entry follows this shape.

- **What it is**: one sentence.
- **How to invoke**: the command you'd type.
- **Expected shape**: what success looks like.
- **Test prompt**: the self-contained prompt to hand a sub-agent.

Do not trust any feature here without running its test. I built a lot in one
session and test coverage for the generated pass-through surface is
effectively zero.

## Index

1. [`coily version`](features/01-version.md)
2. [`coily whoami`](features/02-whoami.md)
3. [`coily lockdown`](features/03-lockdown.md)
4. [`coily aws ...` pass-through](features/05-aws.md)
5. [`coily gh ...` pass-through](features/06-gh.md)
6. [`coily kubectl ...` pass-through](features/07-kubectl.md)
7. [`coily eco status | tail | restart | stop | start`](features/08-eco.md)
8. [`coily install-completion`](features/09-install-completion.md)
9. [`coily skill-gen` (dev only)](features/10-skill-gen.md)
10. [`coily install-skill` (dev only)](features/11-install-skill.md)
11. [subcli-scope extractor + goldens](features/12-subcli-scope.md)
12. [gen-passthrough codegen](features/13-gen-passthrough.md)
13. [Policy / metachar rejection](features/14-policy-metachar.md)
14. [Audit log](features/15-audit-log.md)
15. [Tab completion (runtime)](features/17-tab-completion.md)
16. [Build tags (dev vs prod)](features/18-build-tags.md)
17. [Lint + test + coverage pipeline](features/19-lint-test-coverage.md)
18. [Per-repo command allowlist (`coily.yaml`)](features/20-repo-commands.md)
19. [Skill install chain](features/21-skill-install-chain.md)
20. [Sentry error + log ingest](features/22-sentry.md)

## Non-features: do not test

- Self-update / CI-signed binaries: deliberately unbuilt per SECURITY.md.
- Adversarial review loop: out of scope for v1.
- Embedded sub-binaries (aws/kubectl/gh inside the coily binary): planned but not built. pkg/shell resolves via PATH currently.
- SDK-native ssh/tailscale: planned. Currently ssh shells out.
- Layer 3 end-to-end tests against kind cluster or sandbox AWS: deliberately unbuilt. Scrubbed; not on the roadmap.

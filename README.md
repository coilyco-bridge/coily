# coily

Operator CLI for Kai's homelab (kai-server, coilysiren k3s cluster, and associated AWS/Tailscale resources).

This repo exists for three reasons.

1. **Developer velocity.** Every pass-through verb gets auto-generated exhaustive documentation with every flag known at build time, plus smart defaults (route53 zone IDs discovered via SDK, AWS profile from embedded config, kubectl context pinned to kai-server). All output normalized to yaml. We're all yaml engineers now, might as well own it. One tab-completion install covers `coily aws ...`, `coily kubectl ...`, and `coily gh ...` instead of three separate completion scripts. The same command manifest feeds a generated `SKILL.md` so Claude has a complete offline reference. No round-trip to `aws help`, and no per-tool quirks (aws opens a pager, kubectl doesn't, gh has its own style, coily smooths this out). Flag validation fails fast before the underlying API is ever called.
2. **Safety boundary for AI agents.** See [docs/threat-model.md](docs/threat-model.md). Claude Code and similar agents operating in this environment are granted a narrow allowlist that funnels all privileged operations through `coily`. `coily` is written in Go so it can validate structured arguments, reject shell-metacharacter injection, and re-validate against its own policy independent of the host harness's permission rules.
3. **Auditors love to see me coming.** Every `coily` invocation is appended to a structured JSONL log with session metadata. If something destructive happens there's a row for it. If nothing did, there's a row for that too.

## Install

### Laptop (darwin-arm64)

```
make install           # builds and sudo-installs /usr/local/bin/coily
```

### kai-server (linux-arm64 or linux-amd64)

```
make deploy-server     # cross-compiles, scps to kai-server, sudo-installs
```

### Dev iteration

```
make dev               # builds ./bin/coily-dev (different binary name, not on PATH)
./bin/coily-dev ...    # invoke from repo root only
```

The agent's allowlist only trusts `coily`, never `coily-dev`. Dev builds have `-tags dev` with extra diagnostics. Production builds use `-tags prod` which strips dev code paths.

## Architectural decisions

- **Single binary**, single trust boundary. One entry in the Claude allowlist, `Bash(coily:*)`.
- **Embed `aws`/`kubectl`/`gh`** in the binary via `//go:embed`, extracted per-user to a cache dir with checksum verification. Prevents an agent from substituting `/usr/local/bin/aws` to intercept shell-outs.
- **SDK-native for simple APIs.** ssh/scp (`golang.org/x/crypto/ssh`) and tailscale (`tailscale.com/client/tailscale`). No subprocess means no argv to a shell.
- **Mirror the sub-CLIs exactly.** `coily aws ssm get-parameter` takes the same args as `aws ssm get-parameter`, not a reinvented interface.
- **Config is embedded, not loaded from disk.** Changes require rebuild + sudo install.
- **No self-update** in v1. Updates push from the laptop via `make deploy-server`. The binary cannot rewrite itself. (See docs/threat-model.md for the v2 plan around adversarial-reviewed CI installs.)
- **No `coily shell` / `coily run` escape hatch**, ever.

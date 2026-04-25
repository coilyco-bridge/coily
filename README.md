# coily

Operator CLI for Kai's homelab (kai-server, coilysiren k3s cluster, and associated AWS/Tailscale resources).

A successful read against three different tools, unified into one yaml shape:

```
$ coily whoami
aws:
    account: "..."
    arn: arn:aws:iam::...:user/coilysiren
gh:
    login: coilysiren
    name: Kai
kubectl:
    cluster: kai-server
    current_context: kai-server
```

A rejected invocation. The metacharacter never reaches `aws`:

```
$ coily aws ssm get-parameter --name '/discord/server-id; cat /etc/passwd'
Error: policy: shell metacharacter rejected: arg name contains ";" at index 18
```

Both rows land in the audit log. `decision` distinguishes a coily-side scrub from a downstream failure; `argv` preserves the literal command:

```
$ tail -2 ~/.coily/audit/coilysiren-coily.jsonl
{"decision":"accept","argv":["coily","whoami"],...}
{"decision":"reject","argv":["coily","aws","ssm","get-parameter","--name","/discord/server-id; cat /etc/passwd"],...}
```

This repo exists for three reasons.

1. **Developer velocity.** Every pass-through verb gets auto-generated exhaustive documentation with every flag known at build time, plus smart defaults (route53 zone IDs discovered via SDK, AWS profile from embedded config, kubectl context pinned to kai-server). All output normalized to yaml. We're all yaml engineers now, might as well own it. One tab-completion install covers `coily aws ...`, `coily kubectl ...`, and `coily gh ...` instead of three separate completion scripts. The same command manifest feeds a generated `SKILL.md` so Claude has a complete offline reference. No round-trip to `aws help`, and no per-tool quirks (aws opens a pager, kubectl doesn't, gh has its own style, coily smooths this out). Flag validation fails fast before the underlying API is ever called.
2. **Safety boundary for AI agents.** Claude Code's `deny: "Bash(kubectl delete:*)"` rule is prefix-matched, so every one of these gets past it:

    ```
    sh -c "kubectl delete pod foo"
    echo "kubectl delete pod foo" | sh
    env kubectl delete pod foo
    python -c "import subprocess; subprocess.run(['kubectl','delete','pod','foo'])"
    make delete   # where the Makefile target shells out
    ```

    Denylists are structurally unwinnable against a flexible execution environment. coily's answer is to invert the model: a narrow allowlist (`Bash(coily:*)`) funnels all privileged ops through one Go binary that re-validates structured arguments, rejects shell-metacharacter injection, and enforces its own policy independent of the host harness. Full reasoning in [SECURITY.md](SECURITY.md).
3. **Auditors love to see me coming.** Every `coily` invocation is appended to a structured JSONL log with session metadata. If something destructive happens there's a row for it. If nothing did, there's a row for that too.

## Install

coily itself is never published as a prebuilt binary. Every install is a local build gated by `sudo`, so the install step is always a human (you) compiling from this checkout and placing the result in a root-owned directory. There is no "curl | sh" path and there won't be one.

### Laptop (darwin-arm64)

```
make install           # builds and sudo-installs /usr/local/bin/coily
```

### Laptop (windows-amd64)

```
make install-windows   # builds and installs C:\Program Files\coily\coily.exe
```

Must be run from an elevated shell (Git Bash launched via Ctrl+Shift+Enter, or a PowerShell / cmd "Run as Administrator"). `C:\Program Files\coily\` is admin-write-only by ACL, which is the Windows analog of a root-owned `/usr/local/bin/` on unix - the agent can't overwrite the binary without UAC elevation. Add `C:\Program Files\coily` to `PATH` once after the first install.

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

### What about `aws`, `kubectl`, `gh`?

Those are the one thing coily *does* publish - not coily itself. The `tools-latest` GitHub Release holds pinned `aws` / `kubectl` / `gh` binaries for `darwin/{arm64,amd64}` and `linux/{arm64,amd64}`, refreshed by [`.github/workflows/release-tools.yml`](.github/workflows/release-tools.yml) on a weekly schedule and on workflow_dispatch.

On first use coily reads the embedded [`pkg/shell/tools.yaml`](pkg/shell/tools.yaml) manifest (baked into the binary at build time), fetches the matching binary from `tools-latest`, verifies sha256 against the pin, and caches under `~/.cache/coily/bin/<sha256>/<tool>`. Subsequent runs hit the cache and re-verify the hash before exec. `$PATH` is never consulted for these tools, so an agent who swaps `/usr/local/bin/aws` is ignored. See [SECURITY.md](SECURITY.md) for the full reasoning.

## Per-repo commands (`.coily/coily.yaml`)

Each repo can drop a `coily.yaml` inside a `.coily/` overlay directory at its root to declare the dev commands an operator (human or agent) should run from that tree. `coily test`, `coily lint`, `coily build`. This replaces per-repo Makefiles and `pyinvoke` tasks without widening the security boundary.

```yaml
commands:
  test: go test ./...
  lint:
    run: golangci-lint run ./...
    description: Lint with golangci-lint.
```

- `coily` walks up from the cwd to discover `.coily/coily.yaml`. Run from a subdirectory and it still finds the root. `$COILY_REPO_CONFIG` overrides the walk. A pre-overlay `coily.yaml` at the repo root errors with a pointer at the new path.
- `coily --list` prints built-ins and repo commands, grouped. `coily <cmd> --help` shows what a repo command expands to.
- Every declared token plus any user-supplied extras pass through `policy.ValidateArg`. Shell metacharacters are rejected at load time and at invocation. No carve-outs.
- Audit records use verb `repo.<cmd>`. Same log file as privileged ops.
- Repo commands that collide with a built-in (`aws`, `kubectl`, etc.) are skipped with a stderr warning. Built-ins always win.
- Binaries are resolved via `$PATH` (unlike the pinned `aws`/`kubectl`/`gh`). Repo-level dev tools vary per repo. Their authenticity is the repo's problem, not coily's.

## Architectural decisions

- **Single binary**, single trust boundary. One entry in the Claude allowlist, `Bash(coily:*)`.
- **Pin `aws`/`kubectl`/`gh` by sha256** in an embedded manifest ([`pkg/shell/tools.yaml`](pkg/shell/tools.yaml)). coily fetches the binaries on demand from the `tools-latest` GitHub Release, verifies the hash, and caches under `~/.cache/coily/bin/<sha256>/<tool>`. `$PATH` is never consulted, so an agent substituting `/usr/local/bin/aws` to intercept shell-outs is ignored. The binaries themselves aren't `//go:embed`'d because ~30MB x 3 tools x 4 platforms is too much to ship in-tree.
- **SDK-native for simple APIs.** ssh/scp (`golang.org/x/crypto/ssh`) and tailscale (`tailscale.com/client/tailscale`). No subprocess means no argv to a shell.
- **Mirror the sub-CLIs exactly.** `coily aws ssm get-parameter` takes the same args as `aws ssm get-parameter`, not a reinvented interface.
- **Config is embedded, not loaded from disk.** Changes require rebuild + sudo install.
- **No self-update** in v1. Updates push from the laptop via `make deploy-server`. The binary cannot rewrite itself. (See SECURITY.md for the v2 plan around adversarial-reviewed CI installs.)
- **No `coily shell` / `coily run` escape hatch**, ever.

## Prior art

coily is a personal-scale remix of three existing ideas.

- **[Teleport](https://github.com/gravitational/teleport)** - access broker for SSH, k8s, and cloud APIs with per-session audit. coily keeps the scoped-invocation and JSONL-audit slice, drops the cluster and the web UI.
- **[mise](https://github.com/jdx/mise)** - one CLI multiplexing runtimes, env, and tasks behind consistent verbs. coily applies the same "thin wrapper over N underlying tools" instinct to ops (`aws`, `kubectl`, `gh`, `ssh`) instead of dev envs.
- **[Dagger](https://github.com/dagger/dagger)** - typed, programmable wrapper over container and CI primitives instead of shelled-out pipeline scripts. coily takes the same "validate structured arguments in a real language, don't just shell out" instinct and applies it to `aws`/`kubectl`/`gh`.

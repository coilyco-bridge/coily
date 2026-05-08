# coily

🛡️ coily is a CLI security boundary for privileged ops, 🔒 escape-hatch-resistant and with an 📜 audit trail.

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
$ coily ops aws ssm get-parameter --name '/discord/server-id; cat /etc/passwd'
Error: policy: shell metacharacter rejected: arg name contains ";" at index 18
```

Both rows land in the audit log. `decision` distinguishes a coily-side scrub from a downstream failure; `argv` preserves the literal command:

```
$ tail -2 ~/.coily/audit/coilysiren-coily.jsonl
{"decision":"accept","argv":["coily","whoami"],...}
{"decision":"reject","argv":["coily","ops","aws","ssm","get-parameter","--name","/discord/server-id; cat /etc/passwd"],...}
```

This repo exists for three reasons.

1. **One audited surface for every privileged tool.** External-system pass-throughs (`coily ops aws ...`, `coily ops gh ...`, `coily ops kubectl ...`) live under the `ops` group; standalone pass-throughs (`coily docker ...`, `coily tailscale ...`) plus every package manager nested under `coily pkg` (`coily pkg pnpm`, `coily pkg uv`, `coily pkg cargo`, `coily pkg brew`, ...) all forward verbatim to the underlying binary, gated by argv-level shell-metacharacter rejection and an audit-logged invocation. The pass-through is intentionally thin (`SkipFlagParsing`, no per-leaf subcommand modeling); the upstream tool's own `--help` is the source of truth for verb shape, and the lockdown deny list is the source of truth for read-vs-write gating.

    Package managers all live under the `pkg` namespace:

    ```
    coily pkg pnpm install
    coily pkg uv pip install -r requirements.txt
    coily pkg cargo build --release
    coily pkg brew upgrade
    ```

    Same audit row, same metacharacter rejection. Nothing the upstream tool understands is reshaped on the way through.
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

coily itself is never published as a prebuilt binary. Every install is a local build, either from a checkout or from a tagged source tarball that the user's machine compiles. There is no "curl | sh" path and there won't be one.

The canonical path is `make install` from a checkout: it sudo-installs to a root-owned `/usr/local/bin`, so an agent running unprivileged can't overwrite the binary. A Homebrew tap exists for fresh-machine bootstrap (still build-from-source, no prebuilt artifacts), but installs to user-writable `/opt/homebrew/bin` and so does not preserve the root-owned-binary property. Use brew to bootstrap a new laptop, then switch to `make install` for day-to-day updates.

### Laptop (darwin-arm64)

```
make install           # builds and sudo-installs /usr/local/bin/coily
```

Bootstrap-only alternative on a fresh machine:

```
brew install coilysiren/tap/coily
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

`make test` defaults to [`gotest`](https://github.com/rakyll/gotest), a drop-in `go test` wrapper that colorizes PASS/FAIL lines. Install once with `go install github.com/rakyll/gotest@latest`. To opt out, run `make test GO_TEST="go test"`. CI uses raw `go test` so log output stays clean.

The agent's allowlist trusts `coily`, not `coily-dev`. The rename mostly stops `make dev` from shadowing the installed binary on `$PATH`; the security value is narrow (the workspace deny list already blocks the `go run` path, and `./bin/coily-dev` is not on `$PATH`). Dev builds have `-tags dev` with extra diagnostics. Production builds use `-tags prod` which strips dev code paths.

### What about `aws`, `kubectl`, `gh`?

Resolved via `$PATH` like any other binary. coily used to ship a manifest of pinned binaries fetched from a GitHub Release and verified by sha256, with `$PATH` intentionally bypassed. That machinery is gone: the threat it addressed (an attacker with write to a `$PATH` directory but not to `$HOME`) was a narrow slice that did not justify the release-pipeline + manifest + per-tool refresh cadence. Argv validation, audit logging, and the [lockdown deny list](pkg/lockdown/defaults.yaml) carry the safety boundary now; binary authenticity is the host's problem. See [SECURITY.md](SECURITY.md) for the full reasoning.

## Per-repo commands (`.coily/coily.yaml`)

Each repo can drop a `coily.yaml` inside a `.coily/` overlay directory at its root to declare the dev commands an operator (human or agent) should run from that tree. `coily exec test`, `coily exec lint`, `coily exec build`. This replaces per-repo Makefiles and `pyinvoke` tasks without widening the security boundary.

```yaml
commands:
  test: go test ./...
  lint:
    run: golangci-lint run ./...
    description: Lint with golangci-lint.
```

- `coily` walks up from the cwd to discover `.coily/coily.yaml`. Run from a subdirectory and it still finds the root. `$COILY_REPO_CONFIG` overrides the walk. A pre-overlay `coily.yaml` at the repo root errors with a pointer at the new path.
- `coily --list` prints built-ins and repo commands, grouped. `coily exec <cmd> --help` shows what a repo command expands to.
- Every declared token plus any user-supplied extras pass through `policy.ValidateArg`. Shell metacharacters are rejected at load time and at invocation. No carve-outs.
- Audit records use verb `repo.<cmd>`. Same log file as privileged ops.
- Repo commands sit under `coily exec`, so they cannot shadow built-ins like `aws` or `kubectl`. Pick any name your repo wants.
- Binaries are resolved via `$PATH`. Repo-level dev tools vary per repo. Their authenticity is the repo's problem, not coily's.

## Architectural decisions

- **Single binary**, single trust boundary. One entry in the Claude allowlist, `Bash(coily:*)`.
- **Trust `$PATH` for sub-tool binaries.** coily resolves `aws` / `kubectl` / `gh` / `tailscale` etc. via `exec.LookPath`. An earlier version pinned them by sha256 from a GitHub Release and bypassed `$PATH`; the protection (against an attacker with write to a `$PATH` directory but not `$HOME`) didn't justify the release-pipeline machinery. Argv validation + audit + lockdown deny list carry the boundary instead.
- **SDK-native for simple APIs.** ssh/scp (`golang.org/x/crypto/ssh`) and tailscale (`tailscale.com/client/tailscale`). No subprocess means no argv to a shell.
- **Mirror the sub-CLIs exactly.** `coily ops aws ssm get-parameter` takes the same args as `aws ssm get-parameter`, not a reinvented interface.
- **Config is embedded, not loaded from disk.** Changes require rebuild + sudo install.
- **No self-update** in v1. Updates push from the laptop via `make deploy-server`. The binary cannot rewrite itself. (See SECURITY.md for the v2 plan around adversarial-reviewed CI installs.)
- **No `coily shell` / `coily run` escape hatch**, ever.

## Prior art

coily is a personal-scale remix of three existing ideas.

- **[Teleport](https://github.com/gravitational/teleport)** - access broker for SSH, k8s, and cloud APIs with per-session audit. coily keeps the scoped-invocation and JSONL-audit slice, drops the cluster and the web UI.
- **[mise](https://github.com/jdx/mise)** - one CLI multiplexing runtimes, env, and tasks behind consistent verbs. coily applies the same "thin wrapper over N underlying tools" instinct to ops (`aws`, `kubectl`, `gh`, `ssh`) instead of dev envs.
- **[Dagger](https://github.com/dagger/dagger)** - typed, programmable wrapper over container and CI primitives instead of shelled-out pipeline scripts. coily takes the same "validate structured arguments in a real language, don't just shell out" instinct and applies it to `aws`/`kubectl`/`gh`.

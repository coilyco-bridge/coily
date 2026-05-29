# coily

🛡️ coily is a CLI security boundary for privileged ops, 🔒 escape-hatch-resistant and with an 📜 audit trail.

Operator CLI for Kai's homelab (kai-server, coilysiren k3s cluster, AWS / Tailscale).

```
$ coily whoami                                  # unified view across aws/gh/kubectl
$ coily ops aws ssm get-parameter --name '/x; cat /etc/passwd'
Error: policy: shell metacharacter rejected: arg name contains ";" at index 18
```

Both invocations land in `~/.coily/audit/<repo>.jsonl` as structured rows.

This repo exists for three reasons.

1. **One audited surface for every privileged tool.** External pass-throughs (`coily ops aws/gh/kubectl ...`), standalone pass-throughs (`coily docker`, `coily tailscale`), and package managers (`coily pkg pnpm/uv/cargo/brew`) all forward verbatim, gated by argv shell-metacharacter rejection and an audit row. `coily brew` is a separate top-level for the four mutating brew verbs, scoped to `coilysiren/tap/*` by default.
2. **Safety boundary for AI agents.** Claude Code's `deny: "Bash(kubectl delete:*)"` is prefix-matched and bypassed by `sh -c`, `env`, `python -c subprocess`, Makefile shell-outs, etc. coily inverts the model: a narrow allowlist (`Bash(coily:*)`) funnels privileged ops through one Go binary that re-validates structured args and rejects metacharacter injection. Full reasoning in [SECURITY.md](SECURITY.md).
3. **Auditors love to see me coming.** Every invocation is appended to a structured JSONL log with session metadata.

## Install

coily is never published as a prebuilt binary; every install is a local build. Canonical: `make install` sudo-installs to root-owned `/usr/local/bin`. Brew tap is bootstrap-only (still build-from-source, doesn't preserve root-owned property).

```
make install            # macOS, sudo-install /usr/local/bin/coily
brew tap coilysiren/coily https://forgejo.coilysiren.me/coilysiren/coily && brew install coilysiren/coily/coily
make install-windows    # elevated shell, installs C:\Program Files\coily\coily.exe
make deploy-server      # linux cross-compile + scp + sudo-install on kai-server
make dev                # ./bin/coily-dev for iteration (not on PATH)
```

`make test` defaults to [`gotest`](https://github.com/rakyll/gotest). To opt out: `make test GO_TEST="go test"`. The agent's allowlist trusts `coily`, not `coily-dev`. Dev builds use `-tags dev`, prod uses `-tags prod`.

### `aws`, `kubectl`, `gh` resolution

Via `$PATH` like any other binary. Argv validation + audit + the lockdown deny list carry the safety boundary; binary authenticity is the host's problem. See [SECURITY.md](SECURITY.md).

## Per-repo commands (`.coily/coily.yaml`)

Each repo drops a `coily.yaml` inside `.coily/` to declare its dev verbs. `coily exec test`, `coily exec lint`, etc. Replaces per-repo Makefiles and `pyinvoke` tasks without widening the boundary.

```yaml
commands:
  test: go test ./...
  lint:
    run: golangci-lint run ./...
    description: Lint with golangci-lint.
```

- Walks up from cwd to find `.coily/coily.yaml`. `$COILY_REPO_CONFIG` overrides.
- `coily --list` prints built-ins + repo commands. `coily exec <cmd> --help` shows the expansion.
- Every token passes `policy.ValidateArg`. Metacharacters rejected at load time and at invocation.
- Audit verb is `repo.<cmd>`. Same log file.
- Repo commands sit under `coily exec`, can't shadow built-ins.

## Architectural decisions

- **Single binary, single trust boundary.** One allowlist entry, `Bash(coily:*)`.
- **Trust `$PATH` for sub-tool binaries.** Earlier sha256-pinned manifest dropped: the protection didn't justify the release pipeline.
- **SDK-native for simple APIs.** ssh/scp + tailscale via Go SDKs. No subprocess means no argv to a shell.
- **Mirror the sub-CLIs exactly.** `coily ops aws ssm get-parameter` takes the same args as `aws ssm get-parameter`.
- **Config is embedded.** Changes require rebuild + sudo install.
- **No self-update.** Binary cannot rewrite itself. v2 plan around adversarial-reviewed CI installs in SECURITY.md.
- **No `coily shell` / `coily run` escape hatch.**

## Prior art

coily is a personal-scale remix of [Teleport](https://github.com/gravitational/teleport) (scoped invocation + JSONL audit), [mise](https://github.com/jdx/mise) (one CLI multiplexing many tools), and [Dagger](https://github.com/dagger/dagger) (validate structured args, don't just shell out).

## Contributing

coily does not accept external pull requests - it is a security boundary, and an external change is an attack surface. See [CONTRIBUTING.md](CONTRIBUTING.md); contribute to [ward](https://github.com/coilyco-flight-deck/ward) instead. Issues welcome.

## See also

- [AGENTS.md](AGENTS.md) - agent-facing operating rules.
- [docs/FEATURES.md](docs/FEATURES.md) - inventory of what ships today.
- [.coily/coily.yaml](.coily/coily.yaml) - allowlisted commands.

Cross-reference convention from [coilysiren/agentic-os#59](https://github.com/coilysiren/agentic-os/issues/59).

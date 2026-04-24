# Threat model: why coily exists

This document captures the security rationale behind `coily`. It was written during a Claude Code session on 2026-04-21 after a real incident. A manually-edited `ClusterSecretStore` had drifted from source, silently broke ExternalSecret syncing for the entire cluster, and was only noticed when pods started hitting `CreateContainerConfigError`. The incident itself was benign drift, but it highlighted a broader question. What stops an AI agent (or an attacker via prompt injection against an AI agent) from doing something genuinely destructive?

## Threat model

The agent in question is Claude Code running locally on Kai's laptop, acting against:

- The k3s cluster on `kai-server` (via `kubectl` over tailscale)
- AWS (via `aws` CLI, credentials in `~/.aws/`)
- Tailscale, GitHub, and whatever else shells out through `gh`/`git`/`ssh`

The agent is not assumed hostile. It is assumed to be potentially jailbreakable via prompt injection. Any file it reads, any tool output it processes, any issue body it fetches could contain instructions that steer it into destructive actions. The defensive goal is not to stop a dedicated operator who has direct shell access. It is to bound the blast radius of a confused or prompt-injected agent so that the worst realistic outcome is "session got stuck" rather than "namespace got deleted."

## Why Claude Code's built-in permissions aren't enough

Claude Code supports `allow` / `deny` rules on tool invocations, matched by command prefix. Kai's user settings currently deny `kubectl delete:*`, `kubectl apply:*`, and the rest of the write-family kubectl verbs. See `~/.claude/settings.json`. This is a useful guardrail but not a moat, for four reasons.

### 1. Prefix matching is easy to smuggle past

A `deny: "Bash(kubectl delete:*)"` rule only matches commands whose Bash invocation starts with `kubectl delete`. All of the following sneak past:

```
sh -c "kubectl delete pod foo"
bash -c "kubectl delete pod foo"
echo "kubectl delete pod foo" | sh
sh <<'EOF'
kubectl delete pod foo
EOF
env kubectl delete pod foo
xargs -I{} sh -c "{}" <<< "kubectl delete pod foo"
find . -maxdepth 0 -exec kubectl delete pod foo \;
python -c "import subprocess; subprocess.run(['kubectl','delete','pod','foo'])"
ruby -e "%x(kubectl delete pod foo)"
perl -e 'system("kubectl","delete","pod","foo")'
go run ./throwaway.go   # where throwaway.go shells out
make delete             # where the Makefile target shells out
```

Plus any Node or Deno one-liner that spawns a subprocess. Every one of these starts with a different prefix. Adding more deny rules chases symptoms. Denylists are structurally unwinnable against a flexible execution environment.

### 2. Edits have consequences

The Edit/Write tools aren't covered by bash permission rules at all. An agent that can write files can still cause mutation by:

- Editing a file that cron / launchd / systemd / a file watcher executes
- Dropping a `post-receive` or `pre-commit` git hook
- Modifying a Makefile or deploy script that a human later runs
- Writing into `.claude/`, `CLAUDE.md`, or a plugin source, steering future agent sessions
- Editing `coily` itself, if it lives in a writable checkout

Permissions don't sandbox the consequences of edits, only the edits themselves.

### 3. MCP tools bypass Bash entirely

Loaded MCP servers expose tools that call remote APIs directly without going through the shell. In this environment today, the `aws-eks` MCP server exposes `manage_k8s_resource`, `apply_yaml`, and `add_inline_policy`, each capable of mutating a cluster without invoking `kubectl`. Any `Bash(kubectl ...)` deny rule is irrelevant to those tools. They need their own `deniedMcpServers` or `mcp__aws-eks__*` rules, and those have to be maintained separately as MCP servers come and go.

### 4. Subagents have independent permission evaluation

When the main agent calls the `Agent` tool, a new Claude process starts. It reads the same `settings.json` files, so the existing deny rules do apply to it. But:

- Subagents have independent context windows. The user only sees the final summary. Individual tool calls aren't in the main transcript.
- Subagents run against prompts written by the main agent, not directly by the user. A prompt injection in a file the main agent delegates to a subagent for analysis ("read /tmp/report.md and summarize") now runs with less human oversight than direct tool calls.
- Background subagents can run while the user is away from the terminal.

Denylist rules still bound the damage at the Bash layer, but the agent has more attempts and less scrutiny.

## The allowlist inversion

The structural fix is to invert the list. Instead of enumerating every way to run a dangerous primitive (impossible), enumerate the small set of safe operations and block everything else.

- **Deny broadly.** Every execution-family CLI. The scripting interpreters (python, ruby, perl, deno, node), `go run`, the shells (sh, bash, zsh, plus Windows cmd / powershell / pwsh), Windows scripting hosts and LOLBAS binaries (wscript, cscript, mshta, rundll32, regsvr32), build runners (`make`, package manager scripts), and the direct write verbs on `kubectl`, `helm`, `terraform`, `aws`, `gcloud`, `docker run`. Also deny broad `ssh` except to named hosts. The non-Bash `PowerShell` tool that Claude Code exposes on Windows is denied wholesale, the same way MCP tools are, since it bypasses the Bash prefix matcher entirely.
- **Allow narrowly.** `coily *`, plus read-only tools (ls, grep, cat, git log/diff/status, `kubectl get/describe/logs`), plus editing within explicit directories.

Then `coily` becomes the kernel boundary. Because it's a Go binary the agent cannot edit at runtime (installed from a separate build, not run from source), and because it takes structured flag arguments rather than raw strings, it can:

- Validate inputs against a whitelist of operations
- Reject shell metacharacters (`$`, backticks, `;`, `&&`, `||`, `|`, `>`, `<`, `$(`, `${`, `\`) in any string argument that might reach a shell
- Append every invocation to an audit log (append-only, outside the working tree)
- Re-check permissions against its own config file, loaded from a root-owned path the agent cannot modify
- Prompt for out-of-band confirmation on destructive operations even when the agent's permission check passed

This isn't bulletproof. A dedicated attacker with shell access still has shell access. A bug in `coily` itself could widen the boundary. But it raises the cost of a successful prompt-injection attack from "know that `kubectl delete` is dangerous" to "know Kai's specific coily command surface and find a path through it." That's a meaningful shift in attacker capability required.

## Design guardrails for coily

Principles to preserve as features get added.

- **Installed as a built binary**, not invoked from a writable checkout. The binary lives somewhere root-owned (e.g. `/usr/local/bin/coily`). The source checkout is just for development.
- **Dev builds are a distinct binary (`coily-dev`)** produced only inside the source checkout via `make dev`. The agent's allowlist only trusts `coily`, not `coily-dev`, so dev iterations on Kai's side can't be ridden by an agent. Production builds use `-tags prod` which compiles out any dev-mode conveniences.
- **Config is embedded in the binary at build time** via `//go:embed config.yaml`, not loaded from a path on disk at runtime. Changing config requires editing `config.yaml` in the source checkout, rebuilding, and `sudo install`-ing the new binary, the same review/build/install gate as any code change. This is a stronger bar than a root-owned config file on disk (which required only sudo) and avoids a whole class of "agent convinces a disk-config edit" attacks. Non-secret defaults (tailscale hostnames, etc.) are embedded directly. Secrets come from existing credential stores (`~/.aws/` profile, `~/.kube/config`), not from config.
- **Sub-tool binaries (`aws`, `kubectl`, `gh`) are fetched from a pinned GitHub Release and verified by sha256 before exec.** The coily binary embeds `pkg/shell/tools.yaml`, a small YAML manifest mapping each (tool, goos, goarch) to a download URL and an expected sha256. On first use coily fetches from the `tools-latest` release, verifies the hash, and caches under `~/.cache/coily/bin/<sha256>/<tool>`. Subsequent runs hit the cache and re-verify the hash before exec. PATH is never consulted for these tools, so an agent who swaps `/usr/local/bin/aws` is ignored. The `tools-latest` release is force-overwritten on every prod build by `.github/workflows/release-tools.yml`. Going via GH Releases (rather than `//go:embed` of the binaries themselves) keeps the coily binary small (~30MB per platform per tool x 4 platforms x 3 tools is too much to ship in-tree, and LFS is off the table). Tools with simple APIs (ssh/scp, tailscale) use Go SDKs instead, avoiding any subprocess entirely. ssh is wired through `pkg/ssh` (golang.org/x/crypto/ssh, host keys verified against `~/.ssh/known_hosts`). scp is stubbed in the same package and lights up when a verb needs it. Tailscale SDK (`tailscale.com/client/tailscale`) is not pulled in yet because no verb consumes it. TODO: wire it when an eco verb (or anything else) needs to query tailnet state.
- **Structured args only**. No subcommand takes a free-form string that is later passed to a shell. If a shell-out is absolutely necessary, the Go code uses an explicit argv list, never a composed shell string.
- **Allowlist at the verb level**. `coily k8s restart <deployment>` exists. `coily k8s exec` does not. If a new verb is needed, it's a code change in `coily`, reviewed, committed, built, installed. That review step is the human gate.
- **Append-only audit log** outside the working tree (e.g. `/var/log/coily/audit.jsonl`), writable by the coily process user only. Every invocation logged with timestamp, argv, effective verb, exit code.
- **Destructive verbs require a confirmation token**. Not a TTY prompt. Agents can't use those anyway, and silent auto-confirm is the opposite of what we want. Something like a short-lived token Kai generates with `coily auth issue --ttl 5m --scope k8s-delete` and pastes into the agent's invocation. If the token is missing or expired, the verb refuses.
- **No `coily shell` / `coily run` escape hatch**, ever. The moment one exists, the whole boundary collapses.

## Open questions

- How does `coily` distinguish "Kai at the keyboard" from "Claude in auto mode"? Probably it doesn't need to. The confirmation-token model handles both, and log review happens after the fact regardless.
- How do we handle the `aws-eks` MCP server and other direct-API MCP tools? Resolved 2026-04-21. Kai doesn't use `aws-eks` and is removing it from `~/.claude.json`. A removal script is checked in at `scripts/remove-aws-eks-mcp.sh`. Long-term, expose the operations Kai actually needs through `coily` and keep the MCP removed.
- Interaction with subagents. If `coily` logs include the session ID, we can at least correlate destructive invocations back to specific agent runs for forensics.
- The `Agent` permission rule is currently in allow. That's fine. Subagents inherit the same deny list, so allowing the tool itself doesn't widen the attack surface. Worth revisiting if specific subagents turn out to bypass rules.

## TODO: adversarially-reviewed CI self-update (v2)

The v1 distribution story is "laptop builds, scps to kai-server, sudo installs." That's safe but a big usability hit. The biggest usability hit of the whole design. The v2 plan brings back `coily self-update` without reopening the trust boundary.

Shape:

1. Write new verb in the coily repo on the laptop. Commit, push.
2. **Adversarial review at the commit gate** before anything merges to `main`. A second-opinion agent (ChatGPT via `codex` CLI, or Gemini via `gemini` CLI) reads the diff cold, with no access to the writer's context window, and either approves or blocks. Only approved diffs flow to CI. The reviewer has no ability to modify code, only approve or reject. A prompt injection that steered the writer does not steer the reviewer because the reviewer's context is independent.
3. CI (GitHub Actions) builds + tests + cross-compiles (darwin-arm64, linux-arm64). CI signs the binaries with a private key held only in GitHub Secrets.
4. Signed binaries published to a known location (GH Releases or a private S3 bucket).
5. `coily self-update` fetches latest, verifies the signature against a public key compiled into the running binary, and swaps itself in. Install fails if the signature doesn't verify. Cosign is the usual tool for this.

The adversarial reviewer is the piece that actually defends against prompt injection. Binary signing closes "malicious binary at the distribution URL" even if the storage bucket is compromised, but does nothing against "bad diff written and approved by the same writer." The two mechanisms compose.

Status as of 2026-04-21: Kai wants this but is not setting up codex / gemini today. Revisit once the v1 command surface is useful day-to-day.

## Reference: current Claude Code deny rules

For completeness, the rules currently in `~/.claude/settings.json` (as of 2026-04-21).

**Allow (read + refresh)**: `kubectl get`, `describe`, `logs`, `explain`, `top`, `cluster-info`, `api-resources`, `api-versions`, `version`, `auth can-i`, `diff`, `config view/get-contexts/current-context`, `rollout status/history/restart`.

**Deny (write/update/delete + pod-exec)**: `kubectl apply`, `create`, `delete`, `patch`, `replace`, `edit`, `label`, `annotate`, `scale`, `autoscale`, `set`, `taint`, `cordon`, `uncordon`, `drain`, `expose`, `run`, `rollout undo/pause/resume`, `exec`, `port-forward`, `cp`, `attach`, `proxy`.

These stay in place as the first fence. `coily` is the second fence. The goal is defense in depth, not either/or.

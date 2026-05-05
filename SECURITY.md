# Threat model: why coily exists

> coily is a CLI security boundary for privileged ops, escape-hatch-resistant and with an audit trail.

That's the goal in one sentence. The rest of this document is the rationale behind each of those three properties and the design guardrails that preserve them.

This document was written during a Claude Code session on 2026-04-21 after a real incident. A manually-edited `ClusterSecretStore` had drifted from source, silently broke ExternalSecret syncing for the entire cluster, and was only noticed when pods started hitting `CreateContainerConfigError`. The incident itself was benign drift, but it highlighted a broader question. What stops an AI agent (or an attacker via prompt injection against an AI agent) from doing something genuinely destructive?

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

This isn't bulletproof. A dedicated attacker with shell access still has shell access. A bug in `coily` itself could widen the boundary. But it raises the cost of a successful prompt-injection attack from "know that `kubectl delete` is dangerous" to "know Kai's specific coily command surface and find a path through it." That's a meaningful shift in attacker capability required.

## What about confirmation tokens?

An earlier draft of coily required a short-lived HMAC token (issued via `coily auth issue`, consumed by destructive verbs) on every mutating op. The idea was to force a human-initiated issuance step before any destructive work. That design was removed on 2026-04-24 because it added ritual without adding security:

- The HMAC key lived under `~/.coily/`, readable by the coily process user. An agent could forge tokens directly, never mind calling `auth issue`.
- `coily auth issue` itself was on the allowlist (it had to be, so Kai could invoke it). An agent could call it and consume the output in the same session.
- The token scope check ran inside the same binary the agent was invoking. No out-of-band signer, no separate process.

So the token was a fourth fence made of paper, sitting behind three real ones: the Claude Code deny list, the coily allowlist, and the audit log. Agents could self-authorize trivially, so what the token actually gated was "Kai remembers to type a command first", which is not a security property.

If a genuinely out-of-band confirmation gate is needed later - a yubikey touch, a phone push, a signer running on kai-server that the laptop-agent can't reach - that would have teeth. The previous HMAC design did not, and the ritual was friction without benefit. The allowlist, argv validation, audit log, and Claude Code deny rules are the real fences.

## Design guardrails for coily

Principles to preserve as features get added.

- **Installed as a built binary**, not invoked from a writable checkout. The binary lives somewhere root-owned (e.g. `/usr/local/bin/coily` on unix, `C:\Program Files\coily\coily.exe` on Windows - admin-write-only by default ACL, same boundary property as a root-owned unix path). The source checkout is just for development.
- **Dev builds use a distinct binary name (`coily-dev`).** Produced only inside the source checkout via `make dev`. The agent's allowlist trusts `coily`, not `coily-dev`. The actual security value is narrow: `./bin/coily-dev` is not on `$PATH`, so the agent never finds it anyway, and the workspace deny list already blocks the `go run` path that would invoke the dev source tree. The rename catches one specific footgun (Kai accidentally `cp ./bin/coily-dev /usr/local/bin/coily`) and gives `make dev` something to do that doesn't shadow the installed binary on `$PATH`. Production builds use `-tags prod` which compiles out any dev-mode conveniences.
- **Config defaults are baked into the Go binary**, with optional overlays at `~/.coily/config.yaml` (global) and `./.coily/config.yaml` (per-repo). Earlier coily embedded `config.yaml` via `//go:embed` and pitched that as a security boundary; that claim did not survive scrutiny. An attacker with write to `/etc/coily/` already has root and could replace the binary outright, so embed-vs-disk does not raise the bar. The user-writable overlays are already loaded for the parts of config that matter, so the "agent edits config" path is open for the values that actually shape behavior. Non-secret defaults (tailscale hostnames, audit rotation knobs) live in `pkg/config/config.go`. Secrets come from existing credential stores (`~/.aws/` profile, `~/.kube/config`), not from any config file.
- **Sub-tool binaries (`aws`, `kubectl`, `gh`, `tailscale`, etc.) are resolved via `$PATH`.** An earlier version of coily pinned each by sha256 in an embedded manifest, fetched them on first use from a GitHub Release, and bypassed `$PATH` entirely. That machinery was removed: the threat it addressed (an attacker with write to a `$PATH` directory but not to `$HOME`, where the cache and the coily binary itself live) was a narrow slice that did not justify the release pipeline + manifest + per-tool refresh cadence. The actual safety boundary is argv validation (no shell metacharacters reach the subprocess), the audit log (every invocation is recorded), and the lockdown deny list (raw `aws` / `kubectl` / `gh` are denied at the Bash-tool layer so the agent can only reach them via `coily`). Binary authenticity below that is the host's problem - the same problem `brew` and `apt` and every other package manager have. ssh is still wired through `pkg/ssh` (`golang.org/x/crypto/ssh`, host keys verified against `~/.ssh/known_hosts`) rather than the `ssh` binary, because the SDK is cheap and avoids the argv-to-remote-shell concern entirely.
- **Structured args only**. No subcommand takes a free-form string that is later passed to a shell. If a shell-out is absolutely necessary, the Go code uses an explicit argv list, never a composed shell string.
- **Allowlist at the verb level**. `coily k8s restart <deployment>` exists. `coily k8s exec` does not. If a new verb is needed, it's a code change in `coily`, reviewed, committed, built, installed. That review step is the human gate.
- **Append-only audit log** outside the working tree (e.g. `/var/log/coily/audit.jsonl`), writable by the coily process user only. Every invocation logged with timestamp, argv, effective verb, exit code.
- **No `coily shell` / `coily run` escape hatch**, ever. The moment one exists, the whole boundary collapses. Same rule applies to remote shells: no `coily ssh exec`, no `coily kubectl exec` pass-through. Where free-form remote shell is genuinely needed, the answer is to drop out to raw `ssh` and let the lockdown deny rule force an explicit override, not to wrap a free-form exec inside a coily verb. Named verbs (`coily ssh systemctl status <unit>`, `coily ssh deploy <name>`) cover the legitimate cases without restoring the escape.
- **Every audit row binds to a real repo, no opt-out.** The `--commit-scope` flag (or `$COILY_COMMIT_SCOPE`) is required and cannot be set to `-`, `none`, or `off` (`ErrOptOutRejected`). Default `auto` resolves to `git rev-parse --show-toplevel` of cwd. Verbs that genuinely have no repo to bind to set `verb.Spec.SkipScope = true` at the definition site, so the choice is auditable from source rather than papered over by a per-call flag. The provenance contract is what makes the `Audit-log:` commit trailers a chain of trust instead of a soft hint.

## Open questions

- How does `coily` distinguish "Kai at the keyboard" from "Claude in auto mode"? Probably it doesn't need to. Log review happens after the fact regardless, and the allowlist already bounds what either can do.
- How do we handle the `aws-eks` MCP server and other direct-API MCP tools? Resolved 2026-04-21. Kai doesn't use `aws-eks` and is removing it from `~/.claude.json`. A removal script is checked in at `scripts/remove-aws-eks-mcp.sh`. Long-term, expose the operations Kai actually needs through `coily` and keep the MCP removed.
- Interaction with subagents. If `coily` logs include the session ID, we can at least correlate destructive invocations back to specific agent runs for forensics.
- The `Agent` permission rule is currently in allow. That's fine. Subagents inherit the same deny list, so allowing the tool itself doesn't widen the attack surface. Worth revisiting if specific subagents turn out to bypass rules.
- An out-of-band confirmation gate for destructive ops (yubikey, phone push, remote signer on kai-server) would be a real fence rather than the ritual the old HMAC design provided. Not built; revisit if the allowlist surface grows beyond what's comfortable.

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

## Known boundary holes

Real findings that bound where the boundary actually enforces. New entries land here when the prose-vs-runtime gap is discovered.

- **Claude Desktop on Windows does not enforce the Bash deny list.** Verified 2026-04-23 on Claude Code v2.1.119. Identical repo and `.claude/settings.json` produce different behavior depending on host. The CLI (`claude` in Git Bash) honors `Bash(python:*)` denies; Claude Code inside Claude Desktop (MSIX-packaged agent mode) shows the deny rule loaded under `/permissions` but runs the Bash tool without consulting it. `PowerShell` denies still fire in both hosts because they use a different matcher. Operational implication: lockdown is **CLI-only enforcement** for Bash rules. Agent sessions running from Claude Desktop on Windows effectively run with Bash permissions wide open. Prefer the CLI for any agent work that depends on lockdown for safety.

## Anti-signals

Phrases that survived previous design rounds because nobody tested them. Codified here so the next round is faster.

- **"It's a security boundary because it's plumbed through the gate"** is false unless the gate itself is verified. Plumbing-through is a property of users of the gate, not a property of the gate. A new feature that calls `policy.ValidateArg` does not become part of the boundary just by virtue of the call.
- **"X is an off-host shadow"** is false unless X carries the full record. A summary stream (verb + counts + exit code) is a detection signal, not a shadow. The two have different forensic properties: a shadow lets you reconstruct what happened, a detection signal lets you know that something happened. Don't conflate them in prose.
- **"Drop the feature, then build the replacement"** inverts the right order. If the dropped feature was on the boundary, the boundary degrades during the gap. Build (or accept the loss of) the replacement first, then drop. Otherwise the security claim regresses for every day of the gap.
- **Doc claims must match runtime artifacts.** The security prose in this file describes runtime properties. When prose and runtime drift (a feature shipping less than the prose says), the boundary description silently overstates what's enforced. The `TestSecurityClaims` test in `test/security_claims_test.go` walks each load-bearing claim in this file and asserts it against the actual runtime, so prose-runtime drift surfaces as a test failure rather than as a forensic surprise.

# 5. `coily aws ...` pass-through

**What it is**: Thin pass-through to the system `aws` binary via `pkg/ops/passthrough`. Every invocation goes through argv validation (no shell metacharacters) and lands in the audit log. There is no per-leaf subcommand modeling: `coily aws <args...>` forwards the args verbatim.

**How to invoke**: `coily aws <service> <verb> [flags]`, identical to `aws <service> <verb> ...`. `coily aws --help` shows only the wrapper itself; for upstream help, run `aws --help` directly.

**Expected shape**: Same output as the underlying aws CLI. Flag arguments with shell metacharacters are rejected at the coily boundary; everything else forwards verbatim. Read-only-vs-mutator gating lives in the lockdown deny list (`Bash(aws:*)` deny + verb-level overrides), not inside coily.

**Test prompt**:
> Verify `coily aws sts get-caller-identity` returns the same JSON as `aws sts get-caller-identity`. Verify `coily aws route53 list-hosted-zones` succeeds. Verify passing `coily aws s3 ls 'foo;bar'` fails with a shell-metacharacter rejection before the subprocess runs. Do NOT test any mutating verbs against real AWS.

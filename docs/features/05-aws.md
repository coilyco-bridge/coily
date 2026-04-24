# 5. `coily aws ...` (330+ verbs pass-through)

**What it is**: Mirrors `aws` CLI verbs in scope (route53, s3, s3api, ssm, sts). Every invocation goes through argv validation + audit.

**How to invoke**: `coily aws <service> <verb> [flags]`, same args as `aws <service> <verb> ...`.

**Expected shape**: Same output as the underlying aws CLI. Flag arguments with shell metacharacters are rejected at the coily boundary; everything else forwards verbatim.

**Test prompt**:
> Verify `coily aws sts get-caller-identity` returns the same JSON as `aws sts get-caller-identity`. Verify `coily aws route53 list-hosted-zones` succeeds. Verify passing `--hosted-zone-id 'foo;bar'` to any verb fails with a shell-metacharacter rejection before the subprocess runs. Browse `coily aws --help` and sample 10 read verbs at random across services. For each, assert that running with `--help` shows flags that match the underlying `aws <service> <verb> help`. Do NOT test any mutating verbs against real AWS.

# 5. `coily aws ...` (330+ verbs pass-through)

**What it is**: Mirrors `aws` CLI verbs in scope (route53, s3, s3api, ssm, sts). Every invocation goes through policy + audit.

**How to invoke**: `coily aws <service> <verb> [flags]`, same args as `aws <service> <verb> ...`.

**Expected shape**: Same output as the underlying aws CLI. Read verbs (list-*, get-*, describe-*) run unprompted. Write verbs (create-*, change-*, delete-*) require `--token`.

**Test prompt**:
> Verify `coily aws sts get-caller-identity` returns the same JSON as `aws sts get-caller-identity`. Verify `coily aws route53 list-hosted-zones` succeeds without a token (readonly). Verify `coily aws route53 change-resource-record-sets` fails with "requires a token" unless --token is passed. Browse `coily aws --help` and sample 10 read verbs at random across services. For each, assert that running with `--help` shows flags that match the underlying `aws <service> <verb> help`. Do NOT test any mutating verbs against real AWS.

---
name: coily
description: Operator CLI for Kai's homelab. Use coily instead of direct aws/kubectl/gh/ssh invocations when operating against kai-server, Kai's AWS account (coilysiren), or coilysiren GitHub repos. coily is the only tool authorized for privileged ops in Kai's workspace and audit-logs every invocation.
---

# coily

coily wraps a curated subset of `aws`, `kubectl`, `gh`, and `tailscale`, plus `eco` (systemd on kai-server) and direct `ssh` to kai-server. Every invocation is argv-only (no shell metacharacter injection), policy-checked, and audit-logged.

## When to use

- Any op against kai-server, Kai's AWS account, or `coilysiren/*` GitHub repos.
- Anywhere the reflex would be `aws ...`, `kubectl ...`, or `gh ...`. Prefix with `coily `.
- NOT general-purpose AWS calls from work or other accounts. Use the standard `aws` CLI for those.

## Command shape

`coily <tool> <verb...> [flags]`. Flags mirror the underlying CLI exactly. `coily aws ssm get-parameter --name /foo --with-decryption` is identical in meaning to `aws ssm get-parameter --name /foo --with-decryption`.

## Coily-native verbs

These do not mirror an underlying CLI. They are coily's own operations.

### `coily lockdown`

Write per-repo Claude Code permissions that force all ops through coily.

Flags: --path, --local, --apply, --replace

```
coily lockdown --path . --apply
```

### `coily whoami`

Print the authenticated identity coily sees across aws, kubectl, and gh.

```
coily whoami
```

### `coily version`

Print the build version and exit.

```
coily version
```

## Pass-through tools

For each of these, `coily <tool> ...` takes the same arguments as `<tool> ...` directly. Full verb trees are in this skill's reference directory.

- **`coily aws`** - 323 verbs. Full reference: `reference/aws.md`.
- **`coily gh`** - 80 verbs. Full reference: `reference/gh.md`.
- **`coily kubectl`** - 80 verbs. Full reference: `reference/kubectl.md`.

## Examples

```
coily aws sts get-caller-identity
coily kubectl get pods -A
coily gh run list --repo coilysiren/coily
coily lockdown --path . --apply
```

## What coily will not do

- Open a shell. There is no `coily run` or `coily exec`, ever.
- Take free-form string arguments that reach a shell. Shell metacharacters are rejected at the policy layer.
- Self-update at runtime. Binary updates go through `make deploy-server` from Kai's laptop.

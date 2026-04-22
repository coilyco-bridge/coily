# 17. Tab completion (runtime)

**What it is**: urfave/cli v3 emits completions when invoked with `--generate-shell-completion`.

**How to invoke**: `coily <prefix> --generate-shell-completion` emits colon-separated completions.

**Expected shape**: Each line is `<candidate>:<description>`. Candidates include subcommand names at any depth.

**Test prompt**:
> Run `coily --generate-shell-completion` and assert it emits lines for at least auth/aws/eco/gh/kubectl/lockdown/whoami. Run `coily aws --generate-shell-completion` and assert it emits route53/s3/ssm/sts. Run `coily aws route53 --generate-shell-completion` and assert it emits list-hosted-zones (among others). If any of these shapes are broken, completion is broken.

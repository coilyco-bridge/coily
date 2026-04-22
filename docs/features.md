# coily features inventory

This document enumerates every user-visible feature in coily along with a
testable assertion and a one-line sub-agent test prompt. Structured so that
a sub-agent array can fan out, one agent per feature, each running
independently.

Every feature entry follows this shape.

- **What it is**: one sentence.
- **How to invoke**: the command you'd type.
- **Expected shape**: what success looks like.
- **Test prompt**: the self-contained prompt to hand a sub-agent.

Do not trust any feature here without running its test. I built a lot in one
session and test coverage for the generated pass-through surface is
effectively zero.

---

## 1. `coily version`

**What it is**: Prints the build version injected via `-ldflags`.

**How to invoke**: `coily version`

**Expected shape**: One line of output (git short-SHA, or `dev` for unversioned builds).

**Test prompt**:
> Verify `./bin/coily version` in `/Users/kai/projects/coilysiren/coily/` prints a single non-empty line and exits 0. After rebuilding with `make build`, the output should be either "dev" or a git short hash. Run `git rev-parse --short HEAD` and compare. Report any divergence.

---

## 2. `coily whoami`

**What it is**: Aggregates `aws sts get-caller-identity`, `kubectl auth whoami` (with fallback to config view), and `gh api user` into one yaml block.

**How to invoke**: `coily whoami`

**Expected shape**: Top-level keys `aws`, `kubectl`, `gh`. Each is either a map of identity fields or a map with `error`.

**Test prompt**:
> Verify `./bin/coily whoami` in `/Users/kai/projects/coilysiren/coily/` produces yaml with top-level keys `aws`, `kubectl`, `gh`. For each tool that is authenticated on this host, assert the expected identity fields are present (aws: Account/Arn/UserId; gh: login/id; kubectl: current_context or username). For unauthenticated tools, assert an `error` field is present. Also run it 3 times and assert the audit log at ~/.local/state/coily/audit.jsonl gains 3 entries with verb="whoami".

---

## 3. `coily lockdown`

**What it is**: Writes or merges `.claude/settings.json` with coily's canonical allowlist/denylist.

**How to invoke**:
- `coily lockdown` - dry-run, print plan
- `coily lockdown --apply` - write to `.claude/settings.json`
- `coily lockdown --local --apply` - write to `.claude/settings.local.json`
- `coily lockdown --replace --apply` - clobber existing allow/deny instead of merging

**Expected shape**: Dry-run prints JSON to stdout. Apply writes a settings.json with `permissions.allow`, `permissions.deny`, `deniedMcpServers`. Existing top-level keys preserved. Existing allow/deny merged unless `--replace`.

**Test prompt**:
> In a temp directory, verify `coily lockdown` without flags prints a valid JSON plan. With `--apply` it creates `.claude/settings.json` with 0600 perms. Running it twice does not duplicate entries. Running it against a pre-existing settings.json with a custom allow rule preserves the custom rule and unrelated top-level keys. Running with `--replace --apply` removes the custom rule. Clean up the temp dir when done.

---

## 4. `coily auth issue` + `coily auth verify`

**What it is**: Issues and verifies short-lived HMAC confirmation tokens scoped to a verb.

**How to invoke**:
- `coily auth issue --scope <verb> --ttl <duration>`
- `coily auth verify --scope <verb> --token <token>`

**Expected shape**: Issue prints a base64url token to stdout, TTL + scope note to stderr. Verify exits 0 on valid token, non-zero otherwise. Token expires after TTL. Token for one scope does not verify for another.

**Test prompt**:
> Issue a token via `coily auth issue --scope test.x --ttl 1m`, capture stdout. Verify it with `coily auth verify --scope test.x --token $TOKEN` (exit 0 expected). Verify with wrong scope `test.y` (exit non-zero expected). Issue with a flipped bit in the token (exit non-zero expected). Issue a 1-second token, sleep 2s, verify (exit non-zero expected). Also check that the issuer key file at ~/.local/state/coily/token-issuer.key has 0600 perms.

---

## 5. `coily aws ...` (330+ verbs pass-through)

**What it is**: Mirrors `aws` CLI verbs in scope (route53, s3, s3api, ssm, sts). Every invocation goes through policy + audit.

**How to invoke**: `coily aws <service> <verb> [flags]`, same args as `aws <service> <verb> ...`.

**Expected shape**: Same output as the underlying aws CLI. Read verbs (list-*, get-*, describe-*) run unprompted. Write verbs (create-*, change-*, delete-*) require `--token`.

**Test prompt**:
> Verify `coily aws sts get-caller-identity` returns the same JSON as `aws sts get-caller-identity`. Verify `coily aws route53 list-hosted-zones` succeeds without a token (readonly). Verify `coily aws route53 change-resource-record-sets` fails with "requires a token" unless --token is passed. Browse `coily aws --help` and sample 10 read verbs at random across services. For each, assert that running with `--help` shows flags that match the underlying `aws <service> <verb> help`. Do NOT test any mutating verbs against real AWS.

---

## 6. `coily gh ...` (92+ verbs pass-through)

**What it is**: Mirrors `gh` CLI verbs in scope (api, issue, pr, release, repo, run, search, secret, workflow).

**How to invoke**: `coily gh <verb> [subverb] [flags]`, same shape as gh.

**Expected shape**: Same output as the underlying gh CLI. Reads unprompted, writes (pr merge, issue close, secret set) require `--token`.

**Test prompt**:
> Verify `coily gh api user` returns the same JSON as `gh api user`. Verify `coily gh run list --repo coilysiren/coily` runs and lists runs. Verify `coily gh pr merge` without a token refuses with ErrTokenRequired. Do NOT test writes against real GitHub.

---

## 7. `coily kubectl ...` (54+ verbs pass-through)

**What it is**: Mirrors kubectl verbs. Reads included. Writes (apply, create, delete, patch, replace, edit, label, annotate, scale, etc.) require a token.

**How to invoke**: `coily kubectl <verb> [flags]`, same shape as kubectl.

**Expected shape**: Same output as kubectl for reads. Writes gated by policy.

**Test prompt**:
> Verify `coily kubectl config current-context` returns the same as `kubectl config current-context`. Verify `coily kubectl apply -f /tmp/nonexistent.yaml` without --token refuses with ErrTokenRequired. Verify `coily kubectl get pods` succeeds (readonly). Do NOT test writes against a real cluster.

---

## 8. `coily eco status | tail | restart | stop | start`

**What it is**: Operate the eco-server systemd unit on kai-server via ssh.

**How to invoke**:
- `coily eco status` (readonly)
- `coily eco tail --lines 100 --follow=false` (readonly)
- `coily eco restart --token <token>` (mutating)
- `coily eco stop --token <token>` (mutating)
- `coily eco start --token <token>` (mutating)

**Expected shape**: Reads stream systemd/journalctl output. Writes ssh into kai-server and run `sudo systemctl ...`. Writes without a token fail with ErrTokenRequired.

**Test prompt**:
> Verify `coily eco status` without flags ssh's into kai-server and returns systemctl status output. Verify `coily eco restart` without --token fails with "requires a confirmation token" and never establishes an ssh connection. Issue an eco.restart token and verify that with --token the command reaches the ssh layer (you can stub by setting KAI_SERVER_TAILSCALE_HOST to a non-existent host and asserting the error is "connection refused" / DNS, not a policy error). DO NOT actually restart eco-server in the test.

---

## 9. `coily install-completion`

**What it is**: Writes a shell completion script to the user's home dir.

**How to invoke**:
- `coily install-completion` - auto-detect shell
- `coily install-completion --shell bash|zsh|fish`
- `coily install-completion --dry-run` - print instead of writing

**Expected shape**: Writes `~/.local/share/coily/completion.<shell>` (bash/zsh) or `~/.config/fish/completions/coily.fish`. Prints source instruction to stderr.

**Test prompt**:
> Verify `coily install-completion --shell zsh --dry-run` prints a script containing `compdef _coily_zsh_autocomplete coily`. Verify `coily install-completion --shell bash` (without --dry-run) writes `~/.local/share/coily/completion.bash` and the file is non-empty bash. Verify after sourcing the bash completion, `complete -p coily` shows the completion is registered. Clean up the file when done.

---

## 10. `coily skill-gen` (dev only)

**What it is**: Regenerates `skill/SKILL.md` and `skill/reference/*.md` from `configs/commands/*.yaml`.

**How to invoke**: `./bin/coily-dev skill-gen` from repo root. `--commands-dir` and `--out` flags for custom paths.

**Expected shape**: SKILL.md with frontmatter (`name: coily`, `description: ...`). One reference file per manifest. Deterministic output across runs.

**Test prompt**:
> In the coily repo, run `make skill` and assert `skill/SKILL.md` and `skill/reference/{aws,gh,kubectl}.md` exist. Save the md5 of each. Run `make skill` again and assert the md5s are unchanged (determinism). Assert `coily skill-gen` is NOT in the prod binary (`./bin/coily skill-gen` should fail with "No help topic").

---

## 11. `coily install-skill` (dev only)

**What it is**: Symlinks `./skill/` into `~/.claude/skills/coily/`.

**How to invoke**: `./bin/coily-dev install-skill [--force]` from repo root.

**Expected shape**: Creates a symlink. Refuses to clobber existing without `--force`.

**Test prompt**:
> Run `make install-skill` from the repo. Assert `~/.claude/skills/coily` is a symlink pointing at `<repo>/skill`. Assert `~/.claude/skills/coily/SKILL.md` is readable through the symlink. Assert `./bin/coily install-skill` (prod binary) fails with "No help topic".

---

## 12. subcli-scope extractor + goldens

**What it is**: Walks aws/gh/kubectl help output and emits `configs/commands/<bin>.yaml`. Scoped by `configs/scopes/<bin>.yaml`.

**How to invoke**:
- `make scope-aws | scope-gh | scope-kubectl | scope-all` - regenerate manifests from live tools
- `make update-fixtures` - recapture help fixtures, refresh goldens, regen pass-through, regen skill

**Expected shape**: `configs/commands/*.yaml` files with `binary`, `bin_version`, `scanned_at`, and nested `commands` with paths/flags/help. Golden tests verify parser output matches committed goldens when run against fixtures.

**Test prompt**:
> In the coily repo, run `go test ./cmd/subcli-scope` and assert all tests pass. Run `go test -cover ./cmd/subcli-scope` and assert coverage is >50%. Then manually inspect `cmd/subcli-scope/testdata/aws.golden.yaml` and spot-check 3 random commands: their `help` field should be a clean English sentence, not groff garbage ("FOO()      FOO()"), not a docutils warning ("<string>:... (WARNING/2)").

---

## 13. gen-passthrough codegen

**What it is**: Reads `configs/commands/<bin>.yaml` and emits `pkg/ops/<bin>/generated.go` with a `Command()` function returning a nested `*cli.Command` tree.

**How to invoke**: `make gen-passthrough` or `go run ./cmd/gen-passthrough <bin>`.

**Expected shape**: Each leaf verb has an Action wrapped via `verb.Wrap`, with Kind set by the classifyVerb heuristic (prefix-based). String flags for every known flag. Runs `r.Exec(ctx, bin, argv...)` to the underlying tool.

**Test prompt**:
> In the coily repo, run `make gen-passthrough` and assert `go build ./pkg/ops/...` compiles. Then inspect `pkg/ops/aws/generated.go` and find 5 mutating verbs (change-*, create-*, delete-*, modify-*, put-*) and assert they all set `Kind: policy.Mutating`. Find 5 read-only verbs (list-*, get-*, describe-*) and assert `Kind: policy.ReadOnly`. If any of these are wrong, the classifier in cmd/gen-passthrough/main.go classifyVerb needs a fix. Flag the mis-classifications in the report.

---

## 14. Policy / metachar rejection

**What it is**: pkg/policy.ValidateArg rejects strings containing shell metacharacters before the subprocess layer ever sees them.

**How to invoke**: triggered automatically by every verb going through `verb.Wrap`. Tested directly via `go test ./pkg/policy/`.

**Expected shape**: Any verb invoked with an injection-shaped argument refuses with `policy: shell metacharacter rejected`.

**Test prompt**:
> In the coily repo, run `go test -v ./pkg/policy/` and assert all tests pass. Then build coily and try: `coily lockdown --path 'foo;rm -rf'`. Assert it refuses with a policy error and does NOT actually execute the semicolon-split command. Try 5 more shaped injections from the ShellMeta charset. None should succeed.

---

## 15. Audit log

**What it is**: Every verb invocation is appended as one JSONL line to `~/.local/state/coily/audit.jsonl` with timestamp, verb, argv, exit code, duration, and session id.

**How to invoke**: implicit. Check the log file after running anything.

**Expected shape**: JSONL, one record per invocation, 0600 perms, parent dir 0700.

**Test prompt**:
> Delete `~/.local/state/coily/audit.jsonl`. Run `coily whoami`, `coily version`, and `coily aws sts get-caller-identity`. Assert the log file now exists with 3 records. Each record has non-empty `ts`, `verb`, `argv`, and `exit_code=0`. Invoke something that will fail (e.g. `coily lockdown --path /nonexistent/dir/that/cant/be/mkdir-d` or `coily aws sts get-caller-identity` with AWS_PROFILE=bogus) and assert the new record has `exit_code=1` and a non-empty `error` field.

---

## 16. HMAC token issuer + key file lifecycle

**What it is**: pkg/auth issues tokens signed with an HMAC-SHA256 key stored at `~/.local/state/coily/token-issuer.key`. Created on first use with 0600 perms and 32 random bytes.

**How to invoke**: indirect via `coily auth issue` / `coily auth verify`.

**Expected shape**: Key file 0600, 32+ bytes. Token verify cross-issuer fails. Token verify with tampered signature fails.

**Test prompt**:
> Delete `~/.local/state/coily/token-issuer.key`. Run `coily auth issue --scope x --ttl 1m`. Assert the key file now exists, 0600 perms, ≥32 bytes. Save the token. Delete and recreate the key file. Run verify against the original token. It MUST fail (keys differ). Restore the original key (save it before deleting). Verify should now succeed.

---

## 17. Tab completion (runtime)

**What it is**: urfave/cli v3 emits completions when invoked with `--generate-shell-completion`.

**How to invoke**: `coily <prefix> --generate-shell-completion` emits colon-separated completions.

**Expected shape**: Each line is `<candidate>:<description>`. Candidates include subcommand names at any depth.

**Test prompt**:
> Run `coily --generate-shell-completion` and assert it emits lines for at least auth/aws/eco/gh/kubectl/lockdown/whoami. Run `coily aws --generate-shell-completion` and assert it emits route53/s3/ssm/sts. Run `coily aws route53 --generate-shell-completion` and assert it emits list-hosted-zones (among others). If any of these shapes are broken, completion is broken.

---

## 18. Build tags (dev vs prod)

**What it is**: `coily-dev` (built with `-tags dev`) contains `skill-gen` and `install-skill`. `coily` (prod, `-tags prod`) does not.

**How to invoke**: `make build` produces prod. `make dev` produces `coily-dev` in `./bin/`.

**Expected shape**: `./bin/coily skill-gen` fails with "No help topic". `./bin/coily-dev skill-gen` works.

**Test prompt**:
> Run `make build` and `make dev`. Assert both binaries exist in `./bin/`. Assert `./bin/coily skill-gen 2>&1` contains "No help topic". Assert `./bin/coily-dev skill-gen --help` succeeds.

---

## 19. Lint + test + coverage pipeline

**What it is**: `make test`, `make lint`, `make cover`. golangci-lint is configured for cyclomatic complexity limits (cyclop, gocyclo, gocognit, nestif), security (gosec), style (goimports, misspell), and correctness (errcheck, ineffassign, staticcheck).

**How to invoke**:
- `make lint` - should be clean
- `make test` - should pass
- `make cover` - reports coverage, prints HTML hint

**Expected shape**: Zero lint issues. All tests pass. pkg/* coverage >90% on hand-written packages.

**Test prompt**:
> In the coily repo, run `make lint` and assert zero issues. Run `make test` and assert all tests pass. Run `make cover` and report the coverage number per package. Flag any pkg/* below 80%.

---

## 20. Skill install chain

**What it is**: The generated skill at `<repo>/skill/` is symlinked into `~/.claude/skills/coily/`. Regenerations via `make skill` propagate automatically through the symlink.

**How to invoke**: `make install-skill` once. After that, `make skill` updates the live skill.

**Expected shape**: `~/.claude/skills/coily` is a symlink. `~/.claude/skills/coily/SKILL.md` has valid frontmatter.

**Test prompt**:
> Verify `~/.claude/skills/coily` is a symlink to `<coily-repo>/skill`. Cat its SKILL.md and assert the frontmatter has `name: coily` and a non-empty `description:`. Then `make skill` in the repo, re-cat the skill, assert the file is still readable and the content may have changed.

---

## Non-features: do not test

- Self-update / CI-signed binaries: deliberately unbuilt per docs/threat-model.md.
- Adversarial review loop: out of scope for v1.
- Embedded sub-binaries (aws/kubectl/gh inside the coily binary): planned but not built. pkg/shell resolves via PATH currently.
- SDK-native ssh/tailscale: planned. Currently ssh shells out.
- Layer 3 end-to-end tests against kind cluster or sandbox AWS: deliberately unbuilt. See docs/unresolved.md.

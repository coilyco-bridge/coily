# Unresolved problems and unclear paths

End-of-session scan. Things that are either broken, incomplete, uncertain,
or worth deciding on before the next chunk of work.

## Known bugs and rough edges

### 1. Pass-through flag types are all strings

Every generated flag in `pkg/ops/{aws,gh,kubectl}/generated.go` is a
`cli.StringFlag`, even for things that should be `BoolFlag`, `IntFlag`, or
`StringSliceFlag`. Consequence: boolean flags like `--debug` or
`--no-verify-ssl` will take a value the user didn't supply. The underlying
tool still works because coily forwards the flag pair `--foo <value>` only
when `c.IsSet("foo")` is true, but the UX is off. Fix: extend the
subcli-scope manifest with flag types (parse from help text shape like
"`--foo` (boolean)" or "`--foo` <value>") and have gen-passthrough emit the
right flag kind.

### 2. Mutating-verb classifier is prefix-based and will misclassify

`cmd/gen-passthrough/main.go classifyVerb` uses a hand-maintained prefix
list. New AWS services will ship verbs with prefixes we haven't thought of
("promote-", "publish-", etc.). Kubernetes alpha verbs might slip through.
Consequence: a mutating verb classified as ReadOnly silently loses token
gating. Worth auditing once a week per `make update-fixtures` output diff.
Flag in the features.md test plan for feature #13.

### 3. aws s3api help field is groff garbage

The s3api group still shows `S3API()    S3API()` in its help field
because its help text does not have a DESCRIPTION section in the normal
form. Cosmetic. Fix: extend `summary()` in cmd/subcli-scope/main.go to
pull from SYNOPSIS or NAME sections when DESCRIPTION is absent.

### 4. gh api's --method flag clashes with coily's structure

`gh api --method POST /graphql` is a common shape. Our generated
coily wrapper passes it through correctly, but if the user invokes it
without `--method` (relying on GET default), IsSet returns false and we
don't forward. That's fine. But certain flags like `--field key=value` may
pass through incorrectly with a single c.String value because they're
repeatable and we only capture the last one. Same fix as #1 (flag types,
including StringSliceFlag).

### 5. kubectl context "help" is real

`kubectl config current-context` on Kai's laptop returns the word "help"
because there is a context literally named "help" in his kubeconfig.
Probably a leftover from `kubectl config use-context help` somewhere.
Worth cleaning up with `kubectl config delete-context help` once Kai is
sure it is unused.

### 6. Audit log perms are 0600 but the default path is ~/.local/state

If the config specifies `/var/log/coily/audit.jsonl` coily will fail to
write because that path needs root. Runtime silently falls back to the
configured path and just fails the write with stderr warnings per call.
Consider: either make the fallback loud, or default the config to
`~/.local/state/coily/audit.jsonl`. Right now `config.yaml` uses
`~/.local/state/coily/audit.jsonl` which is correct for personal laptop
use but not for a multi-user deploy.

### 7. `runtime` is init-time. Test isolation is awkward

`cmd/coily/runtime.go` builds a singleton on first call. Tests that want
to exercise `cmd/coily/ops_*.go` can't easily swap in a fake audit writer
or verifier. Consequence: there are no tests in `cmd/coily/`. Fix: pass
the runtime in explicitly rather than globally, probably as a Runner
struct method or a context value. Not urgent, but a real gap.

## Incomplete features

### 8. Embedded sub-binaries

Threat model says `aws/kubectl/gh` should be embedded into coily via
`//go:embed`, extracted to a per-user cache with checksum verification.
Currently `pkg/shell.PathResolver` just does exec.LookPath. The guarantee
"an agent cannot swap /usr/local/bin/aws" is not enforced. Implementation
steps:

1. Download aws-cli-v2, kubectl, gh binaries for each target platform at
   build time (or script-time).
2. `//go:embed` them with build constraints per GOOS/GOARCH.
3. Extract to `$XDG_CACHE_HOME/coily/bin/<sha256>/<tool>` on first use,
   verify sha256, exec from there.
4. Skip extraction if the cached copy checksum-matches.

Blocker: aws CLI v2 is ~30MB per platform. 4 platforms × 3 tools = 12
embedded binaries, ~100MB per final coily binary per platform. Needs
either LFS or a per-platform build + release artifact.

### 9. SDK-native ssh/scp/tailscale

Threat model says these simple-API tools should use Go SDKs instead of
shelling out. Currently eco verbs shell out to `ssh` via pkg/shell. Not
security-critical because ssh's argv surface is small and we construct it
from compile-time constants. But it is a divergence from the stated
design. Implementation:

- ssh/scp: `golang.org/x/crypto/ssh` + `github.com/bramvdbogaerde/go-scp` or similar.
- tailscale: `tailscale.com/client/tailscale`. Currently unused - no coily
  verb consumes it yet.

### 10. `coily eco world`

The survey mentioned "world generation" and seed manipulation for the eco
server. Not built. Design open: is it a single verb that does the full
teardown/regen sequence or three smaller verbs (stop, regen, start)?
Defer until Kai wants it.

### 11. Self-update v2 + adversarial review

Documented as TODO in docs/threat-model.md. Not built. Requires
decisions:

- Which second-opinion tool (codex CLI, gemini CLI, a separate Claude Code
  session)?
- How does the adversarial reviewer gate actually block a merge? GitHub
  Actions + branch protection? A custom git hook?
- Where do signed binaries live? Public GH Releases would make coily
  trivially available to anyone. Private release repo avoids that.
- How does `coily self-update` know the correct URL?

Kai explicitly deferred this on 2026-04-21.

### 12. Layer 3 integration tests (end-to-end)

Described in my earlier Layer 1/2/3 proposal. Layer 1 (fixture golden) and
Layer 2 (whoami smoke) are built. Layer 3 needs:

- A kind cluster for kubectl verbs.
- A dedicated AWS test prefix (`/coily/test/*` SSM, a test Route53 zone).
- A sandbox GitHub repo for gh verbs.

Punted. Not urgent because the pass-through layer is thin and unit tests
on pkg/* cover the interesting logic.

## Open questions

### 13. Should `coily lockdown` require a token?

Currently ReadOnly. It writes local files (`.claude/settings.json`), which
feels like a write operation. But the file it writes is *the* safety
boundary, and requiring a token to turn on the boundary is awkward. My
choice was ReadOnly + require `--apply` as the explicit gate. Reconsider
if an agent is ever observed running `coily lockdown --apply --replace`
in an unexpected context.

### 14. Token scoping granularity

Current: each verb has a scope like `aws.route53.change-resource-record-sets`.
Alternative: broader scopes like `aws.route53.write` that cover multiple
verbs. Narrower is more annoying for Kai (more tokens to issue). Broader
is weaker (a token for "any route53 write" includes "delete-hosted-zone"
which is much nastier than "upsert one record").

Current is narrower. May want a `--scope aws.route53.*` wildcard mode
in `coily auth issue`. Not built.

### 15. Where does the coily config actually live?

Right now the config.yaml is committed in the repo root (gitignored at
/config.yaml, separately committed as /config.example.yaml). The Makefile
copies it into pkg/config/ before build. Kai's personal copy carries
paths like `~/.local/state/coily/audit.jsonl`.

If coily is ever installed on kai-server (linux ARM), should the same
config apply? Or does the server need its own config? Current Makefile
target `make deploy-server` carries Kai's laptop config over to the
server verbatim. Might want a `make deploy-server CONFIG=server.yaml`
override.

### 16. Subagent ID propagation

Audit log reads `CLAUDE_SESSION_ID` from env. Claude Code does not
actually set this env var today (as far as I know). Consequence:
`session_id` in audit records is always empty. Fix options:

- Hook into Claude Code's hooks system to inject CLAUDE_SESSION_ID.
- Give up on session correlation and use invocation timestamps instead.
- Use a different env var that Claude Code does set (TERM_PROGRAM?
  process ancestry check?).

### 17. The generated pass-through surface is opinionated by scope

`configs/scopes/aws.yaml` allows only route53/s3/s3api/ssm/sts. If Kai
suddenly needs `aws lambda list-functions`, they have to edit the scope
yaml, `make update-fixtures`, `make install`. That's a multi-minute
flow for a one-off query. Maybe a `coily aws raw ...` escape hatch? But
that defeats the whole design. Maybe a "bring your own aws" allowlist in
config where the user can add services without rebuilding.

## What I would build next, in order

1. Fix flag types in gen-passthrough (bug #1 above). Highest-utility fix.
2. Extend summary() to handle s3api-style aws help (bug #3).
3. Add a docs/audit.md explaining the log format and a `coily audit tail`
   verb so Kai can review it easily.
4. Pass runtime into verbs explicitly so cmd/coily/ becomes testable (#7).
5. Then: embedded binaries (#8). Big lift but closes the threat model.

## Things that are done but deserve skepticism

- **Classifier heuristic**. High-confidence on common cases. Low-
  confidence on the long tail. See features.md test #13.
- **Completion scripts**. The bash/zsh/fish scripts I wrote are standard
  patterns for urfave/cli v3, but I did not verify any of them work
  end-to-end in a live shell. Sub-agent test #9 should catch regressions.
- **HMAC token key lifecycle**. First-use key creation works and perms are
  tight. But key rotation is not built. If Kai wants to invalidate all
  outstanding tokens, they delete the key file, which invalidates
  everything indiscriminately. Finer rotation would need a key version
  field in the token.
- **lockdown defaults.yaml**. I wrote ~80 rules mostly by thinking through
  the threat model. I did not run `coily lockdown --apply` against my
  laptop's real `~/.claude/settings.json` and audit the merged result. It
  might over-deny something Kai needs every day. Sub-agent test #3 covers
  the mechanics but not "are these the right rules".

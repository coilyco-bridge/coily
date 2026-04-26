# 23. External-orchestrator surface

**What it is**: A small, stable contract for an external agent orchestrator (or any non-interactive consumer) to drive coily, distinguish failure classes, and follow the audit trail.

**Pieces**:

- **Documented exit-code enum** (`pkg/exitcode`):
  - `0` Success — verb ran, underlying tool returned 0.
  - `1` Generic — uncategorized failure (legacy default).
  - `2` PolicyDenied — coily's pre-flight rejected the call (shell-meta validation, etc). Underlying tool was never invoked.
  - `3` UpstreamFailed — underlying tool ran and exited non-zero.
  - `4` Internal — coily-internal failure (config load, manifest miss, audit-write fail).
  - `5` UserError — wrong arg count, missing flag, or other user input error that wasn't a metacharacter reject.

- **YAML error envelope on stderr** for every non-zero exit:
  ```yaml
  error:
    kind: policy_denied        # stable token; see exitcode.Kind()
    message: "policy: shell metacharacter rejected: ..."
    hint: "argv contains a shell metacharacter that coily refuses to forward"
    exit_code: 2
    audit_log_path: /Users/kai/.local/state/coily/audit.jsonl
    timestamp: 1777244025
  ```
  The human-readable error line (`coily: ...`) precedes the envelope so it stays visible in interactive use; the envelope follows for programmatic consumers.

- **`COILY_AUDIT_LOG` env var**: highest-precedence override of the audit log path (wins over file config and the default). Lets an orchestrator point coily at its own log dir without writing a config file.

- **`coily audit path`**: prints the resolved audit log path. One-shot discovery for an orchestrator that doesn't want to mirror the resolution rules itself.

- **`coily audit tail [--follow] [--since <ts|duration>]`**: streams the JSONL log. `--since` accepts unix seconds or a relative duration (`5m`, `1h`); records older than that are skipped. Output is exactly the on-disk JSONL so any JSON parser works.

- **`coily lockdown skill --format yaml`**: structured form of the command tree. Emits to `skills/coily-passthroughs/commands.yaml`:
  ```yaml
  commands:
    - path: [coily, aws, ssm, get-parameter]
      summary: Read one parameter.
      flags: [--name, --with-decryption]
  ```
  Same walker as the markdown SKILL.md (consumed by Claude Code), so the two outputs are diff-able against each other.

**Out of scope** (deferred until a real consumer needs them):

- Network-exposed coily endpoints. Stays local CLI.
- Multi-tenant deployment. Single-host.
- Named lockdown profiles (`--profile orchestrator-strict`). Defer until a second profile is needed; building the abstraction with one consumer usually produces the wrong shape.

**Why these and not more**: the goal is to give an external consumer enough hooks to integrate without locking coily into a wider contract than it needs. Exit codes, error envelope, audit discovery, and structured command tree are the minimum that makes the audit log + retry/handoff loop legible. Everything else can land when the consumer that needs it is real.

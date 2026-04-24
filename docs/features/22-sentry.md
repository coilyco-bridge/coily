# 22. Sentry error + log ingest

**What it is**: When `SENTRY_DSN` is set, coily forwards unhandled errors to Sentry as Issues and emits one structured log line per verb invocation (verb, argc, exit_code, duration_ms, host, error). Log traffic reuses the local audit log's `Writer.OnRecord` callback, so the summary shipped to Sentry mirrors what landed in `~/.coily/audit/<owner>-<repo>.jsonl`. Raw argv is deliberately not sent; Sentry gets the verb name and an argument count, and the local JSONL is ground truth for the full argument list.

When `SENTRY_DSN` is empty, every telemetry call is a no-op and coily makes no network traffic.

**How to invoke**: implicit once the env var is set. The Homelab Makefile / deploy wrapper should hydrate `SENTRY_DSN` at runtime from SSM:

```bash
export SENTRY_DSN=$(aws ssm get-parameter \
  --name /sentry-dsn/coily \
  --with-decryption \
  --query Parameter.Value \
  --output text)
```

On Windows/Git Bash, prefix the `aws ssm get-parameter` call with `MSYS_NO_PATHCONV=1` so MSYS does not rewrite the `/sentry-dsn/coily` leading-slash argument into a Windows path.

Optional: `COILY_ENV` overrides the Sentry environment tag (default: `production`). Release is tagged `coily@<Version>` where `Version` is the value injected at build time via `-ldflags`.

**Expected shape**:

- https://coilysiren.sentry.io under the `coily` project has a new log entry within ~30s of a successful invocation. The log body is `coily <verb> exit=<n>` with `verb`, `argc`, `exit_code`, `duration_ms`, and `host` as structured attributes.
- A deliberately failing invocation both emits a log (with `error` set and `exit_code=1`) and creates an Issue whose exception message matches the returned error.
- With `SENTRY_DSN` unset, `coily version` completes with no outbound network traffic to ingest.sentry.io.

**Test prompt**:

> Export `SENTRY_DSN=$(aws ssm get-parameter --name /sentry-dsn/coily --with-decryption --query Parameter.Value --output text)` (prefix `MSYS_NO_PATHCONV=1` on Windows). Run `coily version` twice. Confirm a log entry appears at https://coilysiren.sentry.io under `coily > Logs` within 30s with `verb=version` and `exit_code=0`. Then force a failure (e.g. `COILY_CONFIG=/nonexistent coily whoami` or an equivalent path the repo already uses for failure tests) and confirm the next log entry has `exit_code=1` and non-empty `error`, and that a corresponding Issue opens in the same project. Finally, `unset SENTRY_DSN` and run `coily version` again, confirming with `lsof -i` or equivalent that the binary makes no outbound TCP connection to `ingest.sentry.io`.

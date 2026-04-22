# 15. Audit log

**What it is**: Every verb invocation is appended as one JSONL line to `~/.local/state/coily/audit.jsonl` with timestamp, verb, argv, exit code, duration, and session id.

**How to invoke**: implicit. Check the log file after running anything.

**Expected shape**: JSONL, one record per invocation, 0600 perms, parent dir 0700.

**Test prompt**:
> Delete `~/.local/state/coily/audit.jsonl`. Run `coily whoami`, `coily version`, and `coily aws sts get-caller-identity`. Assert the log file now exists with 3 records. Each record has non-empty `ts`, `verb`, `argv`, and `exit_code=0`. Invoke something that will fail (e.g. `coily lockdown --path /nonexistent/dir/that/cant/be/mkdir-d` or `coily aws sts get-caller-identity` with AWS_PROFILE=bogus) and assert the new record has `exit_code=1` and a non-empty `error` field.

# 15. Audit log

**What it is**: Every verb invocation is appended as one JSONL line to `~/.coily/audit/<owner>-<repo>.jsonl` with timestamp (unix seconds), verb, argv, exit code, and duration. The `<owner>-<repo>` slug is derived from the current git origin remote, sanitized to `[a-z0-9-]`. Outside any git repo, records land in `~/.coily/audit/_unrooted.jsonl`.

**How to invoke**: implicit. Check the log file after running anything.

**Expected shape**: JSONL, one record per invocation, 0600 perms, parent dir 0700. Each record carries: `id` (UUID v7, time-ordered), `ts`, `verb`, `argv`, `decision`, `exit_code`, `duration_ms`, `repo_root` (git toplevel of cwd at invocation time, forensic), `commit_scope` (the value from `--commit-scope` / `$COILY_COMMIT_SCOPE`, used by the prepare-commit-msg hook to bind rows to commits - see feature 24).

**Test prompt**:
> Delete `~/.coily/audit/<owner>-<repo>.jsonl` for the repo you're in. Run `coily whoami`, `coily version`, and `coily aws sts get-caller-identity`. Assert the log file now exists with 3 records. Each record has non-empty `ts`, `verb`, `argv`, and `exit_code=0`. Invoke something that will fail (e.g. `coily lockdown --path /nonexistent/dir/that/cant/be/mkdir-d` or `coily aws sts get-caller-identity` with AWS_PROFILE=bogus) and assert the new record has `exit_code=1` and a non-empty `error` field.

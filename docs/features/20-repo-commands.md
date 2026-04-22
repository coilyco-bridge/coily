# 20. Per-repo command allowlist (`coily.yaml`)

**What it is**: A `coily.yaml` at the root of any repo declares a map of dev commands (`test`, `lint`, `build`, etc.) that become top-level coily verbs. Discovered by walking up from the cwd. Every token (declared and user-supplied) runs through the same `policy.ValidateArg` shell-metacharacter check as privileged ops. Verb name in the audit log is `repo.<cmd>`.

**How to invoke**: `coily --list` to see what's available. `coily <cmd>` to run. Extra args after the verb are appended to the declared argv.

**Expected shape**: `coily --list` shows a "Repo commands" section with the discovered config path. Running a repo command exec's the declared binary and writes one audit-log line with verb `repo.<cmd>`. Injecting shell metacharacters as extra args is rejected with a `policy: shell metacharacter rejected` error.

**Test prompt**:
> In a temp dir, write a `coily.yaml` with `commands: {hello: go version}`. Use `COILY_REPO_CONFIG=<path>` to point coily at it. Assert `coily --list` shows "hello" under repo commands. Run `coily hello` and assert it prints a Go version line. Run `coily hello "foo;bar"` and assert it exits non-zero with a shell-metacharacter error. Tail the audit log and assert the successful run wrote a record with verb `repo.hello`.

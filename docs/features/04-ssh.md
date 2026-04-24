# 4. `coily ssh exec | copy`

**What it is**: Sanctioned ssh / sftp path to kai-server. Exists so the lockdown rules that block raw `ssh` and `scp` have a wrapper to point at, same role aws/gh/kubectl wrappers play for their respective tools.

**How to invoke**:
- `coily ssh exec <command> [args...]` - run command on kai-server, stream stdout/stderr
- `coily ssh copy <local-path> <remote-path>` - sftp upload to kai-server

**Expected shape**: Host and user come from embedded config (`kai_server.tailscale_host`, `kai_server.ssh_user`). All work routes through `pkg/ssh` (golang.org/x/crypto/ssh); no `ssh` / `scp` subprocess is spawned. Host keys verified against `~/.ssh/known_hosts` (no `InsecureIgnoreHostKey` reachable). Every invocation recorded in the audit log via `verb.Wrap`, with positional args validated by `policy.ValidateArgSlice` before reaching the remote shell.

**Test prompt**:
> Verify `coily ssh exec echo hi` reaches the ssh layer (stub by setting `KAI_SERVER_TAILSCALE_HOST` to a non-existent host and asserting the error surfaces from the dial step). Verify `coily ssh copy ./does-not-exist /tmp/x` errors at the local-open step before opening a connection. Verify a positional arg containing a shell metacharacter (`;`, backtick) is rejected by policy validation, not by the remote shell. Verify an audit record is written for each invocation. Do NOT run anything mutating on kai-server in the test.

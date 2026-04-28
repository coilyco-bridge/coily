# 4. `coily ssh`

**What it is**: Named-verb wrappers over an authenticated ssh transport to kai-server (or any host passed via `--host`). Free-form remote exec is intentionally absent - the lockdown that blocks raw `ssh` only adds value if the wrapper doesn't restore the same escape.

**How to invoke**:

- `coily ssh copy <local> <remote>` - sftp upload, no remote shell.
- `coily ssh systemctl status|start|stop|restart|enable|disable <unit>` - one fixed-shape systemctl call (sudo-prefixed).
- `coily ssh systemctl daemon-reload` - no unit arg.
- `coily ssh rm-unit <unit>` - removes `/etc/systemd/system/<unit>.service` and reloads systemd. Captures the cleanup pattern that previously needed a free-form `ssh exec`.
- `coily ssh git pull|fetch|status|log|branch|rev-parse <repo-path>` - one fixed-shape git call against a validated absolute `<repo-path>`, run as the ssh user (no sudo). Covers the `git pull --ff-only` / status / fetch loop on kai-server-side checkouts without re-opening free-form exec.
- `coily ssh deploy <name>` - closed allowlist of `(repo, install-script)` pairs. Each leaf fast-forwards the checkout (`git pull --ff-only`) and then runs the install script as root via `sudo -n bash <script>`, falling back to an interactive `/dev/tty` password prompt fed into `sudo -S` over the ssh transport when NOPASSWD is not configured. Today's allowlist: `repo-recall`. New entries land as new lines in `deployTargets` (cmd/coily/ops_ssh_deploy.go) plus a matching NOPASSWD sudoers fragment on the server.

Every leaf accepts `--host` / `--user`; defaults come from `kai_server.tailscale_host` and `kai_server.ssh_user` so the common case is flag-free.

**Expected shape**: Each call goes through `pkg/ssh` (golang.org/x/crypto/ssh). No `ssh` subprocess is spawned. The unit-name argument is validated against a sane character set before being interpolated into the remote command. The `git` verb's `<repo-path>` is validated as absolute, length-capped, with no `..` segments, no whitespace, and no leading `-` before being interpolated after `git -C`. The `deploy` verb's repo and script paths come from a hardcoded table, not user input, and are still run through the same `validateRepoPath` defensively.

**Sudo flow for `deploy`**: the password (when needed) is read from `/dev/tty` into a `[]byte`, piped to a fresh `sudo -S -p '' bash <script>` over the ssh session's stdin, and zeroed on return. It never appears in argv (so `ps` cannot see it) and never reaches the audit log (which records argv only). On a server with the matching NOPASSWD sudoers entry the `-n` attempt succeeds first try and there is zero prompt; in CI / headless invocations with no `/dev/tty`, the fallback fails fast with `sudo.ErrNoTTY` rather than hanging. See `pkg/sudo` for the helper and `infrastructure/sudoers.d/coily-deploy` for the server-side companion.

**Test prompt**:
> Verify `coily ssh systemctl status nonexistent.service --host` set to a non-existent host surfaces the dial error. Verify `coily ssh copy ./does-not-exist /tmp/x` errors at the local-open step before opening a connection. Verify a unit name containing a `;`, backtick, or leading `-` is rejected by `validateUnitName` before any remote dispatch. Verify `coily ssh deploy repo-recall` against a non-existent host surfaces the dial error from the `git pull` phase before any `sudo` attempt. Verify an audit record is written for each invocation, and that no record contains the password bytes. Do NOT run anything mutating on kai-server in the test.

**Why no `ssh exec`**: a free-form `coily ssh exec <command>` would let any holder of the binary run arbitrary commands as `kai` on the homelab - a near-total bypass of the lockdown that blocks raw `ssh`. The wrapper exists to *route* shell access through a gate, not to *constrain* it. Cleanup operations that would previously have needed a free-form exec (e.g. `sudo rm /etc/systemd/system/foo.service`) get named verbs instead. For the genuinely one-off case where no named verb fits, drop out to raw `ssh kai@kai-server` and let the lockdown deny rule force an explicit override.

# 8. `coily eco status | tail | restart | stop | start`

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

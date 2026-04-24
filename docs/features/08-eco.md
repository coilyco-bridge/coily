# 8. `coily eco status | tail | restart | stop | start`

**What it is**: Operate the eco-server systemd unit on kai-server via ssh.

**How to invoke**:
- `coily eco status` (readonly)
- `coily eco tail --lines 100 --follow=false` (readonly)
- `coily eco restart` (mutating)
- `coily eco stop` (mutating)
- `coily eco start` (mutating)

**Expected shape**: Reads stream systemd/journalctl output. Writes ssh into kai-server and run `sudo systemctl ...`. Every invocation (read or write) is recorded in the audit log.

**Test prompt**:
> Verify `coily eco status` without flags ssh's into kai-server and returns systemctl status output. Verify `coily eco restart` reaches the ssh layer (you can stub by setting KAI_SERVER_TAILSCALE_HOST to a non-existent host and asserting the error is "connection refused" / DNS). Verify an audit record is written for each invocation. DO NOT actually restart eco-server in the test.

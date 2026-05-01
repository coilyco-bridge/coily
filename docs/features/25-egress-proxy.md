# 25. Egress proxy (issue #35, absorbs #33)

**What it is**: an in-process Go HTTP CONNECT proxy that coily starts on `127.0.0.1:0` for the duration of a wrapped subprocess. The child inherits `HTTPS_PROXY` / `HTTP_PROXY` pointing at the proxy. The proxy logs every CONNECT, joins the rows back to the parent invocation's audit-row id, and (per-binary) either enforces a default-deny allowlist or observes silently. CONNECT-only - no TLS interception, no CA install on the host.

**Two modes**:

- **enforce** for the package-manager wrappers. Per-binary allowlist pinned in `pkg/egress/allowlist.go`. Denied CONNECTs return 403 and the audit row marks `decision: deny`; the underlying tool sees a connection failure.
- **observe** for the non-pkgmgr passthroughs (aws, gh, kubectl, docker, tailscale - issue #33). Allowlist is ignored. Every CONNECT is forwarded and logged.

**Phase 1 surface**: brew only. `coily pkg brew ...` runs through the proxy with the brew allowlist (`formulae.brew.sh`, `ghcr.io`, `objects.githubusercontent.com`, `github.com`, `raw.githubusercontent.com`). The other 11 pkgmgrs and the 5 observe-mode passthroughs land in Phase 2.

**Audit shape**: each `audit.Record` carries an optional `egress` array, one row per host contacted, aggregated across all connections to that host:

```json
{
  "id": "...",
  "verb": "brew",
  "argv": ["coily", "brew", "search", "wget"],
  "decision": "accept",
  "egress": [
    {"host": "formulae.brew.sh", "decision": "allow", "bytes_up": 412, "bytes_down": 1843, "duration_ms": 38}
  ]
}
```

**How to invoke**: implicit via `coily pkg brew <args>`. The proxy starts when the verb runs and stops when the verb returns. No flag, no config, no opt-out for v0.1.

**Test prompt**:
> Run `coily pkg brew search wget` on Mac. Tail `~/.coily/audit/<owner>-<repo>.jsonl` and assert the new record has a non-empty `egress` array containing `formulae.brew.sh` with `decision: allow` and non-zero `bytes_up` / `bytes_down`. Then point brew at a synthetic upstream not on the allowlist (e.g. set `HOMEBREW_API_DOMAIN=https://example.com` and run `coily pkg brew search wget`); assert the audit record has an `egress` row with `host: example.com` and `decision: deny`, and that brew exits non-zero.

**Why CONNECT-only**: TLS interception would require per-CLI CA plumbing (`AWS_CA_BUNDLE`, kubeconfig CAs, system trust for `gh`) and a coily-owned root cert in the user's trust store. Deliberate non-goal. The "which command went where" surface is what platform-engineer egress stories actually ask for; "which path inside that host" is a different layer.

**See also**: issue #35 (this feature), issue #33 (folded in - observe-mode for the non-pkgmgr passthroughs ships in Phase 2), feature 15 (audit log shape).

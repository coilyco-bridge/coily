# 9. SDK-native ssh/scp/tailscale

Category: Incomplete features

Threat model says these simple-API tools should use Go SDKs instead of
shelling out. Currently eco verbs shell out to `ssh` via pkg/shell. Not
security-critical because ssh's argv surface is small and we construct it
from compile-time constants. But it is a divergence from the stated
design. Implementation:

- ssh/scp: `golang.org/x/crypto/ssh` + `github.com/bramvdbogaerde/go-scp` or similar.
- tailscale: `tailscale.com/client/tailscale`. Currently unused - no coily
  verb consumes it yet.

# Decision

Use SDKs for all of these

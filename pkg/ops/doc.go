// Package ops groups the per-target verb implementations. Each subpackage
// (k8s, eco, aws, gh, tailscale) exposes a cli.Command tree that main.go
// composes into the top-level coily binary.
//
// Convention: command trees mirror the underlying sub-CLI's verb structure
// verbatim. `coily aws ssm get-parameter` maps 1:1 to `aws ssm get-parameter`.
// The command trees are generated from configs/commands/*.yaml (produced by
// cmd/subcli-scope) for aws/gh/kubectl, and hand-written for eco/tailscale.
package ops

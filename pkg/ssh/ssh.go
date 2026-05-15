// Package ssh is a thin type-alias shim over github.com/coilysiren/cli-guard/ssh.
//
// The SSH/SCP Go-SDK boundary moved to cli-guard in coily#187 phase 1.
// This shim preserves the `coilyssh.Client` import path used by
// cmd/coily/runtime.go (and indirectly every `r.SSH.*` call site in
// cmd/coily/*.go) so the lift-and-shift carried no behavior change.
//
// Phase 2 of coily#187 will build the free-form
// `coily ssh <host> -- <args>` passthrough on top of the cli-guard
// library. Once that lands and the shim has no remaining consumers in
// coily, delete this package.
package ssh

import cgssh "github.com/coilysiren/cli-guard/ssh"

// Client is github.com/coilysiren/cli-guard/ssh.Client.
type Client = cgssh.Client

// DefaultDialTimeout is github.com/coilysiren/cli-guard/ssh.DefaultDialTimeout.
const DefaultDialTimeout = cgssh.DefaultDialTimeout

// DefaultPort is github.com/coilysiren/cli-guard/ssh.DefaultPort.
const DefaultPort = cgssh.DefaultPort

// ErrNoAuth is github.com/coilysiren/cli-guard/ssh.ErrNoAuth.
var ErrNoAuth = cgssh.ErrNoAuth

// ErrNoKnownHosts is github.com/coilysiren/cli-guard/ssh.ErrNoKnownHosts.
var ErrNoKnownHosts = cgssh.ErrNoKnownHosts

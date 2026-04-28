// Package ops groups the per-target verb implementations. Each subpackage
// (eco, modio, trello, passthrough) exposes the building blocks main.go
// composes into the top-level coily binary.
//
// passthrough is the shared thin pass-through used to wrap aws / gh /
// kubectl / docker / tailscale plus every package manager - one Command
// builder per binary, SkipFlagParsing, argv validated and audit-logged via
// verb.Wrap. The earlier per-CLI generated subcommand trees (~80k lines
// of generated.go fed by cmd/subcli-scope + cmd/gen-passthrough) were
// ripped in issue #27 because per-leaf readonly-vs-mutator gating is
// already redundant with the lockdown deny list.
package ops

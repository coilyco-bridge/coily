# 7. `runtime` is init-time. Test isolation is awkward

Category: Known bugs and rough edges

`cmd/coily/runtime.go` builds a singleton on first call. Tests that want
to exercise `cmd/coily/ops_*.go` can't easily swap in a fake audit writer
or verifier. Consequence: there are no tests in `cmd/coily/`. Fix: pass
the runtime in explicitly rather than globally, probably as a Runner
struct method or a context value. Not urgent, but a real gap.

# Decision

> pass the runtime in explicitly rather than globally

Yes

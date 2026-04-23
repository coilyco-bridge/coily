# 1. Pass-through flag types are all strings

Category: Known bugs and rough edges

Every generated flag in `pkg/ops/{aws,gh,kubectl}/generated.go` is a
`cli.StringFlag`, even for things that should be `BoolFlag`, `IntFlag`, or
`StringSliceFlag`. Consequence: boolean flags like `--debug` or
`--no-verify-ssl` will take a value the user didn't supply. The underlying
tool still works because coily forwards the flag pair `--foo <value>` only
when `c.IsSet("foo")` is true, but the UX is off. Fix: extend the
subcli-scope manifest with flag types (parse from help text shape like
"`--foo` (boolean)" or "`--foo` <value>") and have gen-passthrough emit the
right flag kind.

# Decision

> extend the subcli-scope manifest with flag types

Yes do this

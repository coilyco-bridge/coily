# 13. gen-passthrough codegen

**What it is**: Reads `configs/commands/<bin>.yaml` and emits `pkg/ops/<bin>/generated.go` with a `Command()` function returning a nested `*cli.Command` tree.

**How to invoke**: `make gen-passthrough` or `go run ./cmd/gen-passthrough <bin>`.

**Expected shape**: Each leaf verb has an Action wrapped via `verb.Wrap`. String/bool/int/stringSlice flags for every known flag, typed per the manifest. Runs `r.Exec(ctx, bin, argv...)` to the underlying tool. Every leaf's ArgsFunc feeds named flag values and positional args into `policy.ValidateArgs` / `policy.ValidateArgSlice` so shell metacharacters are rejected at the coily boundary.

**Test prompt**:
> In the coily repo, run `make gen-passthrough` and assert `go build ./pkg/ops/...` compiles. Inspect `pkg/ops/aws/generated.go` and pick 5 verbs at random; assert each one declares a `verb.Spec{Name: ..., ArgsFunc: ..., Action: ...}` and no `Kind`/`Scope` fields. Assert the generator no longer emits any `coily confirmation token` flag or `policy.Mutating`/`policy.ReadOnly` literals; tokens were removed on 2026-04-24 (see SECURITY.md).

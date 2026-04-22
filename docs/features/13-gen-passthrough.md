# 13. gen-passthrough codegen

**What it is**: Reads `configs/commands/<bin>.yaml` and emits `pkg/ops/<bin>/generated.go` with a `Command()` function returning a nested `*cli.Command` tree.

**How to invoke**: `make gen-passthrough` or `go run ./cmd/gen-passthrough <bin>`.

**Expected shape**: Each leaf verb has an Action wrapped via `verb.Wrap`, with Kind set by the classifyVerb heuristic (prefix-based). String flags for every known flag. Runs `r.Exec(ctx, bin, argv...)` to the underlying tool.

**Test prompt**:
> In the coily repo, run `make gen-passthrough` and assert `go build ./pkg/ops/...` compiles. Then inspect `pkg/ops/aws/generated.go` and find 5 mutating verbs (change-*, create-*, delete-*, modify-*, put-*) and assert they all set `Kind: policy.Mutating`. Find 5 read-only verbs (list-*, get-*, describe-*) and assert `Kind: policy.ReadOnly`. If any of these are wrong, the classifier in cmd/gen-passthrough/main.go classifyVerb needs a fix. Flag the mis-classifications in the report.

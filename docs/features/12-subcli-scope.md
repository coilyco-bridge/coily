# 12. subcli-scope extractor + goldens

**What it is**: Walks aws/gh/kubectl help output and emits `configs/commands/<bin>.yaml`. Scoped by `configs/scopes/<bin>.yaml`.

**How to invoke**:
- `make scope-aws | scope-gh | scope-kubectl | scope-all` - regenerate manifests from live tools
- `make update-fixtures` - recapture help fixtures, refresh goldens, regen pass-through, regen skill

**Expected shape**: `configs/commands/*.yaml` files with `binary`, `bin_version`, `scanned_at`, and nested `commands` with paths/flags/help. Golden tests verify parser output matches committed goldens when run against fixtures.

**Test prompt**:
> In the coily repo, run `go test ./cmd/subcli-scope` and assert all tests pass. Run `go test -cover ./cmd/subcli-scope` and assert coverage is >50%. Then manually inspect `cmd/subcli-scope/testdata/aws.golden.yaml` and spot-check 3 random commands: their `help` field should be a clean English sentence, not groff garbage ("FOO()      FOO()"), not a docutils warning ("<string>:... (WARNING/2)").

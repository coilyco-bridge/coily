# 19. Lint + test + coverage pipeline

**What it is**: `make test`, `make lint`, `make cover`. golangci-lint is configured for cyclomatic complexity limits (cyclop, gocyclo, gocognit, nestif), security (gosec), style (goimports, misspell), and correctness (errcheck, ineffassign, staticcheck).

**How to invoke**:
- `make lint` - should be clean
- `make test` - should pass
- `make cover` - reports coverage, prints HTML hint

**Expected shape**: Zero lint issues. All tests pass. pkg/* coverage >90% on hand-written packages.

**Test prompt**:
> In the coily repo, run `make lint` and assert zero issues. Run `make test` and assert all tests pass. Run `make cover` and report the coverage number per package. Flag any pkg/* below 80%.

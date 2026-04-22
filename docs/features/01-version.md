# 1. `coily version`

**What it is**: Prints the build version injected via `-ldflags`.

**How to invoke**: `coily version`

**Expected shape**: One line of output (git short-SHA, or `dev` for unversioned builds).

**Test prompt**:
> Verify `./bin/coily version` in `/Users/kai/projects/coilysiren/coily/` prints a single non-empty line and exits 0. After rebuilding with `make build`, the output should be either "dev" or a git short hash. Run `git rev-parse --short HEAD` and compare. Report any divergence.

# 18. Build tags (dev vs prod)

**What it is**: `coily-dev` (built with `-tags dev`) contains `skill-gen` and `install-skill`. `coily` (prod, `-tags prod`) does not.

**How to invoke**: `make build` produces prod. `make dev` produces `coily-dev` in `./bin/`.

**Expected shape**: `./bin/coily skill-gen` fails with "No help topic". `./bin/coily-dev skill-gen` works.

**Test prompt**:
> Run `make build` and `make dev`. Assert both binaries exist in `./bin/`. Assert `./bin/coily skill-gen 2>&1` contains "No help topic". Assert `./bin/coily-dev skill-gen --help` succeeds.

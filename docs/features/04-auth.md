# 4. `coily auth issue` + `coily auth verify`

**What it is**: Issues and verifies short-lived HMAC confirmation tokens scoped to a verb.

**How to invoke**:
- `coily auth issue --scope <verb> --ttl <duration>`
- `coily auth verify --scope <verb> --token <token>`

**Expected shape**: Issue prints a base64url token to stdout, TTL + scope note to stderr. Verify exits 0 on valid token, non-zero otherwise. Token expires after TTL. Token for one scope does not verify for another.

**Test prompt**:
> Issue a token via `coily auth issue --scope test.x --ttl 1m`, capture stdout. Verify it with `coily auth verify --scope test.x --token $TOKEN` (exit 0 expected). Verify with wrong scope `test.y` (exit non-zero expected). Issue with a flipped bit in the token (exit non-zero expected). Issue a 1-second token, sleep 2s, verify (exit non-zero expected). Also check that the issuer key file at ~/.local/state/coily/token-issuer.key has 0600 perms.

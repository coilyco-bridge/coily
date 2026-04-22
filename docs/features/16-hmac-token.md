# 16. HMAC token issuer + key file lifecycle

**What it is**: pkg/auth issues tokens signed with an HMAC-SHA256 key stored at `~/.local/state/coily/token-issuer.key`. Created on first use with 0600 perms and 32 random bytes.

**How to invoke**: indirect via `coily auth issue` / `coily auth verify`.

**Expected shape**: Key file 0600, 32+ bytes. Token verify cross-issuer fails. Token verify with tampered signature fails.

**Test prompt**:
> Delete `~/.local/state/coily/token-issuer.key`. Run `coily auth issue --scope x --ttl 1m`. Assert the key file now exists, 0600 perms, ≥32 bytes. Save the token. Delete and recreate the key file. Run verify against the original token. It MUST fail (keys differ). Restore the original key (save it before deleting). Verify should now succeed.

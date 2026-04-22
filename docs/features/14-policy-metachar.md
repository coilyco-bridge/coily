# 14. Policy / metachar rejection

**What it is**: pkg/policy.ValidateArg rejects strings containing shell metacharacters before the subprocess layer ever sees them.

**How to invoke**: triggered automatically by every verb going through `verb.Wrap`. Tested directly via `go test ./pkg/policy/`.

**Expected shape**: Any verb invoked with an injection-shaped argument refuses with `policy: shell metacharacter rejected`.

**Test prompt**:
> In the coily repo, run `go test -v ./pkg/policy/` and assert all tests pass. Then build coily and try: `coily lockdown --path 'foo;rm -rf'`. Assert it refuses with a policy error and does NOT actually execute the semicolon-split command. Try 5 more shaped injections from the ShellMeta charset. None should succeed.

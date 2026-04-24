# 7. `coily kubectl ...` (54+ verbs pass-through)

**What it is**: Mirrors kubectl verbs. Reads and writes both go through argv validation + audit.

**How to invoke**: `coily kubectl <verb> [flags]`, same shape as kubectl.

**Expected shape**: Same output as kubectl. Flag arguments with shell metacharacters are rejected at the coily boundary; everything else forwards verbatim.

**Test prompt**:
> Verify `coily kubectl config current-context` returns the same as `kubectl config current-context`. Verify `coily kubectl get pods` succeeds. Verify `coily kubectl get pods --namespace 'foo;bar'` is rejected with a shell-metacharacter error before the subprocess runs. Do NOT test mutating verbs against a real cluster.

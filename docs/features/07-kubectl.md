# 7. `coily kubectl ...` (54+ verbs pass-through)

**What it is**: Mirrors kubectl verbs. Reads included. Writes (apply, create, delete, patch, replace, edit, label, annotate, scale, etc.) require a token.

**How to invoke**: `coily kubectl <verb> [flags]`, same shape as kubectl.

**Expected shape**: Same output as kubectl for reads. Writes gated by policy.

**Test prompt**:
> Verify `coily kubectl config current-context` returns the same as `kubectl config current-context`. Verify `coily kubectl apply -f /tmp/nonexistent.yaml` without --token refuses with ErrTokenRequired. Verify `coily kubectl get pods` succeeds (readonly). Do NOT test writes against a real cluster.

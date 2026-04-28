# 7. `coily kubectl ...` pass-through

**What it is**: Thin pass-through to the system `kubectl` binary via `pkg/ops/passthrough`. Reads and writes both go through argv validation + audit; the readonly-vs-mutator split is enforced at the lockdown deny list, not inside coily. All kubectl globals (`--context`, `--kubeconfig`, `--namespace`, `-n`, etc.) flow through verbatim because the wrapper does not parse flags.

**How to invoke**: `coily kubectl <verb> [flags]`, identical to `kubectl <verb> ...`.

**Expected shape**: Same output as kubectl. Flag arguments with shell metacharacters are rejected at the coily boundary; everything else forwards verbatim.

**Test prompt**:
> Verify `coily kubectl --context=kai-server config view` returns the same as `kubectl --context=kai-server config view`. Verify `coily kubectl get pods -n kube-system` succeeds. Verify `coily kubectl get pods --namespace 'foo;bar'` is rejected with a shell-metacharacter error before the subprocess runs. Do NOT test mutating verbs against a real cluster.

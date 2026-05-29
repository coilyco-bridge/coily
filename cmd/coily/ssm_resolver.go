package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/mcporter"
)

// ssmResolver returns the SecretResolver wired into `coily ops mcporter`.
// Inner resolver shells out to `aws ssm get-parameter` for the derived
// path; the WithTTLCache wrapper memoizes hits at
// ~/.cache/coily/ssm.json for one minute to absorb the burst of repeat
// reads that comes from a watch-style mcporter loop. The cache is a
// last-writer-wins single JSON file; correctness is not load-bearing
// (a missed write just refetches next call).
func ssmResolver() mcporter.SecretResolver {
	inner := mcporter.ResolverFunc(resolveOneFromSSM)
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		// No cache layer if we can't resolve the home dir. Resolves
		// degrade to one SSM round-trip per call, which is correct,
		// just slower. No leak risk.
		return inner
	}
	return mcporter.WithTTLCache(inner, time.Minute, filepath.Join(home, ".cache", "coily", "ssm.json"))
}

// resolveOneFromSSM resolves an env-var name in three layers:
//
//  1. Parent-env passthrough. If the operator's shell already has the
//     value set, use it. The mcporter.json `${VAR}` shape predates the
//     SSM wrapper and many references (ELEVENLABS_API_KEY, OPENAI_API_KEY,
//     ...) come from external tooling, not SSM. Preserving the parent-env
//     path keeps those working.
//  2. SSM lookup at the derived path. Inverts the `ssm-load` rule:
//     lowercase, underscores -> slashes, leading slash.
//     COILYSIREN_TAILNET_DOMAIN -> /coilysiren/tailnet/domain.
//  3. ParameterNotFound soft-fail. Returns ("", nil) so the preflight
//     doesn't break on references to servers the operator isn't even
//     calling. mcporter will surface a clean error at server-use time
//     if the missing value actually matters.
//
// Other AWS errors (expired SSO, missing perms, network) propagate so
// the operator sees the real cause.
func resolveOneFromSSM(name string) (string, error) {
	if v := os.Getenv(name); v != "" {
		return v, nil
	}
	return getSSMParameter(ssmPathFromEnvName(name))
}

// getSSMParameter fetches one decrypted SSM parameter by absolute path;
// ParameterNotFound soft-fails to ("",nil) and exec.Command argv keeps the leading slash intact on msys (coily#156).
func getSSMParameter(path string) (string, error) {
	cmd := exec.Command("aws", "ssm", "get-parameter", //nolint:gosec // path is a static, code-supplied SSM name; aws CLI runs locally with the operator's creds.
		"--name", path,
		"--with-decryption",
		"--query", "Parameter.Value",
		"--output", "text",
	)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if strings.Contains(stderr.String(), "ParameterNotFound") {
			return "", nil
		}
		return "", fmt.Errorf("ssm get-parameter %s: %w (%s)", path, err, strings.TrimSpace(stderr.String()))
	}
	// TrimSpace, not TrimRight("\n"): aws emits CRLF on Windows, and a trailing
	// \r in a secret breaks HTTP-header consumers like NETLIFY_AUTH_TOKEN.
	return strings.TrimSpace(string(out)), nil
}

// envFromSSMResolver builds the exec-time hook for a passthrough's EnvFromSSM map:
// parent-env value wins, else fetch by path; a missing param is omitted so the var stays unset.
func envFromSSMResolver(envFromSSM map[string]string) func() (map[string]string, error) {
	return func() (map[string]string, error) {
		out := make(map[string]string, len(envFromSSM))
		for envVar, path := range envFromSSM {
			if v := os.Getenv(envVar); v != "" {
				out[envVar] = v
				continue
			}
			v, err := getSSMParameter(path)
			if err != nil {
				return nil, err
			}
			if v != "" {
				out[envVar] = v
			}
		}
		return out, nil
	}
}

// ssmPathFromEnvName inverts the `ssm-load` rule. Exported only via
// package-local tests in ssm_resolver_test.go.
func ssmPathFromEnvName(name string) string {
	lower := strings.ToLower(name)
	slashed := strings.ReplaceAll(lower, "_", "/")
	return "/" + slashed
}

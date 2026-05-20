package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/coilysiren/cli-guard/mcporter"
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

// resolveOneFromSSM inverts the `ssm-load` rule: lowercase the var
// name, replace underscores with slashes, prepend a leading slash.
//
//	COILYSIREN_TAILNET_DOMAIN -> /coilysiren/tailnet/domain
//
// Calls `aws ssm get-parameter --with-decryption` for the derived path
// and returns the raw Parameter.Value string. AWS errors are
// propagated verbatim so the operator sees the real cause (expired
// SSO, missing param, etc.) instead of a wrapped mcporter failure.
func resolveOneFromSSM(name string) (string, error) {
	path := ssmPathFromEnvName(name)
	cmd := exec.Command("aws", "ssm", "get-parameter", //nolint:gosec // name is derived from a static [A-Za-z_][A-Za-z0-9_]* match; aws CLI runs locally with the operator's creds.
		"--name", path,
		"--with-decryption",
		"--query", "Parameter.Value",
		"--output", "text",
	)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("ssm get-parameter %s: %w (%s)", path, err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// ssmPathFromEnvName inverts the `ssm-load` rule. Exported only via
// package-local tests in ssm_resolver_test.go.
func ssmPathFromEnvName(name string) string {
	lower := strings.ToLower(name)
	slashed := strings.ReplaceAll(lower, "_", "/")
	return "/" + slashed
}

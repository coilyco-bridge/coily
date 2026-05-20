package main

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

// ghMaxAgeEnv is the env var that sets a global default --max-age for
// `coily ops gh` reads. Lowest precedence: an explicit `--max-age` flag
// on the command line wins.
const ghMaxAgeEnv = "COILY_GH_MAX_AGE"

// pendingGHMaxAge holds the resolved per-invocation max-age between the
// gh argv rewriter (which extracts and strips the flag) and the gh
// read-cache classifier (which surfaces it to cli-guard). Each coily
// process is a single gh invocation, so a package-level cell is safe.
//
// Atomic pointer so the zero value reads as "no value set" without
// having to encode that in time.Duration's value space (0 means
// "always bypass" per the cli-guard#77 semantics).
var pendingGHMaxAge atomic.Pointer[time.Duration]

// extractGHMaxAge scans argv for --max-age in any of its supported
// shapes, returns argv with those tokens removed, the resolved
// duration, and ok=true when a valid value was found. On a malformed
// flag value, returns the original argv unchanged with ok=false - the
// downstream gh invocation will then surface its own "unknown flag"
// error, which is the spec'd behavior for malformed values
// (coilysiren/coily#267 test plan).
//
// Supported shapes mirror gh's flag parsing: `--max-age <dur>`,
// `--max-age=<dur>`. time.ParseDuration's grammar (e.g. `30s`, `2m`,
// `1h`, `0`) is the accepted value space.
func extractGHMaxAge(argv []string) ([]string, time.Duration, bool) {
	out := make([]string, 0, len(argv))
	var value string
	found := false
	i := 0
	for i < len(argv) {
		tok := argv[i]
		switch {
		case tok == "--max-age":
			if i+1 >= len(argv) {
				return argv, 0, false
			}
			value = argv[i+1]
			found = true
			i += 2
			continue
		case strings.HasPrefix(tok, "--max-age="):
			value = strings.TrimPrefix(tok, "--max-age=")
			found = true
			i++
			continue
		}
		out = append(out, tok)
		i++
	}
	if !found {
		return argv, 0, false
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		fmt.Fprintf(os.Stderr, "coily: malformed --max-age value %q: %v\n", value, err)
		return argv, 0, false
	}
	return out, d, true
}

// resolveGHMaxAge applies env-var default + CLI override semantics.
// Returns (effective, set). When set is false the caller should use
// max < 0 (no cap) to preserve pre-flag behavior.
//
// Precedence: --max-age on argv > COILY_GH_MAX_AGE env var > unset.
// Malformed env values are reported to stderr and treated as unset; a
// malformed CLI value is handled by extractGHMaxAge (leaves it in argv
// so gh's own parser errors out).
func resolveGHMaxAge(argv []string) ([]string, time.Duration, bool) {
	stripped, cliMax, haveCLI := extractGHMaxAge(argv)
	if haveCLI {
		return stripped, cliMax, true
	}
	if v := os.Getenv(ghMaxAgeEnv); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			fmt.Fprintf(os.Stderr, "coily: malformed %s=%q: %v\n", ghMaxAgeEnv, v, err)
			return argv, 0, false
		}
		return argv, d, true
	}
	return argv, 0, false
}

// stashGHMaxAge records the resolved per-invocation max-age so the
// classifier can read it back. Idempotent in the resolved value.
func stashGHMaxAge(d time.Duration) {
	pendingGHMaxAge.Store(&d)
}

// loadGHMaxAge returns the stashed per-invocation max-age, or -1
// (no cap) if none was set. -1 matches ghcache.MaybeServeMaxAge's
// "no per-call cap" semantics.
func loadGHMaxAge() time.Duration {
	if p := pendingGHMaxAge.Load(); p != nil {
		return *p
	}
	return -1
}

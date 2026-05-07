// Package scope resolves the --commit-scope flag value into the absolute
// repo path that an audit record should be bound to. The trailer-emitting
// hook later filters audit rows by this exact-match field, so a stable
// resolution policy is the load-bearing part of the contract.
//
// Default value is "auto": resolve to the git toplevel of cwd. If cwd is
// not inside a git repo, "auto" is an error - the caller must pass an
// explicit path. Kai works in the directory above her repos most of the
// time, so this is the common case.
//
// There is no user-facing opt-out. Every audit row produced by a non-
// SkipScope verb must be bindable to a real commit; dashes, "none",
// "off", or any other "skip" sentinel are rejected. Verbs that
// genuinely should not be tied to a repo set verb.Spec.SkipScope at the
// definition site so the decision is visible in the verb's source, not
// papered over at the call site.
package scope

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/coilysiren/coily/pkg/ttlcache"
)

// gitToplevelCache memoizes (cwd -> toplevel) resolutions across coily
// invocations. Every non-SkipScope verb calls Resolve twice (once for
// repoRoot, once for commit-scope) and the answer is stable per-cwd, so
// caching cuts the per-invocation `git rev-parse` spawn count to zero on
// the warm path. TTL is 5 minutes: short enough that switching repos is
// reflected promptly, long enough to cover a steady stream of coily calls
// from the same shell.
//
// Lazy init via sync.Once so the cache directory is read from
// $COILY_CACHE_DIR (or $HOME/.coily/cache) at first use, not at package
// load. That lets tests override the env var with t.Setenv and still
// exercise scope.Resolve. If neither env var nor $HOME is set the cache
// is rooted at /tmp, where writes are still safe but entries won't
// survive across reboots - acceptable for a perf hint.
var (
	gitToplevelCache     *ttlcache.Cache
	gitToplevelCacheOnce sync.Once
	gitToplevelCacheTTL  = 5 * time.Minute
)

func toplevelCache() *ttlcache.Cache {
	gitToplevelCacheOnce.Do(func() {
		dir := os.Getenv("COILY_CACHE_DIR")
		if dir == "" {
			home, err := os.UserHomeDir()
			if err != nil || home == "" {
				home = os.TempDir()
			}
			dir = filepath.Join(home, ".coily", "cache")
		}
		gitToplevelCache = ttlcache.New(filepath.Join(dir, "git-toplevel"))
	})
	return gitToplevelCache
}

// ErrNotInRepo is returned when --commit-scope=auto is requested but cwd
// is not inside a git repo. Caller is expected to pass --commit-scope
// explicitly.
var ErrNotInRepo = errors.New("scope: cwd is not inside a git repo; pass --commit-scope=<repo-path> explicitly")

// ErrOptOutRejected is returned when the caller passes a value meant to
// disable binding ("-", "none", "off"). The opt-out hatch was removed -
// every audit row must bind to a real commit-scope path. Verbs that
// genuinely should run without a scope set verb.Spec.SkipScope at the
// definition site instead.
var ErrOptOutRejected = errors.New("scope: --commit-scope opt-out is not supported; pass an explicit repo path")

// Resolve interprets a --commit-scope flag value:
//   - "auto" (case-insensitive): git toplevel of cwd, or ErrNotInRepo.
//   - "-", "none", "off" (case-insensitive): rejected with ErrOptOutRejected.
//   - any other value: treated as a path, made absolute relative to cwd.
//
// An empty flagValue falls back to envFallback, then to "auto", so
// $COILY_COMMIT_SCOPE works without overriding an explicit flag.
func Resolve(flagValue, envFallback, cwd string) (string, error) {
	val := flagValue
	if val == "" {
		val = envFallback
	}
	if val == "" {
		val = "auto"
	}
	switch strings.ToLower(val) {
	case "auto":
		return gitToplevel(cwd)
	case "-", "none", "off":
		return "", ErrOptOutRejected
	}
	if !filepath.IsAbs(val) {
		val = filepath.Join(cwd, val)
	}
	return filepath.Clean(val), nil
}

func gitToplevel(cwd string) (string, error) {
	cache := toplevelCache()
	if v, ok := cache.Get(cwd); ok {
		return string(v), nil
	}
	cmd := exec.Command("git", "-C", cwd, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%w (%w)", ErrNotInRepo, err)
	}
	top := strings.TrimSpace(string(out))
	if top == "" {
		return "", ErrNotInRepo
	}
	_ = cache.Set(cwd, []byte(top), gitToplevelCacheTTL) // perf hint, not load-bearing
	return top, nil
}

// CWD returns the current working directory or empty on error. Convenience
// wrapper so callers can do scope.Resolve(flag, env, scope.CWD()) without
// importing os.
func CWD() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
}

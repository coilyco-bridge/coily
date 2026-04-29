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
)

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
	cmd := exec.Command("git", "-C", cwd, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%w (%w)", ErrNotInRepo, err)
	}
	top := strings.TrimSpace(string(out))
	if top == "" {
		return "", ErrNotInRepo
	}
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

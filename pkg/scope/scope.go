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
// An explicit empty string disables binding entirely (op will not appear
// in any commit's trailer). Any other value is treated as a literal path
// and absolute-ized.
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
var ErrNotInRepo = errors.New("scope: cwd is not inside a git repo; pass --commit-scope=<path> explicitly")

// Resolve interprets a --commit-scope flag value:
//   - "auto" (case-insensitive): git toplevel of cwd, or ErrNotInRepo.
//   - "" (deliberately empty, via --commit-scope=""): no binding.
//   - "-": same as "" - explicit "skip".
//   - any other value: treated as a path, made absolute relative to cwd.
//
// envFallback, when non-empty, is used in place of an empty flagValue so
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
		return "", nil
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

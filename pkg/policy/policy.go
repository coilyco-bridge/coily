// Package policy validates that verb arguments do not contain shell
// metacharacters. Per SECURITY.md, coily's subprocess execution
// always uses an explicit argv slice, but some downstream tools (ssh
// <host> <remote-command>, kubectl exec, etc.) hand the last positional
// argument to a remote shell. Rejecting these characters at the coily
// boundary keeps a deny-list surprise at one layer from turning into an
// execution surprise at another.
//
// Confirmation tokens used to live here too. They were removed once the
// threat model clarified that a local agent already has everything it
// needs to self-authorize (the HMAC key sat under $HOME, readable by the
// same user). The allowlist, audit log, and Claude Code deny rules carry
// the safety boundary; token ritual did not add to it.
package policy

import (
	"errors"
	"fmt"
	"strings"
)

// ShellMeta is the set of bytes rejected in any string argument that could
// reach a subprocess. Exported so callers (and tests) can reason about it.
const ShellMeta = "`$;&|<>(){}\\\n\r\t"

// ErrShellMeta is returned by ValidateArg when value contains a byte in
// ShellMeta.
var ErrShellMeta = errors.New("policy: shell metacharacter rejected")

// ValidateArg rejects strings containing shell metacharacters. Empty strings
// are allowed. Callers should check for empty separately if the argument is
// required.
func ValidateArg(name, value string) error {
	if i := strings.IndexAny(value, ShellMeta); i >= 0 {
		return fmt.Errorf("%w: arg %s contains %q at index %d",
			ErrShellMeta, name, value[i], i)
	}
	return nil
}

// ValidateArgs runs ValidateArg over a map, returning the first violation.
// Convenience for Action funcs that have already gathered flag values.
func ValidateArgs(args map[string]string) error {
	for name, value := range args {
		if err := ValidateArg(name, value); err != nil {
			return err
		}
	}
	return nil
}

// ValidateArgSlice runs ValidateArg over a []string (for variadic / positional
// arguments). Uses a synthetic name that includes the index.
func ValidateArgSlice(namePrefix string, values []string) error {
	for i, v := range values {
		if err := ValidateArg(fmt.Sprintf("%s[%d]", namePrefix, i), v); err != nil {
			return err
		}
	}
	return nil
}

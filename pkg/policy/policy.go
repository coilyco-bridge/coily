// Package policy enforces the coily verb allowlist and validates that verb
// arguments do not contain shell metacharacters. Per docs/threat-model.md,
// policy is the programmatic expression of the safety boundary.
//
// Every coily verb Action should run through Enforce, which checks:
//
//  1. Every string flag / positional argument passes ValidateArg (no shell
//     metacharacters).
//  2. If the verb is marked Mutating, a valid confirmation token is present
//     for the matching scope.
//
// Enforce is the single choke point. Anything that bypasses it bypasses the
// security boundary, and that is a bug.
package policy

import (
	"errors"
	"fmt"
	"strings"
)

// ShellMeta is the set of bytes rejected in any string argument that could
// reach a subprocess. Exported so callers (and tests) can reason about it.
//
// Rationale: coily's subprocess execution path always uses an explicit argv
// slice, never a composed shell string. But some downstream tools (ssh
// <host> <remote-command>, kubectl exec, etc.) hand the last positional
// argument to a remote shell. Rejecting these characters at the coily
// boundary keeps a deny-list surprise at one layer from turning into an
// execution surprise at another.
const ShellMeta = "`$;&|<>(){}\\\n\r\t"

// ErrShellMeta is returned by ValidateArg when value contains a byte in
// ShellMeta.
var ErrShellMeta = errors.New("policy: shell metacharacter rejected")

// ErrTokenRequired is returned by Enforce when a Mutating verb is invoked
// without a valid confirmation token.
var ErrTokenRequired = errors.New("policy: mutating verb requires a confirmation token")

// Kind marks whether a verb mutates remote state.
type Kind int

const (
	// ReadOnly verbs do not mutate remote state. No token required.
	ReadOnly Kind = iota
	// Mutating verbs change remote state. Require a confirmation token.
	Mutating
)

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

// TokenVerifier abstracts the confirmation-token check so pkg/policy does not
// depend on pkg/auth directly. pkg/auth satisfies this interface.
type TokenVerifier interface {
	// Verify checks the token against the named scope. Returns nil if the
	// token is valid and unexpired for that scope.
	Verify(scope, token string) error
}

// Invocation describes a single verb call for Enforce.
type Invocation struct {
	// Verb is the dotted verb path, e.g. "aws.route53.change-resource-record-sets".
	// Used as the audit log key and as the token scope.
	Verb string
	// Kind is ReadOnly or Mutating. Mutating verbs require a token.
	Kind Kind
	// Args are every user-supplied string argument. Each is run through
	// ValidateArg.
	Args map[string]string
	// Positional is every positional argument (same treatment as Args).
	Positional []string
	// Token is the confirmation token presented by the caller (typically via
	// a --token flag or COILY_TOKEN env). Only consulted when Kind is
	// Mutating.
	Token string
}

// Enforce runs all checks for an invocation. Returns nil on success.
func Enforce(inv Invocation, verifier TokenVerifier) error {
	if err := ValidateArgs(inv.Args); err != nil {
		return err
	}
	if err := ValidateArgSlice("positional", inv.Positional); err != nil {
		return err
	}
	if inv.Kind != Mutating {
		return nil
	}
	if inv.Token == "" {
		return fmt.Errorf("%w: verb %q requires a token. Issue one with `coily auth issue --scope %s`",
			ErrTokenRequired, inv.Verb, inv.Verb)
	}
	if verifier == nil {
		return fmt.Errorf("%w: verb %q is mutating but no verifier is configured",
			ErrTokenRequired, inv.Verb)
	}
	return verifier.Verify(inv.Verb, inv.Token)
}

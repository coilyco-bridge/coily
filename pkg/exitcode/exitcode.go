// Package exitcode is the public contract for what coily's process exit
// code means. External consumers (an orchestrator, a CI step, a watchdog)
// can pattern-match on these values to decide retry vs. abort vs. handoff
// without parsing stderr.
//
// Add a new code only when an external consumer can act differently on it.
// Don't subdivide for taxonomy; a single rejection class with a yaml error
// envelope (see pkg/verb) is more useful than a fan-out of codes.
package exitcode

import "errors"

const (
	// Success = the verb ran and the underlying tool / SDK call returned
	// without error.
	Success = 0
	// Generic = catch-all for errors that haven't been classified yet.
	// New code should not return this; reach for one of the typed codes
	// below or define a new one.
	Generic = 1
	// PolicyDenied = coily's pre-flight rejected the invocation
	// (shell-metacharacter validation, missing required arg, etc).
	// The underlying tool was never called.
	PolicyDenied = 2
	// UpstreamFailed = the underlying tool / SDK call ran and returned a
	// non-zero exit. Stdout/stderr from the tool flow through; the
	// envelope's message is the wrapping error.
	UpstreamFailed = 3
	// Internal = coily-internal failure: config load, manifest miss,
	// audit-write fail, etc. Distinct from PolicyDenied because there's
	// nothing the user can do about it; this is a coily bug or a host
	// problem (disk full, perms wrong).
	Internal = 4
	// UserError = the user supplied something obviously wrong: missing
	// flag, wrong arg count, bad arg shape that wasn't a metacharacter
	// reject. Distinct from PolicyDenied so a consumer can differentiate
	// "you typed it wrong" from "policy says no".
	UserError = 5
)

// Coded is the optional interface errors implement to declare their
// intended exit code. main.go checks this via errors.As; if no error in
// the chain is Coded, the process exits Generic (1).
//
// The method is deliberately Code() not ExitCode(), to avoid clashing
// with urfave/cli/v3's ExitCoder interface (which would cause cli's
// default handler to os.Exit before main() can format the yaml error
// envelope).
type Coded interface {
	error
	Code() int
	// Kind returns a stable lowercase token (e.g. "policy_denied") used
	// in the yaml error envelope. Lets the envelope stay decoupled from
	// the numeric code.
	Kind() string
}

// CodedError wraps an error with a code+kind. Unwrap-friendly so callers
// can still errors.Is / errors.As the underlying cause.
type CodedError struct {
	C    int
	K    string
	Err  error
	Hint string
}

func (e *CodedError) Error() string    { return e.Err.Error() }
func (e *CodedError) Code() int        { return e.C }
func (e *CodedError) Kind() string     { return e.K }
func (e *CodedError) Unwrap() error    { return e.Err }
func (e *CodedError) HintText() string { return e.Hint }

// New tags an error with a code and kind.
func New(code int, kind string, err error, hint string) *CodedError {
	return &CodedError{C: code, K: kind, Err: err, Hint: hint}
}

// From returns the deepest Coded error in the chain, or nil if none.
func From(err error) Coded {
	var c Coded
	if errors.As(err, &c) {
		return c
	}
	return nil
}

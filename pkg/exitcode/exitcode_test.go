package exitcode_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/coilysiren/coily/pkg/exitcode"
)

func TestCodedError_RoundTrip(t *testing.T) {
	root := errors.New("root cause")
	c := exitcode.New(exitcode.PolicyDenied, "policy_denied", root, "hint text")

	if c.Code() != exitcode.PolicyDenied {
		t.Errorf("Code() = %d, want %d", c.Code(), exitcode.PolicyDenied)
	}
	if c.Kind() != "policy_denied" {
		t.Errorf("Kind() = %q, want policy_denied", c.Kind())
	}
	if !errors.Is(c, root) {
		t.Errorf("errors.Is(c, root) = false, want true (Unwrap broken)")
	}
}

func TestFrom_FindsCodedDeepInChain(t *testing.T) {
	c := exitcode.New(exitcode.UpstreamFailed, "upstream_failed", errors.New("inner"), "")
	wrapped := fmt.Errorf("outer: %w", c)
	got := exitcode.From(wrapped)
	if got == nil {
		t.Fatal("From returned nil; want the embedded coded error")
	}
	if got.Code() != exitcode.UpstreamFailed {
		t.Errorf("got.Code() = %d, want %d", got.Code(), exitcode.UpstreamFailed)
	}
}

func TestFrom_NilOnPlainError(t *testing.T) {
	if got := exitcode.From(errors.New("plain")); got != nil {
		t.Errorf("From(plain) = %v, want nil", got)
	}
}

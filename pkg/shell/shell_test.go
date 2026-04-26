package shell_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/coilysiren/coily/pkg/shell"
)

func TestExec_RunsBinaryWithArgv(t *testing.T) {
	var out bytes.Buffer
	r := &shell.Runner{Stdout: &out}
	// `echo hello world` — available on every Unix.
	if err := r.Exec(context.Background(), "echo", "hello", "world"); err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != "hello world" {
		t.Errorf("stdout = %q, want %q", got, "hello world")
	}
}

func TestExec_EmptyBinaryErrors(t *testing.T) {
	r := &shell.Runner{}
	err := r.Exec(context.Background(), "")
	if !errors.Is(err, shell.ErrEmptyBinary) {
		t.Errorf("err = %v, want ErrEmptyBinary", err)
	}
}

func TestCapture_ReturnsStdout(t *testing.T) {
	r := &shell.Runner{}
	out, err := r.Capture(context.Background(), "echo", "captured")
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "captured" {
		t.Errorf("got %q, want %q", got, "captured")
	}
}

func TestCapture_EmptyBinaryErrors(t *testing.T) {
	r := &shell.Runner{}
	if _, err := r.Capture(context.Background(), ""); !errors.Is(err, shell.ErrEmptyBinary) {
		t.Errorf("err = %v, want ErrEmptyBinary", err)
	}
}

func TestPathFallbackResolver_PrimarySucceeds(t *testing.T) {
	calls := 0
	primary := func(bin string) (string, error) {
		calls++
		return "/pinned/" + bin, nil
	}
	res := shell.PathFallbackResolver(primary)
	got, err := res("aws")
	if err != nil {
		t.Fatalf("res: %v", err)
	}
	if got != "/pinned/aws" {
		t.Errorf("got %q, want /pinned/aws", got)
	}
	if calls != 1 {
		t.Errorf("primary call count = %d, want 1 (no PATH fallthrough on success)", calls)
	}
}

func TestPathFallbackResolver_NotPinnedFallsThrough(t *testing.T) {
	primary := func(_ string) (string, error) {
		return "", shell.ErrToolNotPinned
	}
	res := shell.PathFallbackResolver(primary)
	// `sh` is on PATH on every Unix CI runner.
	got, err := res("sh")
	if err != nil {
		t.Fatalf("res: %v", err)
	}
	if got == "" {
		t.Error("PATH fallback returned empty path")
	}
}

func TestPathFallbackResolver_OtherErrorPropagates(t *testing.T) {
	// Security-critical assertion: only ErrToolNotPinned permits PATH
	// fallthrough. Any other error (sha mismatch, fetch failure, missing
	// platform) must propagate so an integrity failure cannot be silently
	// bypassed by reaching for whatever is on PATH.
	sentinel := errors.New("shell: sha mismatch (synthetic)")
	primary := func(_ string) (string, error) {
		return "", sentinel
	}
	res := shell.PathFallbackResolver(primary)
	_, err := res("aws")
	if err == nil {
		t.Fatal("res: nil error, want propagated sentinel")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("err = %v, want errors.Is(sentinel)", err)
	}
}

func TestResolve_PluggedInReplacesPathLookup(t *testing.T) {
	called := 0
	r := &shell.Runner{
		Resolve: func(_ string) (string, error) {
			called++
			return "/bin/echo", nil
		},
	}
	if err := r.Exec(context.Background(), "doesnotexist", "x"); err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if called != 1 {
		t.Errorf("resolver called %d times, want 1", called)
	}
}

func TestExec_ResolverErrorPropagates(t *testing.T) {
	r := &shell.Runner{
		Resolve: func(_ string) (string, error) { return "", errors.New("nope") },
	}
	if err := r.Exec(context.Background(), "x"); err == nil {
		t.Error("expected error")
	}
}

func TestPathResolver_ReportsMissingBinary(t *testing.T) {
	_, err := shell.PathResolver("coily-certainly-does-not-exist-xyzzy")
	if err == nil {
		t.Error("expected error for missing binary")
	}
}

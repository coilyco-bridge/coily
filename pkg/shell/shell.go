// Package shell is the argv-only exec wrapper. Every argument is passed as a
// distinct argv element. No shell string is ever composed. If a downstream
// tool needs a composed command (ssh <host> <remote>), the caller constructs
// the argv slice explicitly and the argv is guarded by policy.ValidateArg at
// the verb layer.
//
// Production runs use FetchingResolver (see tools.go). It consults an
// in-tree manifest (tools.json) and a per-user download cache instead of
// $PATH. PATH is intentionally not trusted - an agent who swaps
// /usr/local/bin/aws is ignored. PathResolver is kept for tests and for
// any caller that genuinely wants $PATH.
package shell

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
)

// Resolver turns a binary name into an executable path.
type Resolver func(bin string) (string, error)

// PathResolver is the default resolver. Looks the binary up on $PATH.
func PathResolver(bin string) (string, error) {
	p, err := exec.LookPath(bin)
	if err != nil {
		return "", fmt.Errorf("shell: %s not found on PATH: %w", bin, err)
	}
	return p, nil
}

// Runner executes subprocesses on behalf of coily verbs. Build one in main
// with the default Resolver and plumb it through to verbs that need it.
type Runner struct {
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader
	// Resolve locates a binary. Defaults to PathResolver when nil.
	Resolve Resolver
}

// ErrEmptyBinary is returned when Exec or Capture is called with "".
var ErrEmptyBinary = errors.New("shell: bin is empty")

func (r *Runner) resolver() Resolver {
	if r.Resolve != nil {
		return r.Resolve
	}
	return PathResolver
}

// Exec runs bin with argv, streaming stdin/stdout/stderr through the Runner's
// fields. Returns the underlying exec error on failure.
func (r *Runner) Exec(ctx context.Context, bin string, argv ...string) error {
	if bin == "" {
		return ErrEmptyBinary
	}
	path, err := r.resolver()(bin)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, path, argv...)
	cmd.Stdout = r.Stdout
	cmd.Stderr = r.Stderr
	cmd.Stdin = r.Stdin
	return cmd.Run()
}

// Capture runs bin with argv and returns stdout as bytes. Stderr is forwarded
// to the Runner's Stderr (defaulting to discard if nil). Useful for verbs
// that parse the subprocess output, like whoami or dns list.
func (r *Runner) Capture(ctx context.Context, bin string, argv ...string) ([]byte, error) {
	if bin == "" {
		return nil, ErrEmptyBinary
	}
	path, err := r.resolver()(bin)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, path, argv...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = r.Stderr
	cmd.Stdin = r.Stdin
	if err := cmd.Run(); err != nil {
		return buf.Bytes(), err
	}
	return buf.Bytes(), nil
}

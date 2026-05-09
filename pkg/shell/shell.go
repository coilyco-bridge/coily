// Package shell is the argv-only exec wrapper. Every argument is passed as a
// distinct argv element. No shell string is ever composed. If a downstream
// tool needs a composed command (ssh <host> <remote>), the caller constructs
// the argv slice explicitly and the argv is guarded by policy.ValidateArg at
// the verb layer.
//
// Binary resolution is via $PATH. coily used to ship a manifest of pinned
// binaries (aws/gh/kubectl/tailscale) fetched from a GitHub Release and
// verified by sha256, with $PATH intentionally bypassed. That machinery was
// removed because the threat it addressed - an attacker with write to a
// $PATH directory but not to $HOME - is a narrow slice that did not justify
// the release-pipeline + manifest + per-tool refresh cadence. Argv
// validation, audit logging, and the lockdown deny list carry the safety
// boundary now; binary authenticity is the host's problem.
package shell

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Resolver turns a binary name into an executable path. Pluggable so tests
// can avoid touching the real $PATH.
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
	// Env, when non-nil, is appended to os.Environ() and passed to the
	// child as cmd.Env. Used by the egress-proxy plumbing to inject
	// HTTPS_PROXY / HTTP_PROXY for the duration of one invocation. When
	// nil, cmd.Env is left unset and the child inherits the parent
	// environment (Go's default).
	Env []string
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
	return r.execIn(ctx, "", bin, argv...)
}

// ExecIn is like Exec but runs the child with cmd.Dir set to dir. Used by
// coily exec when the matched coily.yaml lives in a direct child of cwd:
// the declared argv references paths relative to the child repo, so the
// child must run there, not from cwd. dir is not validated; callers
// ensure it points at a real directory.
func (r *Runner) ExecIn(ctx context.Context, dir, bin string, argv ...string) error {
	return r.execIn(ctx, dir, bin, argv...)
}

func (r *Runner) execIn(ctx context.Context, dir, bin string, argv ...string) error {
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
	if dir != "" {
		cmd.Dir = dir
	}
	if r.Env != nil {
		cmd.Env = append(os.Environ(), r.Env...)
	}
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
	if r.Env != nil {
		cmd.Env = append(os.Environ(), r.Env...)
	}
	if err := cmd.Run(); err != nil {
		return buf.Bytes(), err
	}
	return buf.Bytes(), nil
}

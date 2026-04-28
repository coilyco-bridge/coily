// Package sudo carries the small bits of policy-free plumbing coily
// needs to drive an interactive `sudo` over an ssh transport without
// either (a) carrying a password at rest or (b) leaking it through
// argv/audit. The pattern is described in detail on coily ssh deploy
// in cmd/coily/ops_ssh_deploy.go; this package is the helper.
//
// The hot path on a configured server is `sudo -n` against a NOPASSWD
// sudoers entry, which never reaches this package. ReadPassword runs
// only when -n is denied and an interactive fallback is required.
package sudo

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/term"
)

// ErrNoTTY is returned by ReadPassword when /dev/tty cannot be opened.
// CI/headless invocations should fail fast on this rather than hang
// waiting for input that will never come.
var ErrNoTTY = errors.New("sudo: no controlling tty; cannot prompt for password")

// ReadPassword prompts on /dev/tty (NOT stdin, which the caller will
// reuse to pipe the password into the remote sudo) and returns the
// typed bytes. Callers MUST zero the returned slice once consumed:
//
//	pw, err := sudo.ReadPassword("[sudo via coily] password: ")
//	if err != nil { ... }
//	defer sudo.Zero(pw)
//
// Returning a []byte rather than a string is deliberate: strings are
// immutable and cannot be wiped from memory, which defeats the whole
// point of careful password handling.
func ReadPassword(prompt string) ([]byte, error) {
	fd, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, ErrNoTTY
	}
	defer func() { _ = fd.Close() }()

	if _, err := fmt.Fprint(fd, prompt); err != nil {
		return nil, fmt.Errorf("sudo: write prompt: %w", err)
	}
	pw, err := term.ReadPassword(int(fd.Fd())) //nolint:gosec // fd from os.OpenFile fits in int on every supported platform
	// Always emit a trailing newline so the next stderr line doesn't
	// collide with the prompt position.
	_, _ = fmt.Fprintln(fd)
	if err != nil {
		return nil, fmt.Errorf("sudo: read password: %w", err)
	}
	return pw, nil
}

// Zero overwrites b in place. Use with `defer` after a successful
// ReadPassword to keep the password-bearing buffer alive only as long
// as it is actually needed.
func Zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// PasswordRequired returns true when stderr from a `sudo -n` attempt
// indicates the failure was "password needed" rather than a genuine
// command error. The strings are stable across modern sudo (1.8+) and
// localized only when LC_ALL/LANG is set on the remote; coily's ssh
// transport does not forward locale, so the C-locale strings apply.
func PasswordRequired(stderr string) bool {
	for _, sentinel := range passwordSentinels {
		if contains(stderr, sentinel) {
			return true
		}
	}
	return false
}

var passwordSentinels = []string{
	"a password is required",
	"a terminal is required",
	"sorry, a password is required",
}

// contains is a tiny strings.Contains stand-in to avoid pulling in
// strings just for the substring check; keeps the package's import
// surface to {errors, fmt, os, x/term}.
func contains(haystack, needle string) bool {
	if len(needle) == 0 {
		return true
	}
	if len(haystack) < len(needle) {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

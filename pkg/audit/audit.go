// Package audit writes one JSONL record per coily invocation to an
// append-only log outside the working tree. Per docs/threat-model.md, the
// audit log is the forensic trail if an agent (or a confused human) invokes
// something destructive.
//
// The writer is cheap and safe to call from anywhere, including concurrent
// verb execution. Each call opens the file in O_APPEND|O_CREATE|O_WRONLY
// with 0600 perms, writes one JSON line, and closes. No buffering, no
// global state. If the target directory does not exist it is created with
// 0700.
package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Record is one line in the audit log.
type Record struct {
	Timestamp  time.Time `json:"ts"`
	Version    string    `json:"version,omitempty"`
	Verb       string    `json:"verb"`
	Argv       []string  `json:"argv"`
	ExitCode   int       `json:"exit_code"`
	Error      string    `json:"error,omitempty"`
	SessionID  string    `json:"session_id,omitempty"`
	DurationMS int64     `json:"duration_ms,omitempty"`
}

// Writer appends records to a JSONL file. The zero value is unusable. Use
// NewWriter or set Path explicitly.
type Writer struct {
	// Path is the JSONL file. Must be set.
	Path string
	// Now is used for timestamps. Tests override. Defaults to time.Now.
	Now func() time.Time
	// Session is the value used for SessionID on records that do not set it.
	// Typically populated from CLAUDE_SESSION_ID or similar.
	Session string
}

// NewWriter returns a Writer with Now set to time.Now and Session populated
// from the CLAUDE_SESSION_ID env var if present.
func NewWriter(path string) *Writer {
	return &Writer{
		Path:    path,
		Now:     time.Now,
		Session: os.Getenv("CLAUDE_SESSION_ID"),
	}
}

// ErrPathUnset is returned when Append is called on a Writer with empty Path.
var ErrPathUnset = errors.New("audit: log path not configured")

// Append writes one record as a JSON line. Timestamp and SessionID are
// populated from the Writer if unset on the Record.
func (w *Writer) Append(r Record) error {
	if w.Path == "" {
		return ErrPathUnset
	}
	if r.Timestamp.IsZero() {
		r.Timestamp = w.now()
	}
	if r.SessionID == "" {
		r.SessionID = w.Session
	}

	if err := os.MkdirAll(filepath.Dir(w.Path), 0o700); err != nil {
		return fmt.Errorf("audit: mkdir %s: %w", filepath.Dir(w.Path), err)
	}
	f, err := os.OpenFile(w.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("audit: open %s: %w", w.Path, err)
	}
	defer func() { _ = f.Close() }()
	return json.NewEncoder(f).Encode(&r)
}

// Wrap records an invocation by running fn and logging the result. If fn
// returns an error, Record.Error is set. ExitCode is 0 on success, 1 on
// error. DurationMS is measured around fn. The returned error is whatever
// fn returned, unmodified. Audit failures are written to stderr but do not
// mask fn's error.
func (w *Writer) Wrap(_ context.Context, verb string, argv []string, fn func() error) error {
	start := w.now()
	err := fn()
	rec := Record{
		Verb:       verb,
		Argv:       argv,
		ExitCode:   0,
		DurationMS: w.now().Sub(start).Milliseconds(),
	}
	if err != nil {
		rec.ExitCode = 1
		rec.Error = err.Error()
	}
	if aerr := w.Append(rec); aerr != nil {
		fmt.Fprintf(os.Stderr, "audit: failed to record %s: %v\n", verb, aerr)
	}
	return err
}

func (w *Writer) now() time.Time {
	if w.Now == nil {
		return time.Now()
	}
	return w.Now()
}

// ReadAll decodes every record from r. Useful for tests and for `coily audit
// tail`-style verbs.
func ReadAll(r io.Reader) ([]Record, error) {
	dec := json.NewDecoder(r)
	var out []Record
	for dec.More() {
		var rec Record
		if err := dec.Decode(&rec); err != nil {
			return out, fmt.Errorf("audit: decode: %w", err)
		}
		out = append(out, rec)
	}
	return out, nil
}

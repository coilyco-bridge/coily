// Package audit writes one JSONL record per coily invocation to an
// append-only log outside the working tree. Per docs/threat-model.md, the
// audit log is the forensic trail if an agent (or a confused human) invokes
// something destructive.
//
// Rotation is handled by lumberjack: when the active file reaches MaxSizeMB
// it is renamed with a timestamp suffix and a fresh file is started. Old
// backups past MaxBackups or MaxAgeDays are pruned. All files are written
// with 0600 perms. If the target directory does not exist it is created
// with 0700.
//
// Failure mode: per-call Append errors propagate up. The runtime is expected
// to call Preflight at startup so an unwritable audit dir fails loudly there
// instead of being swallowed across hundreds of invocations.
package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
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
	// MaxSizeMB is the rotation trigger. Zero uses lumberjack's default (100).
	MaxSizeMB int
	// MaxBackups caps the number of rotated files retained. Zero keeps all.
	MaxBackups int
	// MaxAgeDays prunes rotated files older than this. Zero disables.
	MaxAgeDays int
	// Compress gzips rotated files.
	Compress bool

	mu  sync.Mutex
	log *lumberjack.Logger
}

// NewWriter returns a Writer with Now set to time.Now and Session populated
// from the CLAUDE_SESSION_ID env var if present. Rotation fields default to
// zero (lumberjack defaults apply) and can be set by the caller.
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

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&r); err != nil {
		return fmt.Errorf("audit: encode: %w", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if w.log == nil {
		w.log = &lumberjack.Logger{
			Filename:   w.Path,
			MaxSize:    w.MaxSizeMB,
			MaxBackups: w.MaxBackups,
			MaxAge:     w.MaxAgeDays,
			Compress:   w.Compress,
			LocalTime:  true,
		}
	}
	if _, err := w.log.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("audit: write %s: %w", w.Path, err)
	}
	return nil
}

// Close releases the underlying log file. Safe to call multiple times and
// on a Writer that was never used. Call at process exit if you want to be
// tidy; coily's short-lived CLI model doesn't require it.
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.log == nil {
		return nil
	}
	err := w.log.Close()
	w.log = nil
	return err
}

// Wrap records an invocation by running fn and logging the result. If fn
// returns an error, Record.Error is set. ExitCode is 0 on success, 1 on
// error. DurationMS is measured around fn. The returned error is whatever
// fn returned, unmodified. Audit append failures are written to stderr so
// the operator notices, but they do not mask fn's error.
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
		fmt.Fprintf(os.Stderr, "audit: %v\n", aerr)
	}
	return err
}

// Preflight ensures the audit directory exists with 0700 perms and that the
// target path is writable. Call at startup so a broken config blows up
// immediately instead of dropping records silently across the session.
func (w *Writer) Preflight() error {
	if w.Path == "" {
		return ErrPathUnset
	}
	dir := filepath.Dir(w.Path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("audit: mkdir %s: %w", dir, err)
	}
	// Open in append+create mode to verify the path is writable. Don't write
	// anything; just touch the file so a permission failure surfaces here.
	f, err := os.OpenFile(w.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600) // #nosec G304 -- caller-controlled audit path
	if err != nil {
		return fmt.Errorf("audit: open %s: %w", w.Path, err)
	}
	return f.Close()
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

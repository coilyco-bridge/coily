// Package telemetry wires Sentry error capture and log ingest into coily.
//
// When SENTRY_DSN is set, Init boots the Sentry client with EnableLogs=true
// so per-invocation audit lines ship to Sentry as structured logs, and
// CaptureError forwards unhandled errors as Issues. When SENTRY_DSN is
// empty (the default local-dev case), every function in this package is a
// no-op and the CLI makes no network traffic.
//
// The local JSONL audit log in pkg/audit remains the forensic ground
// truth, including the raw argv. This package intentionally does not ship
// argv to Sentry so that accidentally-typed secrets in flag values do not
// leak to a remote service.
package telemetry

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
)

var enabled bool

// Init configures the Sentry client. Returns nil and leaves the package in
// disabled state when SENTRY_DSN is unset. Safe to call once at process
// startup; repeated calls re-initialize the client.
func Init(version string) error {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		return nil
	}
	env := os.Getenv("COILY_ENV")
	if env == "" {
		env = "production"
	}
	err := sentry.Init(sentry.ClientOptions{
		Dsn:         dsn,
		EnableLogs:  true,
		Environment: env,
		Release:     "coily@" + version,
	})
	if err != nil {
		return fmt.Errorf("sentry init: %w", err)
	}
	enabled = true
	return nil
}

// Flush blocks until queued events/logs are delivered or d elapses. Call
// before process exit. Safe to call when disabled (no-op).
func Flush(d time.Duration) {
	if !enabled {
		return
	}
	sentry.Flush(d)
}

// CaptureError forwards err to Sentry as an Issue. No-op when disabled or
// err is nil.
func CaptureError(err error) {
	if !enabled || err == nil {
		return
	}
	sentry.CaptureException(err)
}

// LogInvocation emits one Sentry log entry summarizing a coily verb run.
// Raw argv is omitted by design; the JSONL audit log holds the full
// argument list locally.
func LogInvocation(verb string, argc int, exitCode int, duration time.Duration, errStr string) {
	if !enabled {
		return
	}
	entry := sentry.NewLogger(context.Background()).Info().
		String("verb", verb).
		Int("argc", argc).
		Int("exit_code", exitCode).
		Int64("duration_ms", duration.Milliseconds())
	if host, err := os.Hostname(); err == nil {
		entry = entry.String("host", host)
	}
	if errStr != "" {
		entry = entry.String("error", errStr)
	}
	entry.Emitf("coily %s exit=%d", verb, exitCode)
}

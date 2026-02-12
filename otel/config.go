// Package otel provides OpenTelemetry Logs integration for golog structured logging.
// It implements the golog.Writer and golog.WriterConfig interfaces to bridge
// golog's logging capabilities with OpenTelemetry's Log API for export to
// any OpenTelemetry-compatible backend (OTLP, stdout, etc.).
package otel

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/log"
)

var (
	// UnknownSeverity is the OTel severity used when a golog.Level cannot be mapped
	// to a known OTel severity. This typically happens with custom log levels
	// that don't match the standard golog levels (TRACE, DEBUG, INFO, WARN, ERROR, FATAL).
	// Defaults to log.SeverityError to ensure unknown levels are treated as errors.
	UnknownSeverity = log.SeverityError

	// FlushTimeout specifies how long to wait when flushing log records
	// before giving up. This is used when the application is shutting down
	// to ensure pending records are exported before termination.
	// Defaults to 5 seconds.
	FlushTimeout time.Duration = 5 * time.Second

	// withoutLoggingCtxKey is used as a context key to mark contexts that should
	// not send log messages to OpenTelemetry.
	withoutLoggingCtxKey int
)

// ContextWithoutLogging returns a new context derived from parent that disables
// OpenTelemetry logging for all log levels. When a logger uses this context, no log
// records will be emitted to OpenTelemetry regardless of their level.
//
// The function is idempotent - if the parent context already has OTel logging
// disabled, the same context is returned without modification.
func ContextWithoutLogging(parent context.Context) context.Context {
	if IsContextWithoutLogging(parent) {
		return parent
	}
	return context.WithValue(parent, &withoutLoggingCtxKey, struct{}{})
}

// IsContextWithoutLogging checks whether the given context has OTel logging
// disabled. It returns true if the context was created by ContextWithoutLogging.
//
// Returns false if ctx is nil or if OTel logging is not disabled.
func IsContextWithoutLogging(ctx context.Context) bool {
	return ctx != nil && ctx.Value(&withoutLoggingCtxKey) != nil
}

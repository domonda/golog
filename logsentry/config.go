// Package logsentry provides Sentry integration for golog structured logging.
// It implements the golog.Writer and golog.WriterConfig interfaces to bridge
// golog's logging capabilities with Sentry's error tracking and monitoring system.
package logsentry

import (
	"context"
	"time"

	"github.com/getsentry/sentry-go"
)

var (
	// UnknownLevel is the Sentry level used when a golog.Level cannot be mapped
	// to a known Sentry level. This typically happens with custom log levels
	// that don't match the standard golog levels (TRACE, DEBUG, INFO, WARN, ERROR, FATAL).
	// Defaults to sentry.LevelError to ensure unknown levels are treated as errors.
	UnknownLevel = sentry.LevelError

	// FlushTimeout specifies how long to wait when flushing Sentry events
	// before giving up. This is used when the application is shutting down
	// to ensure pending events are sent to Sentry before termination.
	// Defaults to 3 seconds, which should be sufficient for most network conditions.
	FlushTimeout time.Duration = 3 * time.Second

	// withoutLoggingCtxKey is used as a context key to mark contexts that should
	// not send log messages to Sentry. This allows selective disabling of Sentry
	// logging for specific request contexts or operations.
	withoutLoggingCtxKey int
)

// ContextWithoutLogging returns a new context derived from parent that disables
// Sentry logging for all log levels. When a logger uses this context, no log
// messages will be sent to Sentry regardless of their level.
//
// This is useful for:
// - Testing scenarios where you don't want to pollute Sentry with test data
// - Background operations where Sentry logging is not desired
// - Sensitive operations where logging to external services should be avoided
//
// The function is idempotent - if the parent context already has Sentry logging
// disabled, the same context is returned without modification.
//
// Example:
//
//	ctx := logsentry.ContextWithoutLogging(context.Background())
//	logger.WithContext(ctx).Error("This won't go to Sentry").Log()
func ContextWithoutLogging(parent context.Context) context.Context {
	if IsContextWithoutLogging(parent) {
		return parent
	}
	return context.WithValue(parent, &withoutLoggingCtxKey, struct{}{})
}

// IsContextWithoutLogging checks whether the given context has Sentry logging
// disabled. It returns true if the context was created by ContextWithoutLogging
// or if it contains the same context key that disables Sentry logging.
//
// This function is used internally by the WriterConfig to determine whether
// to skip sending log messages to Sentry for a given context.
//
// Returns false if ctx is nil or if Sentry logging is not disabled.
//
// Example:
//
//	ctx := logsentry.ContextWithoutLogging(context.Background())
//	if logsentry.IsContextWithoutLogging(ctx) {
//	    // Sentry logging is disabled for this context
//	}
func IsContextWithoutLogging(ctx context.Context) bool {
	return ctx != nil && ctx.Value(&withoutLoggingCtxKey) != nil
}

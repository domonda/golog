package logsentry

import (
	"context"
	"time"

	"github.com/getsentry/sentry-go"

	"github.com/domonda/golog"
)

var (
	// UnknownLevel will be used if a golog.Level
	// can't be mapped to a sentry.LevelError.
	UnknownLevel = sentry.LevelError

	FlushTimeout time.Duration = 3 * time.Second

	withoutLoggingCtxKey int
)

const (
	nopWriter golog.NopWriter = "logsentry.nopWriter"
)

// ContextWithoutLogging returns a new context with
// Sentry logging disabled for all levels.
func ContextWithoutLogging(parent context.Context) context.Context {
	if IsContextWithoutLogging(parent) {
		return parent
	}
	return context.WithValue(parent, &withoutLoggingCtxKey, struct{}{})
}

// IsContextWithoutLogging returns true if the passed
// context was returned from ContextWithoutLogging,
// which means Sentry logging disabled for all levels.
func IsContextWithoutLogging(ctx context.Context) bool {
	return ctx != nil && ctx.Value(&withoutLoggingCtxKey) != nil
}

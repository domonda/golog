package golog

import (
	"context"
	"time"
)

// LevelDecider is implemented to decide
// if a Level is active together with a given context.
type LevelDecider interface {
	// IsActive returns if a Level is active together with a given context.
	// It's valid to pass a nil context.
	IsActive(context.Context, Level) bool
}

var deciderCtxKey int

// ContextWithLevelDecider returns a new context with the passed LevelDecider
// added to the parent.
// Logger methods with a context argument and known level will
// check if the passed context has a LevelDecider and call
// its IsActive method to decide if the following message should be logged.
// See also IsActiveContext.
//
// LevelFilter implements LevelDecider and can be added directly to a context.
// Disable all levels below the default info configuration for a context:
//
//	ctx = golog.ContextWithLevelDecider(ctx, log.Levels.Info.FilterOutBelow())
func ContextWithLevelDecider(parent context.Context, decider LevelDecider) context.Context {
	return context.WithValue(parent, &deciderCtxKey, decider)
}

// ContextWithoutLogging returns a new context with logging
// disabled for all levels.
func ContextWithoutLogging(parent context.Context) context.Context {
	return ContextWithLevelDecider(parent, BoolLevelDecider(false))
}

// IsActiveContext returns true by default except when a
// LevelDecider was added to the context using ContextWithLevelDecider,
// then the result of its IsActive method will be returned.
// It's valid to pass a nil context which will return true.
func IsActiveContext(ctx context.Context, level Level) bool {
	if ctx == nil {
		return true
	}
	if decider, _ := ctx.Value(&deciderCtxKey).(LevelDecider); decider != nil {
		return decider.IsActive(ctx, level)
	}
	return true
}

// BoolLevelDecider implements LevelDecider by
// always returning the underlying bool value from its IsActive method
// independent of the arguments.
type BoolLevelDecider bool

// IsActive always returns the underlying bool value of the receiver
// independent of the arguments.
func (b BoolLevelDecider) IsActive(context.Context, Level) bool {
	return bool(b)
}

var timestampCtxKey int

// ContextWithTimestamp returns a new context with the passed timestamp
// added to the parent.
// Logger methods with a context will use the timestamp from the context
// instead of the current time.
func ContextWithTimestamp(parent context.Context, timestamp time.Time) context.Context {
	return context.WithValue(parent, &timestampCtxKey, timestamp)
}

// TimestampFromContext returns the timestamp from the passed context
// or a zero time if no timestamp was set.
func TimestampFromContext(ctx context.Context) time.Time {
	timestamp, _ := ctx.Value(&timestampCtxKey).(time.Time)
	return timestamp
}

// Timestamp returns the timestamp from the passed context
// or the current time if no timestamp was set.
func Timestamp(ctx context.Context) time.Time {
	if timestamp, ok := ctx.Value(&timestampCtxKey).(time.Time); ok {
		return timestamp
	}
	return time.Now()
}

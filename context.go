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

type timestampCtxKey struct{}

// ContextWithTimestamp returns a new context derived from parent that
// carries the passed timestamp. Logger methods that accept a context
// will use this timestamp instead of [time.Now] when emitting log
// records, so the same value can be used across an entire request
// or transaction.
//
// The timestamp may be passed as either a [time.Time] or a [Timestamp];
// in both cases the underlying time.Time is stored in the context.
// A zero timestamp is stored as-is and will be returned by
// [TimestampFromContext], but [TimestampFromContextOrNow] will treat
// it as "not set" and fall back to the current time.
func ContextWithTimestamp[T time.Time | Timestamp](parent context.Context, timestamp T) context.Context {
	if timestamp, ok := any(timestamp).(Timestamp); ok {
		return context.WithValue(parent, timestampCtxKey{}, timestamp.Time)
	}
	return context.WithValue(parent, timestampCtxKey{}, timestamp)
}

// TimestampFromContext returns the timestamp previously stored in ctx
// by [ContextWithTimestamp], or the zero [time.Time] if none was set.
// Use [TimestampFromContextOrNow] if you want the current time as a
// fallback instead of the zero value.
func TimestampFromContext(ctx context.Context) time.Time {
	if t, ok := ctx.Value(timestampCtxKey{}).(time.Time); ok {
		return t
	}
	return time.Time{}
}

// TimestampFromContextOrNow returns the timestamp previously stored in
// ctx by [ContextWithTimestamp], or the result of [time.Now] if no
// timestamp was set or the stored timestamp is the zero value.
func TimestampFromContextOrNow(ctx context.Context) time.Time {
	if t := TimestampFromContext(ctx); !t.IsZero() {
		return t
	}
	return time.Now()
}

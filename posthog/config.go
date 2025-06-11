package posthog

import "context"

var (
	withoutLoggingCtxKey int
)

// ContextWithoutLogging returns a new context with
// PostHog logging disabled for all levels.
func ContextWithoutLogging(parent context.Context) context.Context {
	if IsContextWithoutLogging(parent) {
		return parent
	}
	return context.WithValue(parent, &withoutLoggingCtxKey, struct{}{})
}

// IsContextWithoutLogging returns true if the passed
// context was returned from ContextWithoutLogging,
// which means PostHog logging disabled for all levels.
func IsContextWithoutLogging(ctx context.Context) bool {
	return ctx != nil && ctx.Value(&withoutLoggingCtxKey) != nil
}

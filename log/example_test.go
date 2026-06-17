package log_test

import (
	"context"

	"github.com/domonda/golog"
	"github.com/domonda/golog/log"
)

// Example shows the ready-to-use package logger. No setup is required: the
// logger is configured from the LOG_LEVEL environment variable and writes
// colored text on a terminal or JSON when the output is redirected.
func Example() {
	log.Info("Application started").Log()
	log.Error("Something went wrong").
		Str("component", "auth").
		Log()
}

// Example_context attaches request-scoped attributes to a context; the *Ctx
// logging methods include them automatically, so correlation IDs propagate
// without manual plumbing.
func Example_context() {
	ctx := golog.ContextWithAttribs(context.Background(),
		golog.NewString("request_id", "req-123"),
		golog.NewString("user_id", "user-456"),
	)
	log.InfoCtx(ctx, "processing request").Log()
}

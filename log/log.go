package log

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/domonda/golog"
)

// HTTPMiddlewareHandler returns a HTTP middleware handler that passes through a UUID requestID.
// The requestID will be added as UUID golog.Attrib to the http.Request before calling the next handler.
// If available the X-Request-ID or X-Correlation-ID HTTP request header will be used as requestID.
// It has to be a valid UUID in the format "994d5800-afca-401f-9c2f-d9e3e106e9ef".
// If the request has no requestID, then a random v4 UUID will be used.
// The requestID will also be set at the http.ResponseWriter as X-Request-ID header
// before calling the next handler, which has a chance to change it.
// If onlyHeaders are passed then only those headers are logged if available,
// or pass golog.HTTPNoHeaders to disable header logging.
// To disable logging of the request at all and just pass through
// the requestID pass golog.LevelInvalid as log level.
// See also HTTPMiddlewareFunc.
func HTTPMiddlewareHandler(next http.Handler, level golog.Level, message string, onlyHeaders ...string) http.Handler {
	return golog.HTTPMiddlewareHandler(next, Logger, level, message, onlyHeaders...)
}

// HTTPMiddlewareFunc returns a HTTP middleware function that passes through a UUID requestID.
// The requestID will be added as UUID golog.Attrib to the http.Request before calling the next handler.
// If available the X-Request-ID or X-Correlation-ID HTTP request header will be used as requestID.
// It has to be a valid UUID in the format "994d5800-afca-401f-9c2f-d9e3e106e9ef".
// If the request has no requestID, then a random v4 UUID will be used.
// The requestID will also be set at the http.ResponseWriter as X-Request-ID header
// before calling the next handler, which has a chance to change it.
// If onlyHeaders are passed then only those headers are logged if available,
// or pass golog.HTTPNoHeaders to disable header logging.
// To disable logging of the request at all and just pass through
// the requestID pass golog.LevelInvalid as log level.
// Compatible with github.com/gorilla/mux.MiddlewareFunc.
// See also HTTPMiddlewareHandler.
func HTTPMiddlewareFunc(level golog.Level, message string, onlyHeaders ...string) func(next http.Handler) http.Handler {
	return golog.HTTPMiddlewareFunc(Logger, level, message, onlyHeaders...)
}

func WithValues(values ...golog.Attrib) *golog.Logger {
	return Logger.WithAttribs(values...)
}

func WithLevelFilter(filter golog.LevelFilter) *golog.Logger {
	return Logger.WithLevelFilter(filter)
}

func WithPrefix(prefix string) *golog.Logger {
	return Logger.WithPrefix(prefix)
}

// WithCtx returns a new golog.Logger with the
// PerMessageValues from a context logger appended to this package's Logger.
func WithCtx(ctx context.Context) *golog.Logger {
	return Logger.WithCtx(ctx)
}

// With returns a new Message that can be used to record
// the prefix for a sub-logger.
//
// Example:
//
//	log := log.With().UUID("requestID", requestID).SubLogger()
func With() *golog.Message {
	return Logger.With()
}

// Flush unwritten logs
func Flush() {
	Logger.Flush()
}

// NewMessageAt starts a new message logged with the time t
func NewMessageAt(ctx context.Context, t time.Time, level golog.Level, text string) *golog.Message {
	return Logger.NewMessageAt(ctx, t, level, text)
}

// NewMessage starts a new message
func NewMessage(ctx context.Context, level golog.Level, text string) *golog.Message {
	return Logger.NewMessage(ctx, level, text)
}

// NewMessagef starts a new message formatted using fmt.Sprintf
func NewMessagef(ctx context.Context, level golog.Level, format string, args ...any) *golog.Message {
	return Logger.NewMessagef(ctx, level, format, args...)
}

// FatalAndPanic is a shortcut for Fatal(fmt.Sprint(p)).LogAndPanic()
func FatalAndPanic(p any) {
	Logger.Fatal(fmt.Sprint(p)).Log()
	panic(p)
}

func Fatal(text string) *golog.Message {
	return Logger.Fatal(text)
}

func FatalCtx(ctx context.Context, text string) *golog.Message {
	return Logger.FatalCtx(ctx, text)
}

func Fatalf(format string, args ...any) *golog.Message {
	return Logger.Fatalf(format, args...)
}

func FatalfCtx(ctx context.Context, format string, args ...any) *golog.Message {
	return Logger.FatalfCtx(ctx, format, args...)
}

func Error(text string) *golog.Message {
	return Logger.Error(text)
}

func ErrorCtx(ctx context.Context, text string) *golog.Message {
	return Logger.ErrorCtx(ctx, text)
}

func Errorf(format string, args ...any) *golog.Message {
	return Logger.Errorf(format, args...)
}

func ErrorfCtx(ctx context.Context, format string, args ...any) *golog.Message {
	return Logger.ErrorfCtx(ctx, format, args...)
}

func Warn(text string) *golog.Message {
	return Logger.Warn(text)
}

func WarnCtx(ctx context.Context, text string) *golog.Message {
	return Logger.WarnCtx(ctx, text)
}

func Warnf(format string, args ...any) *golog.Message {
	return Logger.Warnf(format, args...)
}

func WarnfCtx(ctx context.Context, format string, args ...any) *golog.Message {
	return Logger.WarnfCtx(ctx, format, args...)
}

func Info(text string) *golog.Message {
	return Logger.Info(text)
}

func InfoCtx(ctx context.Context, text string) *golog.Message {
	return Logger.InfoCtx(ctx, text)
}

func Infof(format string, args ...any) *golog.Message {
	return Logger.Infof(format, args...)
}

func InfofCtx(ctx context.Context, format string, args ...any) *golog.Message {
	return Logger.InfofCtx(ctx, format, args...)
}

func Debug(text string) *golog.Message {
	return Logger.Debug(text)
}

func DebugCtx(ctx context.Context, text string) *golog.Message {
	return Logger.DebugCtx(ctx, text)
}

func Debugf(format string, args ...any) *golog.Message {
	return Logger.Debugf(format, args...)
}

func DebugfCtx(ctx context.Context, format string, args ...any) *golog.Message {
	return Logger.DebugfCtx(ctx, format, args...)
}

func Trace(text string) *golog.Message {
	return Logger.Trace(text)
}

func TraceCtx(ctx context.Context, text string) *golog.Message {
	return Logger.TraceCtx(ctx, text)
}

func Tracef(format string, args ...any) *golog.Message {
	return Logger.Tracef(format, args...)
}

func TracefCtx(ctx context.Context, format string, args ...any) *golog.Message {
	return Logger.TracefCtx(ctx, format, args...)
}

func NewLevelWriter(level golog.Level) *golog.LevelWriter {
	return Logger.NewLevelWriter(level)
}

func FatalWriter() *golog.LevelWriter {
	return Logger.FatalWriter()
}

func ErrorWriter() *golog.LevelWriter {
	return Logger.ErrorWriter()
}

func WarnWriter() *golog.LevelWriter {
	return Logger.WarnWriter()
}

func InfoWriter() *golog.LevelWriter {
	return Logger.InfoWriter()
}

func DebugWriter() *golog.LevelWriter {
	return Logger.DebugWriter()
}

func TraceWriter() *golog.LevelWriter {
	return Logger.TraceWriter()
}

package log

import (
	"context"
	"fmt"
	"net/http"

	"github.com/domonda/golog"
)

// HTTPMiddlewareHandler returns a HTTP middleware handler that passes through a UUID requestID value.
// The requestID will be added as value to the http.Request before calling the next handler.
// If available the X-Request-ID or X-Correlation-ID HTTP request header will be used as requestID.
// It has to be a valid UUID in the format "994d5800-afca-401f-9c2f-d9e3e106e9ef".
// If the request has no requestID, then a random v4 UUID will be used.
// The requestID will also be set at the http.ResponseWriter as X-Request-ID header
// before calling the next handler, which has a chance to change it.
// If restrictHeaders are passed, then only those headers are logged if available,
// or pass golog.HTTPNoHeaders to disable header logging.
// To disable logging of the request at all and just pass through
// the requestID pass golog.LevelInvalid as log level.
// See also HTTPMiddlewareFunc.
func HTTPMiddlewareHandler(next http.Handler, level golog.Level, message string, restrictHeaders ...string) http.Handler {
	return golog.HTTPMiddlewareHandler(next, Logger, level, message, restrictHeaders...)
}

// HTTPMiddlewareFunc returns a HTTP middleware function that passes through a UUID requestID value.
// The requestID will be added as value to the http.Request before calling the next handler.
// If available the X-Request-ID or X-Correlation-ID HTTP request header will be used as requestID.
// It has to be a valid UUID in the format "994d5800-afca-401f-9c2f-d9e3e106e9ef".
// If the request has no requestID, then a random v4 UUID will be used.
// The requestID will also be set at the http.ResponseWriter as X-Request-ID header
// before calling the next handler, which has a chance to change it.
// If restrictHeaders are passed, then only those headers are logged if available,
// or pass golog.HTTPNoHeaders to disable header logging.
// To disable logging of the request at all and just pass through
// the requestID pass golog.LevelInvalid as log level.
// Compatible with github.com/gorilla/mux.MiddlewareFunc.
// See also HTTPMiddlewareHandler.
func HTTPMiddlewareFunc(level golog.Level, message string, restrictHeaders ...string) func(next http.Handler) http.Handler {
	return golog.HTTPMiddlewareFunc(Logger, level, message, restrictHeaders...)
}

func WithValues(values ...golog.Value) *golog.Logger {
	return Logger.WithValues(values...)
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
//   log := log.With().UUID("requestID", requestID).SubLogger()
func With() *golog.Message {
	return Logger.With()
}

// Flush unwritten logs
func Flush() {
	Logger.Flush()
}

// FatalAndPanic is a shortcut for Fatal(fmt.Sprint(p)).LogAndPanic()
func FatalAndPanic(p any) {
	Logger.Fatal(fmt.Sprint(p)).Log()
	panic(p)
}

func Fatal(text string) *golog.Message {
	return Logger.Fatal(text)
}

func Fatalf(format string, args ...any) *golog.Message {
	return Logger.Fatalf(format, args...)
}

func Error(text string) *golog.Message {
	return Logger.Error(text)
}

func Errorf(format string, args ...any) *golog.Message {
	return Logger.Errorf(format, args...)
}

func Warn(text string) *golog.Message {
	return Logger.Warn(text)
}

func Warnf(format string, args ...any) *golog.Message {
	return Logger.Warnf(format, args...)
}

func Info(text string) *golog.Message {
	return Logger.Info(text)
}

func Infof(format string, args ...any) *golog.Message {
	return Logger.Infof(format, args...)
}

func Debug(text string) *golog.Message {
	return Logger.Debug(text)
}

func Debugf(format string, args ...any) *golog.Message {
	return Logger.Debugf(format, args...)
}

func Trace(text string) *golog.Message {
	return Logger.Trace(text)
}

func Tracef(format string, args ...any) *golog.Message {
	return Logger.Tracef(format, args...)
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

package log

import (
	"context"
	"fmt"
	"net/http"

	"github.com/domonda/golog"
)

// AddLoggerToContext returns a new context.Context with the default Logger.
// See WithContext
func AddLoggerToContext(parent context.Context) context.Context {
	return Logger.AddToContext(parent)
}

// // WithContext returns a golog.Logger if ctx has one
// // or the default Logger variable.
// // This behaviour differs from golog.FromContext
// // that returns a nil Logger if ctx has none.
// // See Context
// func WithContext(ctx context.Context) *golog.Logger {
// 	if l := golog.LoggerFromContext(ctx); l != nil {
// 		return l
// 	}
// 	return Logger
// }

// Request creates a new requestLogger with a UUID requestID,
// logs the passed request's metdata with a golog.HTTPRequestMessage (default "HTTP request")
// using golog.HTTPRequestLevel (default golog.DefaultLevels.Info)
// and returns the requestLogger and requestID together with a new context.Context
// derived from the request.Context() that has requestLogger added as value,
// so functions receiving this ctx can get the requestLogger
// by calling WithContext(ctx).
// If available the X-Request-ID or X-Correlation-ID HTTP request header will be used as requestID.
// It has to be a valid UUID in the format "994d5800-afca-401f-9c2f-d9e3e106e9ef".
// Else a random v4 UUID will be generated as requestID.
//
// Example:
//   func ServeHTTP(w http.ResponseWriter, r *http.Request) {
//       log, requestID, ctx := log.Request(r)
//       log.Debug("Using request sub-logger").Log()
//       doSomething(ctx)
//       ...
//   }
//   func doSomething(ctx context.Context) {
//       // Logger from ctx will implicitely add the requestID
//       // value to the folloing log message:
//       log.WithContext(ctx).Info("doSomething").Log()
//       ...
//   }
func Request(request *http.Request) (requestLogger *golog.Logger, requestID [16]byte, ctx context.Context) {
	xRequestID := request.Header.Get("X-Request-ID")
	if xRequestID == "" {
		xRequestID = request.Header.Get("X-Correlation-ID")
	}
	requestID, err := golog.ParseUUID(xRequestID)
	if err != nil {
		requestID = golog.NewUUID()
	}
	requestLogger, ctx = Logger.LogRequestWithIDContext(requestID, request)
	return requestLogger, requestID, ctx
}

// HTTPMiddlewareFunc returns a HTTP handler middleware function that
// creates a new sub-logger with a requestID (UUID),
// logs the request metadata using it,
// and adds it as value to the context of the request
// so it can be retrieved with WithContext(request.Context())
// in further handlers after this middleware handler.
// If available the X-Request-ID or X-Correlation-ID HTTP request header will be used as requestID.
// It has to be a valid UUID in the format "994d5800-afca-401f-9c2f-d9e3e106e9ef".
// Else a random v4 UUID will be generated as requestID.
// The requestID will also be set at the http.ResponseWriter as X-Request-ID header
// before calling the next handler, which has a chance to change it.
// Compatible with github.com/gorilla/mux.MiddlewareFunc
func HTTPMiddlewareFunc() func(next http.Handler) http.Handler {
	return Logger.HTTPMiddlewareFunc()
}

func WithValues(values ...golog.NamedValue) *golog.Logger {
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
//   log := log.With().UUID("requestID", requestID).NewLogger()
func With() *golog.Message {
	return Logger.With()
}

// Flush unwritten logs
func Flush() {
	Logger.Flush()
}

// FatalAndPanic is a shortcut for Fatal(fmt.Sprint(p)).LogAndPanic()
func FatalAndPanic(p interface{}) {
	Logger.Fatal(fmt.Sprint(p)).Log()
	panic(p)
}

func Fatal(text string) *golog.Message {
	return Logger.Fatal(text)
}

func Fatalf(format string, args ...interface{}) *golog.Message {
	return Logger.Fatalf(format, args...)
}

func Error(text string) *golog.Message {
	return Logger.Error(text)
}

func Errorf(format string, args ...interface{}) *golog.Message {
	return Logger.Errorf(format, args...)
}

func Warn(text string) *golog.Message {
	return Logger.Warn(text)
}

func Warnf(format string, args ...interface{}) *golog.Message {
	return Logger.Warnf(format, args...)
}

func Info(text string) *golog.Message {
	return Logger.Info(text)
}

func Infof(format string, args ...interface{}) *golog.Message {
	return Logger.Infof(format, args...)
}

func Debug(text string) *golog.Message {
	return Logger.Debug(text)
}

func Debugf(format string, args ...interface{}) *golog.Message {
	return Logger.Debugf(format, args...)
}

func Trace(text string) *golog.Message {
	return Logger.Trace(text)
}

func Tracef(format string, args ...interface{}) *golog.Message {
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

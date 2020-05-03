package golog

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Logger struct {
	config           Config
	prefix           string
	perMessageValues []NamedValue
	mtx              sync.Mutex
}

// NewLogger returns a Logger with the given config and perMessageValues.
// If config is nil, then a nil Logger will be returned.
// A nil Logger is still valid to use but will not log anything.
// Any perMessageValues will be repeated for every new log message.
func NewLogger(config Config, perMessageValues ...NamedValue) *Logger {
	if config == nil {
		return nil
	}
	return &Logger{
		config:           config,
		perMessageValues: perMessageValues,
	}
}

// NewLogger returns a Logger with the given config, prefix, and perMessageValues.
// If config is nil, then a nil Logger will be returned.
// A nil Logger is still valid to use but will not log anything.
// Every log message will begin with the passed prefix.
// Any perMessageValues will be repeated for every new log message.
func NewLoggerWithPrefix(config Config, prefix string, perMessageValues ...NamedValue) *Logger {
	if config == nil {
		return nil
	}
	return &Logger{
		config:           config,
		prefix:           prefix,
		perMessageValues: perMessageValues,
	}
}

type ctxKey struct{}

// LoggerFromContext returns a Logger if ctx has one
// or else a nil Logger wich is still valid to use
// but does not produce any log output.
// See Logger.Context
func LoggerFromContext(ctx context.Context) *Logger {
	l, _ := ctx.Value(ctxKey{}).(*Logger)
	return l
}

// LoggerFromRequest returns a Logger if the request context has one
// or else a nil Logger wich is still valid to use
// but does not produce any log output.
func LoggerFromRequest(request *http.Request) *Logger {
	return LoggerFromContext(request.Context())
}

// AddToContext returns a new context.Context with this Logger.
// If this Logger is a nil Logger, then the passed in
// parent context is returned.
// See LoggerFromContext
func (l *Logger) AddToContext(parent context.Context) context.Context {
	if l == nil {
		return parent
	}
	return context.WithValue(parent, ctxKey{}, l)
}

// AddToRequest returs a shallow copy of the passed request
// with the logger added as value to its context
// so LoggerFromRequest(request) will return it.
//
// Example:
//   // Add userID to request logger
//   request = golog.LoggerFromRequest(request).
//       With().
//       UUID("userID", userID).
//       NewLogger().
//       AddToRequest(request)
func (l *Logger) AddToRequest(request *http.Request) *http.Request {
	if l == nil {
		return request
	}
	return request.WithContext(l.AddToContext(request.Context()))
}

// LogRequestWithID creates a new requestLogger with a new requestID (UUID),
// logs the passed request's metdata with a golog.HTTPRequestMessage (default "HTTP request")
// using golog.HTTPRequestLevel (default golog.DefaultLevels.Info)
// and returns the requestLogger.
// If restrictHeaders are passed, then only those headers are logged if available.
// To disable header logging, pass an impossible header name.
//
// Example:
//   func ServeHTTP(w http.ResponseWriter, r *http.Request) {
//       log := globalLogger.LogRequestWithID(golog.NewUUID(), r)
//       log.Debug("Using request sub-logger").Log()
//       ...
//   }
func (l *Logger) LogRequestWithID(requestID interface{}, requestToLog *http.Request, restrictHeaders ...string) (requestLogger *Logger) {
	requestLogger = l.With().Val("requestID", requestID).SubLogger()
	requestLogger.NewMessage(*HTTPRequestLevel, HTTPRequestMessage).Request(requestToLog, restrictHeaders...).Log()
	return requestLogger
}

// LogRequestWithIDContext creates a new requestLogger with a new requestID (UUID),
// logs the passed request's metdata with a golog.HTTPRequestMessage (default "HTTP request")
// using golog.HTTPRequestLevel (default golog.DefaultLevels.Info)
// and returns the requestLogger together with a new context.Context
// derived from the request.Context() that has requestLogger added as value,
// so functions receiving this ctx can get the requestLogger
// by calling LoggerFromContext(ctx).
// If restrictHeaders are passed, then only those headers are logged if available.
// To disable header logging, pass an impossible header name.
//
// Example:
//   func ServeHTTP(w http.ResponseWriter, r *http.Request) {
//       log, ctx := globalLogger.LogRequestWithIDContext(golog.NewUUID(), r)
//       log.Debug("Using request sub-logger").Log()
//       doSomething(ctx)
//       ...
//   }
func (l *Logger) LogRequestWithIDContext(requestID interface{}, requestToLog *http.Request, restrictHeaders ...string) (requestLogger *Logger, ctx context.Context) {
	requestLogger = l.LogRequestWithID(requestID, requestToLog, restrictHeaders...)
	ctx = requestLogger.AddToContext(requestToLog.Context())
	return requestLogger, ctx
}

// HTTPMiddlewareFunc returns a HTTP handler middleware function that
// creates a new sub-logger with a UUID requestID,
// logs the request metadata using it,
// and adds it as value to the context of the request
// so it can be retrieved with LoggerFromContext(request.Context())
// in further handlers after this middleware handler.
// If restrictHeaders are passed, then only those headers are logged if available.
// To disable header logging, pass an impossible header name.
// If available the X-Request-ID or X-Correlation-ID HTTP request header will be used as requestID.
// It has to be a valid UUID in the format "994d5800-afca-401f-9c2f-d9e3e106e9ef".
// Else a random v4 UUID will be generated as requestID.
// The requestID will also be set at the http.ResponseWriter as X-Request-ID header
// before calling the next handler, which has a chance to change it.
// Compatible with github.com/gorilla/mux.MiddlewareFunc
func (l *Logger) HTTPMiddlewareFunc(restrictHeaders ...string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				xRequestID := r.Header.Get("X-Request-ID")
				if xRequestID == "" {
					xRequestID = r.Header.Get("X-Correlation-ID")
				}
				requestID, err := ParseUUID(xRequestID)
				if err != nil {
					requestID = NewUUID()
				}
				w.Header().Set("X-Request-ID", FormatUUID(requestID))
				requestLogger := l.LogRequestWithID(requestID, r, restrictHeaders...)
				next.ServeHTTP(w, requestLogger.AddToRequest(r))
			},
		)
	}
}

// WithValues returns a new Logger with the passed
// perMessageValues appended to the existing perMessageValues.
func (l *Logger) WithValues(perMessageValues ...NamedValue) *Logger {
	if l == nil || len(perMessageValues) == 0 {
		return l
	}
	return &Logger{
		config:           l.config,
		prefix:           l.prefix,
		perMessageValues: append(l.perMessageValues, perMessageValues...),
	}
}

// WithCtx returns a new sub Logger with the
// PerMessageValues from a context logger appended to the existing PerMessageValues,
// if there was a Logger added as value to the context,
// else l is returned unchanged.
func (l *Logger) WithCtx(ctx context.Context) *Logger {
	return l.WithValues(LoggerFromContext(ctx).PerMessageValues()...)
}

func (l *Logger) WithPrefix(prefix string) *Logger {
	if l == nil {
		return nil
	}
	return &Logger{
		config:           l.config,
		perMessageValues: l.perMessageValues,
		prefix:           prefix,
	}
}

func (l *Logger) WithLevelFilter(filter LevelFilter) *Logger {
	if l == nil {
		return nil
	}
	return &Logger{
		config:           NewDerivedConfig(&l.config, filter),
		prefix:           l.prefix,
		perMessageValues: l.perMessageValues,
	}
}

// With returns a new Message that can be used to record
// the prefix for a sub-logger.
//
// Example:
//   log := log.With().UUID("requestID", requestID).NewLogger()
func (l *Logger) With() *Message {
	if l == nil {
		return nil
	}
	return newMessage(l, new(recordingFormatter), "")
}

func (l *Logger) Config() Config {
	if l == nil {
		return nil
	}
	return l.config
}

func (l *Logger) Prefix() string {
	if l == nil {
		return ""
	}
	return l.prefix
}

func (l *Logger) PerMessageValues() []NamedValue {
	if l == nil {
		return nil
	}
	return l.perMessageValues
}

func (l *Logger) IsActive(level Level) bool {
	if l == nil {
		return false
	}
	l.mtx.Lock()
	active := l.config.IsActive(level)
	l.mtx.Unlock()
	return active
}

// Flush unwritten logs
func (l *Logger) Flush() {
	if l == nil {
		return
	}
	l.config.Formatter().FlushUnderlying()
}

func (l *Logger) NewMessageAt(t time.Time, level Level, text string) *Message {
	if !l.IsActive(level) {
		return nil
	}
	m := newMessage(l, l.config.Formatter().Clone(level), text)
	m.formatter.WriteText(t, l.config.Levels(), level, l.prefix, text)
	for _, namedValue := range l.perMessageValues {
		namedValue.Log(m)
	}
	return m
}

func (l *Logger) NewMessage(level Level, text string) *Message {
	if !l.IsActive(level) {
		return nil
	}
	return l.NewMessageAt(time.Now(), level, text)
}

func (l *Logger) NewMessagef(level Level, format string, args ...interface{}) *Message {
	if !l.IsActive(level) {
		return nil
	}
	return l.NewMessageAt(time.Now(), level, fmt.Sprintf(format, args...))
}

// FatalAndPanic is a shortcut for Fatal(fmt.Sprint(p)).LogAndPanic()
func (l *Logger) FatalAndPanic(p interface{}) {
	l.Fatal(fmt.Sprint(p)).Log()
	panic(p)
}

func (l *Logger) Fatal(text string) *Message {
	return l.NewMessage(l.config.Fatal(), text)
}

func (l *Logger) Fatalf(format string, args ...interface{}) *Message {
	return l.NewMessagef(l.config.Fatal(), format, args...)
}

func (l *Logger) Error(text string) *Message {
	return l.NewMessage(l.config.Error(), text)
}

// Errorf uses fmt.Errorf underneath to support Go 1.13 wrapped error formatting with %w
func (l *Logger) Errorf(format string, args ...interface{}) *Message {
	return l.NewMessage(l.config.Error(), fmt.Errorf(format, args...).Error())
}

func (l *Logger) Warn(text string) *Message {
	return l.NewMessage(l.config.Warn(), text)
}

func (l *Logger) Warnf(format string, args ...interface{}) *Message {
	return l.NewMessagef(l.config.Warn(), format, args...)
}

func (l *Logger) Info(text string) *Message {
	return l.NewMessage(l.config.Info(), text)
}

func (l *Logger) Infof(format string, args ...interface{}) *Message {
	return l.NewMessagef(l.config.Info(), format, args...)
}

func (l *Logger) Debug(text string) *Message {
	return l.NewMessage(l.config.Debug(), text)
}

func (l *Logger) Debugf(format string, args ...interface{}) *Message {
	return l.NewMessagef(l.config.Debug(), format, args...)
}

func (l *Logger) Trace(text string) *Message {
	return l.NewMessage(l.config.Trace(), text)
}

func (l *Logger) Tracef(format string, args ...interface{}) *Message {
	return l.NewMessagef(l.config.Trace(), format, args...)
}

func (l *Logger) NewLevelWriter(level Level) *LevelWriter {
	return &LevelWriter{logger: l, level: level}
}

func (l *Logger) FatalWriter() *LevelWriter {
	return l.NewLevelWriter(l.config.Fatal())
}

func (l *Logger) ErrorWriter() *LevelWriter {
	return l.NewLevelWriter(l.config.Error())
}

func (l *Logger) WarnWriter() *LevelWriter {
	return l.NewLevelWriter(l.config.Warn())
}

func (l *Logger) InfoWriter() *LevelWriter {
	return l.NewLevelWriter(l.config.Info())
}

func (l *Logger) DebugWriter() *LevelWriter {
	return l.NewLevelWriter(l.config.Debug())
}

func (l *Logger) TraceWriter() *LevelWriter {
	return l.NewLevelWriter(l.config.Trace())
}

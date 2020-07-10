package golog

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Logger struct {
	config Config
	prefix string
	values Values
	mtx    sync.Mutex
}

// NewLogger returns a Logger with the given config and per message values.
// If config is nil, then a nil Logger will be returned.
// A nil Logger is still valid to use but will not log anything.
// The passed perMessageValues will be repeated for every new log message.
func NewLogger(config Config, perMessageValues ...Value) *Logger {
	if config == nil {
		return nil
	}
	return &Logger{
		config: config,
		values: perMessageValues,
	}
}

// NewLogger returns a Logger with the given config, prefix, and per message values.
// If config is nil, then a nil Logger will be returned.
// A nil Logger is still valid to use but will not log anything.
// Every log message will begin with the passed prefix.
// The passed perMessageValues will be repeated for every new log message.
func NewLoggerWithPrefix(config Config, prefix string, perMessageValues ...Value) *Logger {
	if config == nil {
		return nil
	}
	return &Logger{
		config: config,
		prefix: prefix,
		values: perMessageValues,
	}
}

// WithCtx returns a new sub Logger with the Values from
// the context add to if as per message values.
// Returns l unchanged, ther there were no Values added to the context.
func (l *Logger) WithCtx(ctx context.Context) *Logger {
	return l.WithValues(ValuesFromContext(ctx)...)
}

// With returns a new Message that can be used to record
// the prefix for a sub-logger.
//
// Example:
//   log := log.With().UUID("requestID", requestID).SubLogger()
func (l *Logger) With() *Message {
	if l == nil {
		return nil
	}
	return newMessageFromPool(l, NewValueRecorder(), "")
}

// WithLevelFilter returns a clone of the logger using
// the passed filter or returns nil if the logger was nil.
func (l *Logger) WithLevelFilter(filter LevelFilter) *Logger {
	if l == nil {
		return nil
	}
	return &Logger{
		config: NewDerivedConfig(&l.config, filter),
		prefix: l.prefix,
		values: l.values,
	}
}

// Config returns the configuration of the logger
// or nil if the logger is nil.
func (l *Logger) Config() Config {
	if l == nil {
		return nil
	}
	return l.config
}

// Values returns the values that will be repeated
// for every message of the logger.
// See Logger.WithValues
func (l *Logger) Values() Values {
	if l == nil {
		return nil
	}
	return l.values
}

// WithValues returns a new Logger with the passed
// perMessageValues appended to the existing perMessageValues.
// See Logger.Values
func (l *Logger) WithValues(perMessageValues ...Value) *Logger {
	if l == nil || len(perMessageValues) == 0 {
		return l
	}
	return &Logger{
		config: l.config,
		prefix: l.prefix,
		values: MergeValues(l.values, perMessageValues),
	}
}

// Prefix returns the prefix string that will be
// added in front over every log message of the logger.
// See Logger.WithPrefix
func (l *Logger) Prefix() string {
	if l == nil {
		return ""
	}
	return l.prefix
}

// WithPrefix returns a clone of the logger using
// the passed prefix or returns nil if the logger was nil.
// See Logger.Prefix
func (l *Logger) WithPrefix(prefix string) *Logger {
	if l == nil {
		return nil
	}
	return &Logger{
		config: l.config,
		values: l.values,
		prefix: prefix,
	}
}

// IsActive returns if the passed level is active at the logger
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
	m := newMessageFromPool(l, l.config.Formatter().Clone(level), text)
	m.formatter.WriteText(t, l.config.Levels(), level, l.prefix, text)
	for _, namedValue := range l.values {
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

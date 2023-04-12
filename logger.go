package golog

import (
	"context"
	"fmt"
	"time"
)

type Logger struct {
	config  Config
	prefix  string
	attribs Attribs
}

// NewLogger returns a Logger with the given config and per message attributes.
// If config is nil, then a nil Logger will be returned.
// A nil Logger is still valid to use but will not log anything.
// The passed perMessageAttribs will be repeated for every new log message.
func NewLogger(config Config, perMessageAttribs ...Attrib) *Logger {
	if config == nil {
		return nil
	}
	return &Logger{
		config:  config,
		attribs: perMessageAttribs,
	}
}

// NewLogger returns a Logger with the given config, prefix, and per message attributes.
// If config is nil, then a nil Logger will be returned.
// A nil Logger is still valid to use but will not log anything.
// Every log message will begin with the passed prefix.
// The passed perMessageAttribs will be repeated for every new log message.
func NewLoggerWithPrefix(config Config, prefix string, perMessageAttribs ...Attrib) *Logger {
	if config == nil {
		return nil
	}
	return &Logger{
		config:  config,
		prefix:  prefix,
		attribs: perMessageAttribs,
	}
}

// WithCtx returns a new sub Logger with the Attribs from
// the context add to if as per message values.
// Returns the logger unchanged if there were no Attribs added to the context.
func (l *Logger) WithCtx(ctx context.Context) *Logger {
	return l.WithAttribs(AttribsFromContext(ctx)...)
}

// With returns a new Message that can be used to record
// the prefix for a sub-logger.
//
// Example:
//
//	log := log.With().UUID("requestID", requestID).SubLogger()
func (l *Logger) With() *Message {
	if l == nil {
		return nil
	}
	return newMessageFromPool(l, NewAttribsRecorder(), LevelInvalid, "")
}

// WithLevelFilter returns a clone of the logger using
// the passed filter or returns nil if the logger was nil.
func (l *Logger) WithLevelFilter(filter LevelFilter) *Logger {
	if l == nil {
		return nil
	}
	return &Logger{
		config:  NewDerivedConfig(&l.config, filter),
		prefix:  l.prefix,
		attribs: l.attribs,
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

// Attribs returns the attributes that will be repeated
// for every message of the logger.
// See Logger.WithAttribs
func (l *Logger) Attribs() Attribs {
	if l == nil {
		return nil
	}
	return l.attribs
}

// WithAttribs returns a new Logger with the passed
// perMessageAttribs merged with the existing perMessageAttribs.
// See Logger.Attribs and MergeAttribs
func (l *Logger) WithAttribs(perMessageAttribs ...Attrib) *Logger {
	if l == nil || len(perMessageAttribs) == 0 {
		return l
	}
	return &Logger{
		config:  l.config,
		prefix:  l.prefix,
		attribs: MergeAttribs(l.attribs, perMessageAttribs),
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
		config:  l.config,
		attribs: l.attribs,
		prefix:  prefix,
	}
}

// IsActive returns if the passed level is active at the logger
func (l *Logger) IsActive(level Level) bool {
	if l == nil {
		return false
	}
	return l.config.IsActive(level)
}

// Flush unwritten logs
func (l *Logger) Flush() {
	if l == nil {
		return
	}
	l.config.Writer().FlushUnderlying()
}

func (l *Logger) NewMessageAt(t time.Time, level Level, text string) *Message {
	if !l.IsActive(level) {
		return nil
	}
	m := newMessageFromPool(l, l.config.Writer().BeginMessage(l, t, level, l.prefix, text), level, text)
	if attribs := m.logger.attribs; len(attribs) > 0 {
		loggerWithoutAttribs := Logger{
			config: l.config,
			prefix: l.prefix,
		}
		// Temporarely set the message logger to a copy without attribs
		// because logging the attribs will be prevented
		// if the logger already has attribs with the same keys
		m.logger = &loggerWithoutAttribs
		attribs.Log(m)
		m.logger = l // Restore logger with attribs
	}
	return m
}

func (l *Logger) NewMessage(level Level, text string) *Message {
	if !l.IsActive(level) {
		return nil
	}
	return l.NewMessageAt(time.Now(), level, text)
}

func (l *Logger) NewMessagef(level Level, format string, args ...any) *Message {
	if !l.IsActive(level) {
		return nil
	}
	return l.NewMessageAt(time.Now(), level, fmt.Sprintf(format, args...))
}

// FatalAndPanic is a shortcut for Fatal(fmt.Sprint(p)).LogAndPanic()
func (l *Logger) FatalAndPanic(p any) {
	l.Fatal(fmt.Sprint(p)).Log()
	panic(p)
}

func (l *Logger) Fatal(text string) *Message {
	return l.NewMessage(l.config.Fatal(), text)
}

func (l *Logger) Fatalf(format string, args ...any) *Message {
	return l.NewMessagef(l.config.Fatal(), format, args...)
}

func (l *Logger) Error(text string) *Message {
	return l.NewMessage(l.config.Error(), text)
}

// Errorf uses fmt.Errorf underneath to support Go 1.13 wrapped error formatting with %w
func (l *Logger) Errorf(format string, args ...any) *Message {
	return l.NewMessage(l.config.Error(), fmt.Errorf(format, args...).Error())
}

func (l *Logger) Warn(text string) *Message {
	return l.NewMessage(l.config.Warn(), text)
}

func (l *Logger) Warnf(format string, args ...any) *Message {
	return l.NewMessagef(l.config.Warn(), format, args...)
}

func (l *Logger) Info(text string) *Message {
	return l.NewMessage(l.config.Info(), text)
}

func (l *Logger) Infof(format string, args ...any) *Message {
	return l.NewMessagef(l.config.Info(), format, args...)
}

func (l *Logger) Debug(text string) *Message {
	return l.NewMessage(l.config.Debug(), text)
}

func (l *Logger) Debugf(format string, args ...any) *Message {
	return l.NewMessagef(l.config.Debug(), format, args...)
}

func (l *Logger) Trace(text string) *Message {
	return l.NewMessage(l.config.Trace(), text)
}

func (l *Logger) Tracef(format string, args ...any) *Message {
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

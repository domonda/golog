package golog

import (
	"context"
	"fmt"
	"time"
)

// Logger starts new log messages.
// A nil Logger is valid to use but will not log anything.
type Logger struct {
	config  Config  // Can be shared between loggers
	prefix  string  // Prefix for every log message
	attribs Attribs // Attributes that will be repeated for every message
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
// the per message attribs for a sub-logger.
//
// Example:
//
//	log := log.With().UUID("requestID", requestID).SubLogger()
func (l *Logger) With() *Message {
	if l == nil {
		return nil
	}
	// Using nil as writer will make the message
	// record attribs instead of writing them.
	// Message.SubLogger() will then create a new
	// logger with the recorded attribs.
	return messageFromPool(l, l.attribs, nil, LevelInvalid, "")
}

// WithLevelFilter returns a clone of the logger using
// the passed filter or returns nil if the logger was nil.
func (l *Logger) WithLevelFilter(filter LevelFilter) *Logger {
	if l == nil {
		return nil
	}
	return &Logger{
		config:  NewDerivedConfigWithFilter(&l.config, filter),
		prefix:  l.prefix,
		attribs: l.attribs,
	}
}

func (l *Logger) WithAdditionalWriterConfigs(configs ...WriterConfig) *Logger {
	if l == nil || len(configs) == 0 {
		return l
	}
	return &Logger{
		config:  NewDerivedConfigWithAdditionalWriterConfigs(&l.config, configs...),
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
		attribs: l.attribs.AppendUnique(perMessageAttribs...),
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
func (l *Logger) IsActive(ctx context.Context, level Level) bool {
	return l != nil && IsActiveContext(ctx, level) && l.config.IsActive(ctx, level)
}

// Flush unwritten logs
func (l *Logger) Flush() {
	if l == nil {
		return
	}
	for _, w := range l.config.WriterConfigs() {
		w.FlushUnderlying()
	}
}

// NewMessageAt starts a new message logged with the given timestamp
func (l *Logger) NewMessageAt(ctx context.Context, timestamp time.Time, level Level, text string) *Message {
	// Logging should always err on the side of robustness
	// so accept nil to prevent panics.
	if ctx == nil {
		ctx = context.Background()
	}
	if !l.IsActive(ctx, level) {
		return nil
	}
	configs := l.config.WriterConfigs()
	if c := WriterConfigsFromContext(ctx); len(c) > 0 {
		configs = uniqueWriterConfigs(append(configs, c...))
	}
	writers, _ := writersPool.Get().([]Writer) // Empty slices but with capacity
	if writers == nil {
		writers = make([]Writer, 0, len(configs))
	}
	for _, config := range configs {
		if w := config.WriterForNewMessage(ctx, level); w != nil {
			w.BeginMessage(l.config, timestamp, level, l.prefix, text)
			writers = append(writers, w)
		}
	}
	// Get new Message without attribs so that the logger
	// attribs can get logged without detecting that
	// attribs with their keys are already present
	msg := messageFromPool(l, nil, writers, level, text)
	if len(l.attribs) > 0 {
		for _, attrib := range l.attribs {
			attrib.Log(msg)
		}
		// After logger attribs have been written
		// set them at the message to prevent
		// writing more attribs with the same keys
		msg.attribs = l.attribs
	}
	// Context attribs are logged after logger attribs
	// meaning they are ignored if attribs
	// with the same key were already logged
	AttribsFromContext(ctx).Log(msg)
	// After the attribs from the logger
	// and the context have been logged,
	// further attribs can be logged using
	// the methods of the returned Message.
	return msg
}

// NewMessage starts a new message
func (l *Logger) NewMessage(ctx context.Context, level Level, text string) *Message {
	return l.NewMessageAt(ctx, Timestamp(ctx), level, text)
}

// NewMessagef starts a new message formatted using fmt.Sprintf
func (l *Logger) NewMessagef(ctx context.Context, level Level, format string, args ...any) *Message {
	return l.NewMessageAt(ctx, Timestamp(ctx), level, fmt.Sprintf(format, args...))
}

// FatalAndPanic is a shortcut for Fatal(fmt.Sprint(p)).LogAndPanic()
func (l *Logger) FatalAndPanic(p any) {
	l.Fatal(fmt.Sprint(p)).Log()
	panic(p)
}

func (l *Logger) Fatal(text string) *Message {
	return l.NewMessage(context.Background(), l.config.FatalLevel(), text)
}

func (l *Logger) FatalCtx(ctx context.Context, text string) *Message {
	return l.NewMessage(ctx, l.config.FatalLevel(), text)
}

func (l *Logger) Fatalf(format string, args ...any) *Message {
	return l.NewMessagef(context.Background(), l.config.FatalLevel(), format, args...)
}

func (l *Logger) FatalfCtx(ctx context.Context, format string, args ...any) *Message {
	return l.NewMessagef(ctx, l.config.FatalLevel(), format, args...)
}

func (l *Logger) Error(text string) *Message {
	return l.NewMessage(context.Background(), l.config.ErrorLevel(), text)
}

func (l *Logger) ErrorAt(timestamp time.Time, text string) *Message {
	return l.NewMessageAt(context.Background(), timestamp, l.config.ErrorLevel(), text)
}

func (l *Logger) ErrorCtx(ctx context.Context, text string) *Message {
	return l.NewMessage(ctx, l.config.ErrorLevel(), text)
}

// Errorf uses fmt.Errorf underneath to support Go 1.13 wrapped error formatting with %w
func (l *Logger) Errorf(format string, args ...any) *Message {
	return l.NewMessage(context.Background(), l.config.ErrorLevel(), fmt.Errorf(format, args...).Error())
}

// ErrorfCtx uses fmt.Errorf underneath to support Go 1.13 wrapped error formatting with %w
func (l *Logger) ErrorfCtx(ctx context.Context, format string, args ...any) *Message {
	return l.NewMessage(ctx, l.config.ErrorLevel(), fmt.Errorf(format, args...).Error())
}

func (l *Logger) Warn(text string) *Message {
	return l.NewMessage(context.Background(), l.config.WarnLevel(), text)
}

func (l *Logger) WarnAt(timestamp time.Time, text string) *Message {
	return l.NewMessageAt(context.Background(), timestamp, l.config.WarnLevel(), text)
}

func (l *Logger) WarnCtx(ctx context.Context, text string) *Message {
	return l.NewMessage(ctx, l.config.WarnLevel(), text)
}

func (l *Logger) Warnf(format string, args ...any) *Message {
	return l.NewMessagef(context.Background(), l.config.WarnLevel(), format, args...)
}

func (l *Logger) WarnfCtx(ctx context.Context, format string, args ...any) *Message {
	return l.NewMessagef(ctx, l.config.WarnLevel(), format, args...)
}

func (l *Logger) Info(text string) *Message {
	return l.NewMessage(context.Background(), l.config.InfoLevel(), text)
}

func (l *Logger) InfoAt(timestamp time.Time, text string) *Message {
	return l.NewMessageAt(context.Background(), timestamp, l.config.InfoLevel(), text)
}

func (l *Logger) InfoCtx(ctx context.Context, text string) *Message {
	return l.NewMessage(ctx, l.config.InfoLevel(), text)
}

func (l *Logger) Infof(format string, args ...any) *Message {
	return l.NewMessagef(context.Background(), l.config.InfoLevel(), format, args...)
}

func (l *Logger) InfofCtx(ctx context.Context, format string, args ...any) *Message {
	return l.NewMessagef(ctx, l.config.InfoLevel(), format, args...)
}

func (l *Logger) Debug(text string) *Message {
	return l.NewMessage(context.Background(), l.config.DebugLevel(), text)
}

func (l *Logger) DebugAt(timestamp time.Time, text string) *Message {
	return l.NewMessageAt(context.Background(), timestamp, l.config.DebugLevel(), text)
}

func (l *Logger) DebugCtx(ctx context.Context, text string) *Message {
	return l.NewMessage(ctx, l.config.DebugLevel(), text)
}

func (l *Logger) Debugf(format string, args ...any) *Message {
	return l.NewMessagef(context.Background(), l.config.DebugLevel(), format, args...)
}

func (l *Logger) DebugfCtx(ctx context.Context, format string, args ...any) *Message {
	return l.NewMessagef(ctx, l.config.DebugLevel(), format, args...)
}

func (l *Logger) Trace(text string) *Message {
	return l.NewMessage(context.Background(), l.config.TraceLevel(), text)
}

func (l *Logger) TraceAt(timestamp time.Time, text string) *Message {
	return l.NewMessageAt(context.Background(), timestamp, l.config.TraceLevel(), text)
}

func (l *Logger) TraceCtx(ctx context.Context, text string) *Message {
	return l.NewMessage(ctx, l.config.TraceLevel(), text)
}

func (l *Logger) Tracef(format string, args ...any) *Message {
	return l.NewMessagef(context.Background(), l.config.TraceLevel(), format, args...)
}

func (l *Logger) TracefCtx(ctx context.Context, format string, args ...any) *Message {
	return l.NewMessagef(ctx, l.config.TraceLevel(), format, args...)
}

func (l *Logger) NewLevelWriter(level Level) *LevelWriter {
	return &LevelWriter{logger: l, level: level}
}

func (l *Logger) FatalWriter() *LevelWriter {
	return l.NewLevelWriter(l.config.FatalLevel())
}

func (l *Logger) ErrorWriter() *LevelWriter {
	return l.NewLevelWriter(l.config.ErrorLevel())
}

func (l *Logger) WarnWriter() *LevelWriter {
	return l.NewLevelWriter(l.config.WarnLevel())
}

func (l *Logger) InfoWriter() *LevelWriter {
	return l.NewLevelWriter(l.config.InfoLevel())
}

func (l *Logger) DebugWriter() *LevelWriter {
	return l.NewLevelWriter(l.config.DebugLevel())
}

func (l *Logger) TraceWriter() *LevelWriter {
	return l.NewLevelWriter(l.config.TraceLevel())
}

package golog

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Logger struct {
	config Config
	hooks  []Hook
	mtx    sync.Mutex
}

func NewLogger(config Config, hooks ...Hook) *Logger {
	if config == nil {
		return nil
	}
	return &Logger{
		config: config,
		hooks:  hooks,
	}
}

func ContextLogger(ctx context.Context) *Logger {
	l, _ := ctx.Value(ctxKey{}).(*Logger)
	return l
}

type ctxKey struct{}

func (l *Logger) Context(ctx context.Context) context.Context {
	if l == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, l)
}

func (l *Logger) Config() Config {
	return l.config
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

func (l *Logger) WithHooks(hooks ...Hook) *Logger {
	if l == nil {
		return nil
	}
	return &Logger{
		config: l.config,
		hooks:  append(l.hooks, hooks...),
	}
}

func (l *Logger) WithLevelFilter(filter LevelFilter) *Logger {
	if l == nil {
		return nil
	}
	return &Logger{
		config: NewDerivedConfig(&l.config, filter),
		hooks:  l.hooks,
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

func (l *Logger) NewMessageAt(t time.Time, level Level, text string) *Message {
	if !l.IsActive(level) {
		return nil
	}
	m := newMessage(l, l.config.Formatter().Clone(), text)
	m.formatter.WriteText(t, l.config.Levels(), level, text)
	for _, hook := range l.hooks {
		hook.Log(m)
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

func (l *Logger) Fatal(text string) *Message {
	return l.NewMessage(l.config.Fatal(), text)
}

func (l *Logger) Fatalf(format string, args ...interface{}) *Message {
	return l.NewMessagef(l.config.Fatal(), format, args...)
}

func (l *Logger) Error(text string) *Message {
	return l.NewMessage(l.config.Error(), text)
}

func (l *Logger) Errorf(format string, args ...interface{}) *Message {
	return l.NewMessagef(l.config.Error(), format, args...)
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

func (l *Logger) LogFatal(text string) {
	l.NewMessage(l.config.Fatal(), text).Log()
}

func (l *Logger) LogFatalf(format string, args ...interface{}) {
	l.NewMessagef(l.config.Fatal(), format, args...).Log()
}

func (l *Logger) LogError(text string) {
	l.NewMessage(l.config.Error(), text).Log()
}

func (l *Logger) LogErrorf(format string, args ...interface{}) {
	l.NewMessagef(l.config.Error(), format, args...).Log()
}

func (l *Logger) LogWarn(text string) {
	l.NewMessage(l.config.Warn(), text).Log()
}

func (l *Logger) LogWarnf(format string, args ...interface{}) {
	l.NewMessagef(l.config.Warn(), format, args...).Log()
}

func (l *Logger) LogInfo(text string) {
	l.NewMessage(l.config.Info(), text).Log()
}

func (l *Logger) LogInfof(format string, args ...interface{}) {
	l.NewMessagef(l.config.Info(), format, args...).Log()
}

func (l *Logger) LogDebug(text string) {
	l.NewMessage(l.config.Debug(), text).Log()
}

func (l *Logger) LogDebugf(format string, args ...interface{}) {
	l.NewMessagef(l.config.Debug(), format, args...).Log()
}

func (l *Logger) LogTrace(text string) {
	l.NewMessage(l.config.Trace(), text).Log()
}

func (l *Logger) LogTracef(format string, args ...interface{}) {
	l.NewMessagef(l.config.Trace(), format, args...).Log()
}

func (l *Logger) LogFatalAndExit(text string) {
	l.NewMessage(l.config.Fatal(), text).LogAndExit()
}

func (l *Logger) LogFatalfAndExit(format string, args ...interface{}) {
	l.NewMessagef(l.config.Fatal(), format, args...).LogAndExit()
}

func (l *Logger) LogErrorAndExit(text string) {
	l.NewMessage(l.config.Error(), text).LogAndExit()
}

func (l *Logger) LogErrorfAndExit(format string, args ...interface{}) {
	l.NewMessagef(l.config.Error(), format, args...).LogAndExit()
}

func (l *Logger) LogWarnAndExit(text string) {
	l.NewMessage(l.config.Warn(), text).LogAndExit()
}

func (l *Logger) LogWarnfAndExit(format string, args ...interface{}) {
	l.NewMessagef(l.config.Warn(), format, args...).LogAndExit()
}

func (l *Logger) NewLevelWriter(level Level, exit bool) *LevelWriter {
	return &LevelWriter{logger: l, level: level, exit: exit}
}

func (l *Logger) FatalWriter() *LevelWriter {
	return l.NewLevelWriter(l.config.Fatal(), true)
}

func (l *Logger) ErrorWriter() *LevelWriter {
	return l.NewLevelWriter(l.config.Error(), true)
}

func (l *Logger) WarnWriter() *LevelWriter {
	return l.NewLevelWriter(l.config.Warn(), true)
}

func (l *Logger) InfoWriter() *LevelWriter {
	return l.NewLevelWriter(l.config.Info(), true)
}

func (l *Logger) DebugWriter() *LevelWriter {
	return l.NewLevelWriter(l.config.Debug(), true)
}

func (l *Logger) TraceWriter() *LevelWriter {
	return l.NewLevelWriter(l.config.Trace(), true)
}

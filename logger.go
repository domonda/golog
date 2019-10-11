package golog

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Logger struct {
	levels      *Levels
	levelFilter LevelFilter
	formatter   Formatter
	hooks       []Hook
	mtx         sync.Mutex
}

func NewLogger(levels *Levels, levelFilter LevelFilter, formatters ...Formatter) *Logger {
	l := &Logger{
		levels:      levels,
		levelFilter: levelFilter,
	}
	switch len(formatters) {
	case 0:
		// return nil ?
	case 1:
		l.formatter = formatters[0]
	default:
		l.formatter = MultiFormatter(formatters)
	}
	return l
}

func (l *Logger) CloneWithHooks(hooks ...Hook) *Logger {
	if l == nil {
		return nil
	}
	return &Logger{
		levels:      l.levels,
		levelFilter: l.levelFilter,
		formatter:   l.formatter,
		hooks:       append(l.hooks, hooks...),
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

func (l *Logger) AddHook(hook Hook) {
	l.mtx.Lock()
	l.hooks = append(l.hooks, hook)
	l.mtx.Unlock()
}

func (l *Logger) SetHooks(hooks ...Hook) {
	l.mtx.Lock()
	l.hooks = hooks
	l.mtx.Unlock()
}

func (l *Logger) SetLevelFilter(filter LevelFilter) {
	l.mtx.Lock()
	l.levelFilter = filter
	l.mtx.Unlock()
}

func (l *Logger) GetLevels() *Levels {
	return l.levels
}

func (l *Logger) GetLevelFatal() Level {
	return l.levels.Fatal
}

func (l *Logger) GetLevelError() Level {
	return l.levels.Error
}

func (l *Logger) GetLevelWarn() Level {
	return l.levels.Warn
}

func (l *Logger) GetLevelInfo() Level {
	return l.levels.Info
}

func (l *Logger) GetLevelDebug() Level {
	return l.levels.Debug
}

func (l *Logger) GetLevelTrace() Level {
	return l.levels.Trace
}

func (l *Logger) GetLevelFilter() LevelFilter {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	return l.levelFilter
}

func (l *Logger) IsActive(level Level) bool {
	if l == nil {
		return false
	}
	l.mtx.Lock()
	active := l.levelFilter.IsActive(level)
	l.mtx.Unlock()
	return active
}

// Record returns a new Message that can be used to record
// the prefix for a sub-logger.
//
// Example:
//   log := log.Record().Str("requestID", requestID).NewLogger()
func (l *Logger) Record() *Message {
	if l == nil {
		return nil
	}
	return newMessage(l, new(recordingFormatter))
}

func (l *Logger) NewMessageAt(t time.Time, level Level, msg string) *Message {
	if !l.IsActive(level) {
		return nil
	}
	m := newMessage(l, l.formatter.Clone())
	m.formatter.WriteMsg(t, l.levels, level, msg)
	for _, hook := range l.hooks {
		hook.Log(m)
	}
	return m
}

func (l *Logger) NewMessage(level Level, msg string) *Message {
	if !l.IsActive(level) {
		return nil
	}
	return l.NewMessageAt(time.Now(), level, msg)
}

func (l *Logger) NewMessagef(level Level, format string, args ...interface{}) *Message {
	if !l.IsActive(level) {
		return nil
	}
	return l.NewMessageAt(time.Now(), level, fmt.Sprintf(format, args...))
}

func (l *Logger) Fatal(msg string) *Message {
	return l.NewMessage(l.levels.Fatal, msg)
}

func (l *Logger) Fatalf(format string, args ...interface{}) *Message {
	return l.NewMessagef(l.levels.Fatal, format, args...)
}

func (l *Logger) Error(msg string) *Message {
	return l.NewMessage(l.levels.Error, msg)
}

func (l *Logger) Errorf(format string, args ...interface{}) *Message {
	return l.NewMessagef(l.levels.Error, format, args...)
}

func (l *Logger) Warn(msg string) *Message {
	return l.NewMessage(l.levels.Warn, msg)
}

func (l *Logger) Warnf(format string, args ...interface{}) *Message {
	return l.NewMessagef(l.levels.Warn, format, args...)
}

func (l *Logger) Info(msg string) *Message {
	return l.NewMessage(l.levels.Info, msg)
}

func (l *Logger) Infof(format string, args ...interface{}) *Message {
	return l.NewMessagef(l.levels.Info, format, args...)
}

func (l *Logger) Debug(msg string) *Message {
	return l.NewMessage(l.levels.Debug, msg)
}

func (l *Logger) Debugf(format string, args ...interface{}) *Message {
	return l.NewMessagef(l.levels.Debug, format, args...)
}

func (l *Logger) Trace(msg string) *Message {
	return l.NewMessage(l.levels.Trace, msg)
}

func (l *Logger) Tracef(format string, args ...interface{}) *Message {
	return l.NewMessagef(l.levels.Trace, format, args...)
}

func (l *Logger) LogFatal(msg string) {
	l.NewMessage(l.levels.Fatal, msg).Log()
}

func (l *Logger) LogFatalf(format string, args ...interface{}) {
	l.NewMessagef(l.levels.Fatal, format, args...).Log()
}

func (l *Logger) LogError(msg string) {
	l.NewMessage(l.levels.Error, msg).Log()
}

func (l *Logger) LogErrorf(format string, args ...interface{}) {
	l.NewMessagef(l.levels.Error, format, args...).Log()
}

func (l *Logger) LogWarn(msg string) {
	l.NewMessage(l.levels.Warn, msg).Log()
}

func (l *Logger) LogWarnf(format string, args ...interface{}) {
	l.NewMessagef(l.levels.Warn, format, args...).Log()
}

func (l *Logger) LogInfo(msg string) {
	l.NewMessage(l.levels.Info, msg).Log()
}

func (l *Logger) LogInfof(format string, args ...interface{}) {
	l.NewMessagef(l.levels.Info, format, args...).Log()
}

func (l *Logger) LogDebug(msg string) {
	l.NewMessage(l.levels.Debug, msg).Log()
}

func (l *Logger) LogDebugf(format string, args ...interface{}) {
	l.NewMessagef(l.levels.Debug, format, args...).Log()
}

func (l *Logger) LogTrace(msg string) {
	l.NewMessage(l.levels.Trace, msg).Log()
}

func (l *Logger) LogTracef(format string, args ...interface{}) {
	l.NewMessagef(l.levels.Trace, format, args...).Log()
}

func (l *Logger) LogFatalAndExit(msg string) {
	l.NewMessage(l.levels.Fatal, msg).LogAndExit()
}

func (l *Logger) LogFatalfAndExit(format string, args ...interface{}) {
	l.NewMessagef(l.levels.Fatal, format, args...).LogAndExit()
}

func (l *Logger) LogErrorAndExit(msg string) {
	l.NewMessage(l.levels.Error, msg).LogAndExit()
}

func (l *Logger) LogErrorfAndExit(format string, args ...interface{}) {
	l.NewMessagef(l.levels.Error, format, args...).LogAndExit()
}

func (l *Logger) LogWarnAndExit(msg string) {
	l.NewMessage(l.levels.Warn, msg).LogAndExit()
}

func (l *Logger) LogWarnfAndExit(format string, args ...interface{}) {
	l.NewMessagef(l.levels.Warn, format, args...).LogAndExit()
}

func (l *Logger) NewLevelWriter(level Level, exit bool) *LevelWriter {
	return &LevelWriter{logger: l, level: level, exit: exit}
}

func (l *Logger) FatalWriter() *LevelWriter {
	return l.NewLevelWriter(l.levels.Fatal, true)
}

func (l *Logger) ErrorWriter() *LevelWriter {
	return l.NewLevelWriter(l.levels.Error, true)
}

func (l *Logger) WarnWriter() *LevelWriter {
	return l.NewLevelWriter(l.levels.Warn, true)
}

func (l *Logger) InfoWriter() *LevelWriter {
	return l.NewLevelWriter(l.levels.Info, true)
}

func (l *Logger) DebugWriter() *LevelWriter {
	return l.NewLevelWriter(l.levels.Debug, true)
}

func (l *Logger) TraceWriter() *LevelWriter {
	return l.NewLevelWriter(l.levels.Trace, true)
}

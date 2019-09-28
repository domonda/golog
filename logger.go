package golog

import (
	"fmt"
	"io"
	"time"
)

type output struct {
	writer       io.Writer
	newFormatter NewFormatterFunc
	format       *Format
}

type Logger struct {
	levelFilter LevelFilter
	outputs     []output
	prefix      []*Message
}

func NewLogger(levelFilter LevelFilter, writer io.Writer, newFormatterFunc NewFormatterFunc, format *Format, prefix ...*Message) *Logger {
	return &Logger{
		levelFilter: levelFilter,
		outputs: []output{
			{
				writer:       writer,
				newFormatter: newFormatterFunc,
				format:       format,
			},
		},
		prefix: prefix,
	}
}

func (l *Logger) subLogger(prefixMessage *Message) *Logger {
	if l == nil {
		return nil
	}
	return &Logger{
		levelFilter: l.levelFilter,
		outputs:     l.outputs,
		prefix:      append(l.prefix, prefixMessage),
	}
}

// Prefix returns a new Message that can be used to record
// the prefix for a sub-logger.
//
// Example:
//   log := log.Prefix().Str("requestID", requestID).Logger()
func (l *Logger) Prefix() *Message {
	if l == nil {
		return nil
	}
	return NewMessage(l, l.newFormatter())
}

func (l *Logger) NewMessageAt(t time.Time, level Level, msg string) *Message {
	if l == nil || !l.levelFilter.IsActive(level) {
		return nil
	}

	formatter := l.newFormatter()
	formatter.WriteIntro(t, level, msg, nil)
	return NewMessage(l, formatter)
}

func (l *Logger) newFormatter() Formatter {
	if len(l.outputs) == 1 {
		return l.outputs[0].newFormatter(l.outputs[0].writer, l.outputs[0].format)
	}

	mf := make(MultiFormatter, len(l.outputs)) // todo optimize allocation
	for i := range l.outputs {
		mf[i] = l.outputs[i].newFormatter(l.outputs[i].writer, l.outputs[i].format)
	}
	return mf
}

func (l *Logger) NewMessage(level Level, msg string) *Message {
	if l == nil {
		return nil
	}
	return l.NewMessageAt(time.Now(), level, msg)
}

func (l *Logger) NewMessagef(level Level, format string, args ...interface{}) *Message {
	if l == nil || !l.levelFilter.IsActive(level) {
		return nil // Don't do fmt.Sprintf
	}
	return l.NewMessageAt(time.Now(), level, fmt.Sprintf(format, args...))
}

func (l *Logger) Fatal(msg string) *Message {
	return l.NewMessage(LevelFatal, msg)
}

func (l *Logger) Fatalf(format string, args ...interface{}) *Message {
	return l.NewMessagef(LevelFatal, format, args...)
}

func (l *Logger) Error(msg string) *Message {
	return l.NewMessage(LevelError, msg)
}

func (l *Logger) Errorf(format string, args ...interface{}) *Message {
	return l.NewMessagef(LevelError, format, args...)
}

func (l *Logger) Warn(msg string) *Message {
	return l.NewMessage(LevelWarn, msg)
}

func (l *Logger) Warnf(format string, args ...interface{}) *Message {
	return l.NewMessagef(LevelWarn, format, args...)
}

func (l *Logger) Info(msg string) *Message {
	return l.NewMessage(LevelInfo, msg)
}

func (l *Logger) Infof(format string, args ...interface{}) *Message {
	return l.NewMessagef(LevelInfo, format, args...)
}

func (l *Logger) Debug(msg string) *Message {
	return l.NewMessage(LevelDebug, msg)
}

func (l *Logger) Debugf(format string, args ...interface{}) *Message {
	return l.NewMessagef(LevelDebug, format, args...)
}

func (l *Logger) Trace(msg string) *Message {
	return l.NewMessage(LevelTrace, msg)
}

func (l *Logger) Tracef(format string, args ...interface{}) *Message {
	return l.NewMessagef(LevelTrace, format, args...)
}

func (l *Logger) LogFatal(msg string) {
	l.NewMessage(LevelFatal, msg).Log()
}

func (l *Logger) LogFatalf(format string, args ...interface{}) {
	l.NewMessagef(LevelFatal, format, args...).Log()
}

func (l *Logger) LogError(msg string) {
	l.NewMessage(LevelError, msg).Log()
}

func (l *Logger) LogErrorf(format string, args ...interface{}) {
	l.NewMessagef(LevelError, format, args...).Log()
}

func (l *Logger) LogWarn(msg string) {
	l.NewMessage(LevelWarn, msg).Log()
}

func (l *Logger) LogWarnf(format string, args ...interface{}) {
	l.NewMessagef(LevelWarn, format, args...).Log()
}

func (l *Logger) LogInfo(msg string) {
	l.NewMessage(LevelInfo, msg).Log()
}

func (l *Logger) LogInfof(format string, args ...interface{}) {
	l.NewMessagef(LevelInfo, format, args...).Log()
}

func (l *Logger) LogDebug(msg string) {
	l.NewMessage(LevelDebug, msg).Log()
}

func (l *Logger) LogDebugf(format string, args ...interface{}) {
	l.NewMessagef(LevelDebug, format, args...).Log()
}

func (l *Logger) LogTrace(msg string) {
	l.NewMessage(LevelTrace, msg).Log()
}

func (l *Logger) LogTracef(format string, args ...interface{}) {
	l.NewMessagef(LevelTrace, format, args...).Log()
}

func (l *Logger) NewLevelWriter(level Level) *LevelWriter {
	return &LevelWriter{logger: l, level: level}
}

func (l *Logger) FatalWriter() *LevelWriter {
	return l.NewLevelWriter(LevelFatal)
}

func (l *Logger) ErrorWriter() *LevelWriter {
	return l.NewLevelWriter(LevelError)
}

func (l *Logger) WarnWriter() *LevelWriter {
	return l.NewLevelWriter(LevelWarn)
}

func (l *Logger) InfoWriter() *LevelWriter {
	return l.NewLevelWriter(LevelInfo)
}

func (l *Logger) DebugWriter() *LevelWriter {
	return l.NewLevelWriter(LevelDebug)
}

func (l *Logger) TraceWriter() *LevelWriter {
	return l.NewLevelWriter(LevelTrace)
}

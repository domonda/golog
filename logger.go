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
}

func NewLogger(levelFilter LevelFilter, writer io.Writer, newFormatter NewFormatterFunc, format *Format) *Logger {
	return &Logger{
		levelFilter: levelFilter,
		outputs:     []output{{writer, newFormatter, format}},
	}
}

func (l *Logger) Clone() *Logger {
	return &Logger{
		levelFilter: l.levelFilter,
	}
}

func (l *Logger) NewMessageAt(t time.Time, level Level, msg string) *Message {
	if !l.levelFilter.IsActive(level) {
		return nil
	}

	var f Formatter

	if len(l.outputs) == 1 {
		f = l.outputs[0].newFormatter(l.outputs[0].writer, l.outputs[0].format)
	} else {
		mf := make(MultiFormatter, len(l.outputs)) // todo optimize allocation
		for i := range l.outputs {
			mf[i] = l.outputs[i].newFormatter(l.outputs[i].writer, l.outputs[i].format)
		}
		f = mf
	}

	f.Begin(t, level, msg, nil)

	return NewMessage(l, f)
}

func (l *Logger) NewMessage(level Level, msg string) *Message {
	return l.NewMessageAt(time.Now(), level, msg)
}

func (l *Logger) Fatal(msg string) *Message {
	return l.NewMessage(LevelFatal, msg)
}

func (l *Logger) Fatalf(format string, args ...interface{}) *Message {
	return l.NewMessage(LevelFatal, fmt.Sprintf(format, args...))
}

func (l *Logger) Error(msg string) *Message {
	return l.NewMessage(LevelError, msg)
}

func (l *Logger) Errorf(format string, args ...interface{}) *Message {
	return l.NewMessage(LevelError, fmt.Sprintf(format, args...))
}

func (l *Logger) Warn(msg string) *Message {
	return l.NewMessage(LevelWarn, msg)
}

func (l *Logger) Warnf(format string, args ...interface{}) *Message {
	return l.NewMessage(LevelWarn, fmt.Sprintf(format, args...))
}

func (l *Logger) Info(msg string) *Message {
	return l.NewMessage(LevelInfo, msg)
}

func (l *Logger) Infof(format string, args ...interface{}) *Message {
	return l.NewMessage(LevelInfo, fmt.Sprintf(format, args...))
}

func (l *Logger) Debug(msg string) *Message {
	return l.NewMessage(LevelDebug, msg)
}

func (l *Logger) Debugf(format string, args ...interface{}) *Message {
	return l.NewMessage(LevelDebug, fmt.Sprintf(format, args...))
}

func (l *Logger) Trace(msg string) *Message {
	return l.NewMessage(LevelTrace, msg)
}

func (l *Logger) Tracef(format string, args ...interface{}) *Message {
	return l.NewMessage(LevelTrace, fmt.Sprintf(format, args...))
}

package golog

import (
	"fmt"
	"log"
	"strings"
)

// LevelWriter writes unstructured messages to a Logger with a fixed Level.
// It can be used as a shim/wrapper for third party packages
// that need a standard log.Logger, an io.Writer,
// or an an interface implementation with a Printf method.
type LevelWriter struct {
	logger *Logger
	level  Level
}

// Write implements io.Writer
func (w *LevelWriter) Write(data []byte) (int, error) {
	w.Msg(strings.TrimSuffix(string(data), "\n"))
	return len(data), nil
}

// Msg writes a string message.
func (w *LevelWriter) Msg(msg string) {
	if w.logger == nil {
		return
	}
	w.logger.NewMessage(w.level, msg).Log()
}

func (w *LevelWriter) Print(v ...interface{}) {
	w.Msg(fmt.Sprint(v...))
}

func (w *LevelWriter) Println(v ...interface{}) {
	msg := fmt.Sprintln(v...)
	w.Msg(msg[:len(msg)-1])
}

func (w *LevelWriter) Printf(format string, v ...interface{}) {
	if w.logger == nil {
		return
	}
	w.logger.NewMessagef(w.level, format, v...).Log()
}

// Func returns a function with the log.Printf call signature.
func (w *LevelWriter) Func() func(format string, v ...interface{}) {
	return func(format string, v ...interface{}) {
		w.Printf(format, v...)
	}
}

// StdLogger returns a new log.Logger that writes to the LevelWriter.
// See https://golang.org/pkg/log/
func (w *LevelWriter) StdLogger() *log.Logger {
	return log.New(w, "", 0)
}

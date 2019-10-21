package log

import (
	"context"
	"fmt"
	"os"

	"github.com/domonda/golog"
)

var (
	Levels = golog.DefaultLevels

	Format = golog.Format{
		TimestampKey:    "time",
		TimestampFormat: "2006-01-02 15:04:05.999",
		LevelKey:        "level",
		MessageKey:      "message",
	}

	Config = golog.NewConfig(
		&Levels,
		Levels.Debug.FilterOutBelow(),
		golog.NewTextFormatter(os.Stdout, &Format, golog.NoColorizer),
	)

	// Logger uses a golog.DerivedConfig referencing the
	// exported package variable Config.
	// This way Config can be changed after initialization of Logger
	// without the need to create and set a new golog.Logger.
	Logger = golog.NewLogger(golog.NewDerivedConfig(&Config))
)

func Context(ctx context.Context) context.Context {
	return Logger.Context(ctx)
}

func ContextLogger(ctx context.Context) *golog.Logger {
	return golog.ContextLogger(ctx)
}

func WithHooks(hooks ...golog.Hook) *golog.Logger {
	return Logger.WithHooks(hooks...)
}

func WithLevelFilter(filter golog.LevelFilter) *golog.Logger {
	return Logger.WithLevelFilter(filter)
}

func With() *golog.Message {
	return Logger.With()
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

func NewLevelWriter(level golog.Level, exit bool) *golog.LevelWriter {
	return Logger.NewLevelWriter(level, exit)
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

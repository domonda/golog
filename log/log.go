package log

import (
	"context"
	"fmt"
	"os"

	"github.com/fatih/color"

	"github.com/domonda/golog"
)

var (
	Levels = golog.DefaultLevels

	Format = golog.Format{
		TimestampKey:    "time",
		TimestampFormat: "2006-01-02 15:04:05.000",
		LevelKey:        "level",
		MessageKey:      "message",
	}

	Colorizer = golog.ConsoleColorizer{
		TimespampColor: color.New(color.FgHiBlack),

		OtherLevelColor: color.New(color.FgWhite),
		FatalLevelColor: color.New(color.FgHiRed),
		ErrorLevelColor: color.New(color.FgRed),
		WarnLevelColor:  color.New(color.FgYellow),
		InfoLevelColor:  color.New(color.FgCyan),
		DebugLevelColor: color.New(color.FgMagenta),
		TraceLevelColor: color.New(color.FgHiBlack),

		MsgColor:    color.New(color.FgHiWhite),
		KeyColor:    color.New(color.FgCyan),
		NilColor:    color.New(color.FgWhite),
		TrueColor:   color.New(color.FgGreen),
		FalseColor:  color.New(color.FgYellow),
		IntColor:    color.New(color.FgWhite),
		UintColor:   color.New(color.FgWhite),
		FloatColor:  color.New(color.FgWhite),
		UUIDColor:   color.New(color.FgWhite),
		StringColor: color.New(color.FgWhite),
		ErrorColor:  color.New(color.FgRed),
	}

	Config = golog.NewConfig(
		&Levels,
		Levels.LevelOfNameOrDefault(os.Getenv("LOG_LEVEL"), Levels.Debug).FilterOutBelow(),
		golog.NewTextFormatter(os.Stdout, &Format, &Colorizer),
	)

	// Logger uses a golog.DerivedConfig referencing the
	// exported package variable Config.
	// This way Config can be changed after initialization of Logger
	// without the need to create and set a new golog.Logger.
	Logger = golog.NewLogger(golog.NewDerivedConfig(&Config))

	Registry                     golog.Registry
	AddImportPathToPackageLogger = false
)

func NewPackageLogger(packageName string, filters ...golog.LevelFilter) *golog.Logger {
	config := golog.NewDerivedConfig(&Config, filters...)
	pkg := Registry.AddPackageConfig(config)
	logger := golog.NewLoggerWithPrefix(config, packageName+": ")
	if AddImportPathToPackageLogger {
		logger = logger.With().Str("pkg", pkg).NewLogger()
	}
	return logger
}

// Context returns a new context.Context with the default Logger.
// See ContextLogger
func Context(parent context.Context) context.Context {
	return Logger.Context(parent)
}

// ContextLogger returns a Logger if ctx has one
// or a nil Logger wich is still valid to use
// but does not produce any log output.
// See Context
func ContextLogger(ctx context.Context) *golog.Logger {
	return golog.ContextLogger(ctx)
}

func WithHooks(hooks ...golog.Hook) *golog.Logger {
	return Logger.WithHooks(hooks...)
}

func WithLevelFilter(filter golog.LevelFilter) *golog.Logger {
	return Logger.WithLevelFilter(filter)
}

func WithPrefix(prefix string) *golog.Logger {
	return Logger.WithPrefix(prefix)
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

func NewLevelWriter(level golog.Level) *golog.LevelWriter {
	return Logger.NewLevelWriter(level)
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

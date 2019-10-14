package log

import (
	"context"
	"os"

	"github.com/domonda/golog"
)

var (
	DefaultFormat = &golog.Format{
		TimestampKey:    "time",
		TimestampFormat: "2006-01-02 15:04:05.999",
		LevelKey:        "level",
		MessageKey:      "message",
	}

	DefaultConfig = golog.NewConfig(
		golog.DefaultLevels,
		golog.DefaultLevels.Debug.FilterAbove(),
		golog.NewTextFormatter(os.Stdout, DefaultFormat, golog.NoColorizer),
	)

	Config = golog.NewDerivedConfig(&DefaultConfig)

	Logger = golog.NewLogger(Config)
)

func Context(ctx context.Context) context.Context {
	return Logger.Context(ctx)
}

func ContextLogger(ctx context.Context) *golog.Logger {
	return golog.ContextLogger(ctx)
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

func LogFatal(text string) {
	Logger.LogFatal(text)
}

func LogFatalf(format string, args ...interface{}) {
	Logger.LogFatalf(format, args...)
}

func LogError(text string) {
	Logger.LogError(text)
}

func LogErrorf(format string, args ...interface{}) {
	Logger.LogErrorf(format, args...)
}

func LogWarn(text string) {
	Logger.LogWarn(text)
}

func LogWarnf(format string, args ...interface{}) {
	Logger.LogWarnf(format, args...)
}

func LogInfo(text string) {
	Logger.LogInfo(text)
}

func LogInfof(format string, args ...interface{}) {
	Logger.LogInfof(format, args...)
}

func LogDebug(text string) {
	Logger.LogDebug(text)
}

func LogDebugf(format string, args ...interface{}) {
	Logger.LogDebugf(format, args...)
}

func LogTrace(text string) {
	Logger.LogTrace(text)
}

func LogTracef(format string, args ...interface{}) {
	Logger.LogTracef(format, args...)
}

func LogFatalAndExit(text string) {
	Logger.LogFatalAndExit(text)
}

func LogFatalfAndExit(format string, args ...interface{}) {
	Logger.LogFatalfAndExit(format, args...)
}

func LogErrorAndExit(text string) {
	Logger.LogErrorAndExit(text)
}

func LogErrorfAndExit(format string, args ...interface{}) {
	Logger.LogErrorfAndExit(format, args...)
}

func LogWarnAndExit(text string) {
	Logger.LogWarnAndExit(text)
}

func LogWarnfAndExit(format string, args ...interface{}) {
	Logger.LogWarnfAndExit(format, args...)
}

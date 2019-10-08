package log

import (
	"context"
	"os"
	"sync"

	"github.com/domonda/golog"
)

var (
	Levels      = golog.DefaultLevels
	LevelFilter = Levels.Debug.FilterAbove()

	Format = &golog.Format{
		TimestampKey:    "time",
		TimestampFormat: "2006-01-02 15:04:05.999",
		LevelKey:        "level",
		MessageKey:      "message",
	}

	Formatter golog.Formatter = golog.NewTextFormatter(os.Stdout, Format, golog.NoColorizer)

	logger    *golog.Logger
	loggerMtx sync.Mutex
)

func GetLogger() *golog.Logger {
	loggerMtx.Lock()
	if logger == nil {
		logger = golog.NewLogger(Levels, LevelFilter, Formatter)
	}
	l := logger
	loggerMtx.Unlock()
	return l
}

func SetLogger(l *golog.Logger) {
	loggerMtx.Lock()
	logger = l
	loggerMtx.Unlock()
}

func Context(ctx context.Context) context.Context {
	return GetLogger().Context(ctx)
}

func ContextLogger(ctx context.Context) *golog.Logger {
	return golog.ContextLogger(ctx)
}

func Fatal(msg string) *golog.Message {
	return GetLogger().Fatal(msg)
}

func Fatalf(format string, args ...interface{}) *golog.Message {
	return GetLogger().Fatalf(format, args...)
}

func Error(msg string) *golog.Message {
	return GetLogger().Error(msg)
}

func Errorf(format string, args ...interface{}) *golog.Message {
	return GetLogger().Errorf(format, args...)
}

func Warn(msg string) *golog.Message {
	return GetLogger().Warn(msg)
}

func Warnf(format string, args ...interface{}) *golog.Message {
	return GetLogger().Warnf(format, args...)
}

func Info(msg string) *golog.Message {
	return GetLogger().Info(msg)
}

func Infof(format string, args ...interface{}) *golog.Message {
	return GetLogger().Infof(format, args...)
}

func Debug(msg string) *golog.Message {
	return GetLogger().Debug(msg)
}

func Debugf(format string, args ...interface{}) *golog.Message {
	return GetLogger().Debugf(format, args...)
}

func Trace(msg string) *golog.Message {
	return GetLogger().Trace(msg)
}

func Tracef(format string, args ...interface{}) *golog.Message {
	return GetLogger().Tracef(format, args...)
}

func LogFatal(msg string) {
	GetLogger().LogFatal(msg)
}

func LogFatalf(format string, args ...interface{}) {
	GetLogger().LogFatalf(format, args...)
}

func LogError(msg string) {
	GetLogger().LogError(msg)
}

func LogErrorf(format string, args ...interface{}) {
	GetLogger().LogErrorf(format, args...)
}

func LogWarn(msg string) {
	GetLogger().LogWarn(msg)
}

func LogWarnf(format string, args ...interface{}) {
	GetLogger().LogWarnf(format, args...)
}

func LogInfo(msg string) {
	GetLogger().LogInfo(msg)
}

func LogInfof(format string, args ...interface{}) {
	GetLogger().LogInfof(format, args...)
}

func LogDebug(msg string) {
	GetLogger().LogDebug(msg)
}

func LogDebugf(format string, args ...interface{}) {
	GetLogger().LogDebugf(format, args...)
}

func LogTrace(msg string) {
	GetLogger().LogTrace(msg)
}

func LogTracef(format string, args ...interface{}) {
	GetLogger().LogTracef(format, args...)
}

func LogFatalAndExit(msg string) {
	GetLogger().LogFatalAndExit(msg)
}

func LogFatalfAndExit(format string, args ...interface{}) {
	GetLogger().LogFatalfAndExit(format, args...)
}

func LogErrorAndExit(msg string) {
	GetLogger().LogErrorAndExit(msg)
}

func LogErrorfAndExit(format string, args ...interface{}) {
	GetLogger().LogErrorfAndExit(format, args...)
}

func LogWarnAndExit(msg string) {
	GetLogger().LogWarnAndExit(msg)
}

func LogWarnfAndExit(format string, args ...interface{}) {
	GetLogger().LogWarnfAndExit(format, args...)
}
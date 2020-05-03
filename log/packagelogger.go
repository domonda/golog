package log

import (
	"os"

	"github.com/domonda/golog"
)

var (
	// PackageRegistry holds package logger configurations.
	// See NewPackageLogger
	PackageRegistry golog.Registry

	// AddImportPathToPackageLogger controls weather the package import path
	// will be logged as value "pkg" with every package logger message.
	AddImportPathToPackageLogger = false
)

// NewPackageLogger creates a logger for a package
// where every log message will be prefixed with pkgName+": ".
// Note that pkgName is the name, not the import path of the package.
// It still has to be unique for all package loggers because
// the logger config is added to PackageRegistry by pkgName.
// The PackageRegistry can be used to change package logging
// configurations at runtime.
// If any filters are passed then they take precedence before the parent Config filter.
// But if an environment variable "LOG_LEVEL_PKG_" + pkgName is defined
// with a valid log level name, then this log level will be used as filter,
// instead of anything passed for filters.
// If AddImportPathToPackageLogger is true, then the package import path
// will be logged as value "pkg" with every message.
func NewPackageLogger(pkgName string, filters ...golog.LevelFilter) *golog.Logger {
	if pkgName == "" {
		panic("empty pkgName passed to NewPackageLogger")
	}

	if levelName := os.Getenv("LOG_LEVEL_PKG_" + pkgName); levelName != "" {
		if level := Levels.LevelOfName(levelName); level != golog.LevelInvalid {
			filters = []golog.LevelFilter{level.FilterOutBelow()}
		}
	}

	config := golog.NewDerivedConfig(&Config, filters...)
	pkgPath := PackageRegistry.AddPackageConfig(pkgName, config)
	logger := golog.NewLoggerWithPrefix(config, pkgName+": ")

	if AddImportPathToPackageLogger {
		logger = logger.With().Str("pkg", pkgPath).SubLogger()
	}
	return logger
}

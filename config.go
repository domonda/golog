package golog

import (
	"context"
)

// FilterHTTPHeaders holds names of HTTP headers
// that should not be logged for requests.
// Defaults are "Authorization" and "Cookie".
var FilterHTTPHeaders = map[string]struct{}{
	"Authorization": {},
	"Cookie":        {},
}

// GlobalPanicLevel causes any log message with that
// level or higher to panic the message without formatted values
// after the complete log message has been written including values.
// The default value LevelInvalid disables this behaviour.
// Useful to catch any otherwise ignored warning or error
// messages in automated tests. Don't use in production.
var GlobalPanicLevel Level = LevelInvalid

// Config implements LevelDecider
var _ LevelDecider = Config(nil)

// Config is an interface that gets implemented by actual Logger configurations.
//
// See also DerivedConfig.
type Config interface {
	WriterConfigs() []WriterConfig
	Levels() *Levels
	// IsActive implements the LevelDecider interface.
	// It's valid to pass a nil context.
	IsActive(ctx context.Context, level Level) bool
	FatalLevel() Level
	ErrorLevel() Level
	WarnLevel() Level
	InfoLevel() Level
	DebugLevel() Level
	TraceLevel() Level
}

func NewConfig(levels *Levels, filter LevelFilter, writers ...WriterConfig) Config {
	if levels == nil {
		panic("golog.Config needs Levels")
	}
	writers = uniqueWriterConfigs(writers)
	if len(writers) == 0 {
		panic("golog.Config needs a Writer")
	}
	return &config{
		levels:  levels,
		filter:  filter,
		writers: writers,
	}
}

type config struct {
	levels  *Levels
	filter  LevelFilter
	writers []WriterConfig
}

func (c *config) WriterConfigs() []WriterConfig {
	return c.writers
}

func (c *config) Levels() *Levels {
	return c.levels
}

func (c *config) IsActive(ctx context.Context, level Level) bool {
	return c.filter.IsActive(ctx, level)
}

func (c *config) FatalLevel() Level {
	return c.levels.Fatal
}

func (c *config) ErrorLevel() Level {
	return c.levels.Error
}

func (c *config) WarnLevel() Level {
	return c.levels.Warn
}

func (c *config) InfoLevel() Level {
	return c.levels.Info
}

func (c *config) DebugLevel() Level {
	return c.levels.Debug
}

func (c *config) TraceLevel() Level {
	return c.levels.Trace
}

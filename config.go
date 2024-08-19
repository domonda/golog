package golog

import "context"

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

type Config interface {
	Writer() Writer
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

func NewConfig(levels *Levels, filter LevelFilter, writers ...Writer) Config {
	switch len(writers) {
	case 0:
		panic("golog.Config needs a Writer")

	case 1:
		return &config{
			levels: levels,
			filter: filter,
			writer: writers[0],
		}

	default:
		return &config{
			levels: levels,
			filter: filter,
			writer: MultiWriter(writers),
		}
	}
}

type config struct {
	levels *Levels
	filter LevelFilter
	writer Writer
}

func (c *config) Writer() Writer {
	return c.writer
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

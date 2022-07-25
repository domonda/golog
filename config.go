package golog

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

type Config interface {
	Formatter() Formatter
	Levels() *Levels
	IsActive(level Level) bool
	Fatal() Level
	Error() Level
	Warn() Level
	Info() Level
	Debug() Level
	Trace() Level
}

func NewConfig(levels *Levels, filter LevelFilter, formatters ...Formatter) Config {
	switch len(formatters) {
	case 0:
		panic("golog.Config needs a Formatter")

	case 1:
		return &config{
			levels:    levels,
			filter:    filter,
			formatter: formatters[0],
		}

	default:
		return &config{
			levels:    levels,
			filter:    filter,
			formatter: MultiFormatter(formatters),
		}
	}
}

type config struct {
	levels    *Levels
	filter    LevelFilter
	formatter Formatter
}

func (c *config) Formatter() Formatter {
	return c.formatter
}

func (c *config) Levels() *Levels {
	return c.levels
}

func (c *config) IsActive(level Level) bool {
	return c.filter.IsActive(level)
}

func (c *config) Fatal() Level {
	return c.levels.Fatal
}

func (c *config) Error() Level {
	return c.levels.Error
}

func (c *config) Warn() Level {
	return c.levels.Warn
}

func (c *config) Info() Level {
	return c.levels.Info
}

func (c *config) Debug() Level {
	return c.levels.Debug
}

func (c *config) Trace() Level {
	return c.levels.Trace
}

package log

import (
	"os"

	"github.com/domonda/golog"
)

// Defaults
var (
	// Levels are the default log levels.
	Levels = &golog.DefaultLevels

	// Format is the default log message format.
	Format = *golog.NewDefaultFormat()

	// Colorizer is the default log message colorizer.
	Colorizer = *NewStyledColorizer()

	// Config is the default log configuration using golog.DefaultLevels
	// and filtering out all log levels below the value of the LOG_LEVEL
	// environment variable or "DEBUG" if the LOG_LEVEL environment variable
	// is not set.
	//
	// The log output is written to stdout using a colorized text writer
	// if the process is attached to a terminal or else using a JSON writer.
	Config = golog.NewConfig(
		Levels,
		Levels.LevelOfNameOrDefault(os.Getenv("LOG_LEVEL"), Levels.Debug).FilterOutBelow(),
		golog.DecideWriterConfigForTerminal(
			golog.NewTextWriterConfig(os.Stdout, &Format, &Colorizer),
			golog.NewJSONWriterConfig(os.Stdout, &Format),
		),
	)

	// Logger uses a [golog.DerivedConfig] referencing the
	// exported package variable [Config].
	// This way [Config] can be changed after initialization of [Logger]
	// without the need to create and set a new [golog.Logger].
	Logger = golog.NewLogger(golog.NewDerivedConfig(&Config))
)

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

	// Config is the default log configuration.
	Config = golog.NewConfig(
		Levels,
		Levels.LevelOfNameOrDefault(os.Getenv("LOG_LEVEL"), Levels.Debug).FilterOutBelow(),
		golog.NewTextWriterConfig(os.Stdout, &Format, &Colorizer),
	)

	// Logger uses a golog.DerivedConfig referencing the
	// exported package variable Config.
	// This way Config can be changed after initialization of Logger
	// without the need to create and set a new golog.Logger.
	Logger = golog.NewLogger(golog.NewDerivedConfig(&Config))
)

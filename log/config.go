package log

import (
	"os"

	"github.com/domonda/golog"
)

var (
	Levels = &golog.DefaultLevels

	Format = golog.Format{
		TimestampKey:    "time",
		TimestampFormat: "2006-01-02 15:04:05.000",
		LevelKey:        "level",
		PrefixSep:       ": ",
		MessageKey:      "message",
	}

	Colorizer = *NewStyledColorizer()

	Config = golog.NewConfig(
		Levels,
		Levels.LevelOfNameOrDefault(os.Getenv("LOG_LEVEL"), Levels.Debug).FilterOutBelow(),
		golog.NewTextWriter(os.Stdout, &Format, &Colorizer),
	)

	// Logger uses a golog.DerivedConfig referencing the
	// exported package variable Config.
	// This way Config can be changed after initialization of Logger
	// without the need to create and set a new golog.Logger.
	Logger = golog.NewLogger(golog.NewDerivedConfig(&Config))
)

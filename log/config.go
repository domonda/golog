package log

import (
	"os"

	"github.com/fatih/color"

	"github.com/domonda/golog"
)

var (
	Levels = &golog.DefaultLevels

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
		Levels,
		Levels.LevelOfNameOrDefault(os.Getenv("LOG_LEVEL"), Levels.Debug).FilterOutBelow(),
		golog.NewTextFormatter(os.Stdout, &Format, &Colorizer),
	)

	// Logger uses a golog.DerivedConfig referencing the
	// exported package variable Config.
	// This way Config can be changed after initialization of Logger
	// without the need to create and set a new golog.Logger.
	Logger = golog.NewLogger(golog.NewDerivedConfig(&Config))
)

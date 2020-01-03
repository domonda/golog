package main

import (
	"errors"
	"os"

	color "github.com/fatih/color"

	"github.com/domonda/go-types/uu"
	"github.com/domonda/golog"
	"github.com/domonda/golog/log"
)

func main() {
	colorizer := &golog.ConsoleColorizer{
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

	log.Config = golog.NewConfig(
		log.Levels,
		golog.NoFilter,
		golog.NewTextFormatter(os.Stdout, &log.Format, colorizer),
	)

	log.Fatal("Message").Int("int", 123456).Float("float", 1233456789.99).Bool("t", true).Bool("f", false).UUID("UUID", uu.IDv4()).Err(errors.New("an error")).Log()
	log.Error("Message").Str("String", "Hello World!").Nil("SomethingNil").Log()
	log.Warn("Message").Log()
	log.Info("Message").Log()
	log.Debug("Message").Log()
	log.Trace("Message").Log()
	// log.Logger.NewMessage(-25, "Message").Log()

}

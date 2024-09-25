package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	color "github.com/fatih/color"

	"github.com/domonda/go-types/uu"
	"github.com/domonda/golog"
	"github.com/domonda/golog/log"
)

func main() {
	consoleColorizer := &ConsoleColorizer{
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

	printMessages(consoleColorizer)

	fmt.Println()
	fmt.Println()

	printMessages(log.NewStyledColorizer())
}

func printMessages(colorizer golog.Colorizer) {
	log.Config = golog.NewConfig(
		log.Levels,
		golog.AllLevelsActive,
		golog.NewTextWriterConfig(os.Stdout, &log.Format, colorizer),
	)

	ctx := context.TODO() // Config UUID series

	log.Fatal("Message").Int("int", 123456).Float("float", 1233456789.99).Bool("t", true).Bool("f", false).UUID("UUID", uu.NewID(ctx)).Err(errors.New("an error")).Str("afterError", "blah").Log()
	log.Error("Message").Str("String", "Hello World!").Nil("SomethingNil").Log()
	log.Warn("Message").Log()
	log.Info("Message").Log()
	log.Debug("Message").Log()
	log.Trace("Message").Log()
	// log.Logger.NewMessage(-25, "Message").Log()
}

type ConsoleColorizer struct {
	MsgColor        *color.Color
	OtherLevelColor *color.Color
	FatalLevelColor *color.Color
	ErrorLevelColor *color.Color
	WarnLevelColor  *color.Color
	InfoLevelColor  *color.Color
	DebugLevelColor *color.Color
	TraceLevelColor *color.Color
	TimespampColor  *color.Color
	KeyColor        *color.Color
	NilColor        *color.Color
	TrueColor       *color.Color
	FalseColor      *color.Color
	IntColor        *color.Color
	UintColor       *color.Color
	FloatColor      *color.Color
	StringColor     *color.Color
	ErrorColor      *color.Color
	UUIDColor       *color.Color
}

func (c *ConsoleColorizer) ColorizeMsg(str string) string {
	if c.MsgColor == nil {
		return str
	}
	return c.MsgColor.Sprint(str)
}

func (c *ConsoleColorizer) ColorizeTimestamp(str string) string {
	if c.TimespampColor == nil {
		return str
	}
	return c.TimespampColor.Sprint(str)
}

func (c *ConsoleColorizer) ColorizeLevel(levels *golog.Levels, level golog.Level) string {
	levelColor := c.OtherLevelColor
	switch level {
	case levels.Fatal:
		levelColor = c.FatalLevelColor
	case levels.Error:
		levelColor = c.ErrorLevelColor
	case levels.Warn:
		levelColor = c.WarnLevelColor
	case levels.Info:
		levelColor = c.InfoLevelColor
	case levels.Debug:
		levelColor = c.DebugLevelColor
	case levels.Trace:
		levelColor = c.TraceLevelColor
	}

	name := levels.Name(level)
	if levelColor == nil {
		return name
	}
	return levelColor.Sprint(name)
}

func (c *ConsoleColorizer) ColorizeKey(str string) string {
	if c.KeyColor == nil {
		return str
	}
	return c.KeyColor.Sprint(str)
}

func (c *ConsoleColorizer) ColorizeNil(str string) string {
	if c.NilColor == nil {
		return str
	}
	return c.NilColor.Sprint(str)
}

func (c *ConsoleColorizer) ColorizeTrue(str string) string {
	if c.TrueColor == nil {
		return str
	}
	return c.TrueColor.Sprint(str)
}

func (c *ConsoleColorizer) ColorizeFalse(str string) string {
	if c.FalseColor == nil {
		return str
	}
	return c.FalseColor.Sprint(str)
}

func (c *ConsoleColorizer) ColorizeInt(str string) string {
	if c.IntColor == nil {
		return str
	}
	return c.IntColor.Sprint(str)
}

func (c *ConsoleColorizer) ColorizeUint(str string) string {
	if c.UintColor == nil {
		return str
	}
	return c.UintColor.Sprint(str)
}

func (c *ConsoleColorizer) ColorizeFloat(str string) string {
	if c.FloatColor == nil {
		return str
	}
	return c.FloatColor.Sprint(str)
}

func (c *ConsoleColorizer) ColorizeString(str string) string {
	if c.StringColor == nil {
		return str
	}
	return c.StringColor.Sprint(str)
}

func (c *ConsoleColorizer) ColorizeError(str string) string {
	if c.ErrorColor == nil {
		return str
	}
	return c.ErrorColor.Sprint(str)
}

func (c *ConsoleColorizer) ColorizeUUID(str string) string {
	if c.UUIDColor == nil {
		return str
	}
	return c.UUIDColor.Sprint(str)
}

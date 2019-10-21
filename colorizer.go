package golog

import "github.com/fatih/color"

type Colorizer interface {
	ColorizeMsg(string) string
	ColorizeTimestamp(string) string
	ColorizeLevel(*Levels, Level) string
	ColorizeKey(string) string
	ColorizeNil(string) string
	ColorizeTrue(string) string
	ColorizeFalse(string) string
	ColorizeInt(string) string
	ColorizeUint(string) string
	ColorizeFloat(string) string
	ColorizeString(string) string
	ColorizeError(string) string
	ColorizeUUID(string) string
}

const NoColorizer noColorizer = 0

type noColorizer int

func (noColorizer) ColorizeMsg(str string) string                    { return str }
func (noColorizer) ColorizeTimestamp(str string) string              { return str }
func (noColorizer) ColorizeLevel(levels *Levels, level Level) string { return levels.Name(level) }
func (noColorizer) ColorizeKey(str string) string                    { return str }
func (noColorizer) ColorizeNil(str string) string                    { return str }
func (noColorizer) ColorizeTrue(str string) string                   { return str }
func (noColorizer) ColorizeFalse(str string) string                  { return str }
func (noColorizer) ColorizeInt(str string) string                    { return str }
func (noColorizer) ColorizeUint(str string) string                   { return str }
func (noColorizer) ColorizeFloat(str string) string                  { return str }
func (noColorizer) ColorizeString(str string) string                 { return str }
func (noColorizer) ColorizeError(str string) string                  { return str }
func (noColorizer) ColorizeUUID(str string) string                   { return str }

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

func (c *ConsoleColorizer) ColorizeLevel(levels *Levels, level Level) string {
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

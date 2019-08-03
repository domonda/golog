package golog

import "github.com/fatih/color"

type Colorizer interface {
	ColorizeKey(string) string
	ColorizeTrue(string) string
	ColorizeFalse(string) string
	ColorizeInt(string) string
	ColorizeUint(string) string
	ColorizeFloat(string) string
	ColorizeString(string) string
	ColorizeUUID(string) string
	ColorizeTimestamp(string) string
	ColorizeLevel(string) string
	ColorizeMsg(string) string
}

var (
	DefaultColorizer  Colorizer = inactiveColorizer{}
	InactiveColorizer Colorizer = inactiveColorizer{}
)

type inactiveColorizer struct{}

func (inactiveColorizer) ColorizeKey(str string) string       { return str }
func (inactiveColorizer) ColorizeTrue(str string) string      { return str }
func (inactiveColorizer) ColorizeFalse(str string) string     { return str }
func (inactiveColorizer) ColorizeInt(str string) string       { return str }
func (inactiveColorizer) ColorizeUint(str string) string      { return str }
func (inactiveColorizer) ColorizeFloat(str string) string     { return str }
func (inactiveColorizer) ColorizeString(str string) string    { return str }
func (inactiveColorizer) ColorizeUUID(str string) string      { return str }
func (inactiveColorizer) ColorizeTimestamp(str string) string { return str }
func (inactiveColorizer) ColorizeLevel(str string) string     { return str }
func (inactiveColorizer) ColorizeMsg(str string) string       { return str }

type ConsoleColorizer struct {
	KeyColor       *color.Color
	TrueColor      *color.Color
	FalseColor     *color.Color
	IntColor       *color.Color
	UintColor      *color.Color
	FloatColor     *color.Color
	StringColor    *color.Color
	UUIDColor      *color.Color
	TimespampColor *color.Color
	LevelColor     *color.Color
	MsgColor       *color.Color
}

func (c *ConsoleColorizer) ColorizeKey(str string) string {
	if c.KeyColor == nil {
		return str
	}
	return c.KeyColor.Sprint(str)
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

func (c *ConsoleColorizer) ColorizeUUID(str string) string {
	if c.UUIDColor == nil {
		return str
	}
	return c.UUIDColor.Sprint(str)
}

func (c *ConsoleColorizer) ColorizeTimespamp(str string) string {
	if c.TimespampColor == nil {
		return str
	}
	return c.TimespampColor.Sprint(str)
}

func (c *ConsoleColorizer) ColorizeLevel(str string) string {
	if c.LevelColor == nil {
		return str
	}
	return c.LevelColor.Sprint(str)
}

func (c *ConsoleColorizer) ColorizeMsg(str string) string {
	if c.MsgColor == nil {
		return str
	}
	return c.MsgColor.Sprint(str)
}

package golog

import "github.com/muesli/termenv"

var _ Colorizer = new(StyledColorizer) // make sure StyledColorizer implements Colorizer

type StyledColorizer struct {
	MsgStyle        termenv.Style
	OtherLevelStyle termenv.Style
	FatalLevelStyle termenv.Style
	ErrorLevelStyle termenv.Style
	WarnLevelStyle  termenv.Style
	InfoLevelStyle  termenv.Style
	DebugLevelStyle termenv.Style
	TraceLevelStyle termenv.Style
	TimespampStyle  termenv.Style
	KeyStyle        termenv.Style
	NilStyle        termenv.Style
	TrueStyle       termenv.Style
	FalseStyle      termenv.Style
	IntStyle        termenv.Style
	UintStyle       termenv.Style
	FloatStyle      termenv.Style
	StringStyle     termenv.Style
	ErrorStyle      termenv.Style
	UUIDStyle       termenv.Style
}

func (c *StyledColorizer) ColorizeMsg(str string) string {
	return c.MsgStyle.Styled(str)
}

func (c *StyledColorizer) ColorizeTimestamp(str string) string {
	return c.TimespampStyle.Styled(str)
}

func (c *StyledColorizer) ColorizeLevel(levels *Levels, level Level) string {
	var levelStyle termenv.Style
	switch level {
	case levels.Fatal:
		levelStyle = c.FatalLevelStyle
	case levels.Error:
		levelStyle = c.ErrorLevelStyle
	case levels.Warn:
		levelStyle = c.WarnLevelStyle
	case levels.Info:
		levelStyle = c.InfoLevelStyle
	case levels.Debug:
		levelStyle = c.DebugLevelStyle
	case levels.Trace:
		levelStyle = c.TraceLevelStyle
	default:
		levelStyle = c.OtherLevelStyle
	}

	str := levels.Name(level)
	return levelStyle.Styled(str)
}

func (c *StyledColorizer) ColorizeKey(str string) string {
	return c.KeyStyle.Styled(str)
}

func (c *StyledColorizer) ColorizeNil(str string) string {
	return c.NilStyle.Styled(str)
}

func (c *StyledColorizer) ColorizeTrue(str string) string {
	return c.TrueStyle.Styled(str)
}

func (c *StyledColorizer) ColorizeFalse(str string) string {
	return c.FalseStyle.Styled(str)
}

func (c *StyledColorizer) ColorizeInt(str string) string {
	return c.IntStyle.Styled(str)
}

func (c *StyledColorizer) ColorizeUint(str string) string {
	return c.UintStyle.Styled(str)
}

func (c *StyledColorizer) ColorizeFloat(str string) string {
	return c.FloatStyle.Styled(str)
}

func (c *StyledColorizer) ColorizeString(str string) string {
	return c.StringStyle.Styled(str)
}

func (c *StyledColorizer) ColorizeError(str string) string {
	return c.ErrorStyle.Styled(str)
}

func (c *StyledColorizer) ColorizeUUID(str string) string {
	return c.UUIDStyle.Styled(str)
}

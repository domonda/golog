package log

import (
	"github.com/muesli/termenv"

	"github.com/domonda/golog"
)

func NewStyledColorizer() *golog.StyledColorizer {
	profile := termenv.ColorProfile()
	profile = termenv.TrueColor

	return &golog.StyledColorizer{
		TimespampStyle: termenv.String().Foreground(profile.Color("#A0A0A0")),

		OtherLevelStyle: termenv.String().Foreground(profile.Color("#808080")),
		FatalLevelStyle: termenv.String().Foreground(profile.Color("#F00000")).Bold(),
		ErrorLevelStyle: termenv.String().Foreground(profile.Color("#F00000")),
		WarnLevelStyle:  termenv.String().Foreground(profile.Color("#FFAA00")),
		InfoLevelStyle:  termenv.String().Foreground(profile.Color("#00AAAA")),
		DebugLevelStyle: termenv.String().Foreground(profile.Color("#A0A0F0")),
		TraceLevelStyle: termenv.String().Foreground(profile.Color("#808080")),

		MsgStyle:    termenv.String().Foreground(profile.Color("#FFFFFF")),
		KeyStyle:    termenv.String().Foreground(profile.Color("#00DDDD")),
		NilStyle:    termenv.String().Foreground(profile.Color("#F0F0F0")).Italic(),
		TrueStyle:   termenv.String().Foreground(profile.Color("#00CC00")),
		FalseStyle:  termenv.String().Foreground(profile.Color("#BB0000")),
		IntStyle:    termenv.String().Foreground(profile.Color("#F0F0F0")),
		UintStyle:   termenv.String().Foreground(profile.Color("#F0F0F0")),
		FloatStyle:  termenv.String().Foreground(profile.Color("#F0F0F0")),
		UUIDStyle:   termenv.String().Foreground(profile.Color("#F0F0F0")),
		StringStyle: termenv.String().Foreground(profile.Color("#F0F0F0")),
		ErrorStyle:  termenv.String().Foreground(profile.Color("#F00000")),
	}
}

// Colorizer = golog.ConsoleColorizer{
// 	TimespampColor: color.New(color.FgHiBlack),

// 	OtherLevelColor: color.New(color.FgWhite),
// 	FatalLevelColor: color.New(color.FgHiRed),
// 	ErrorLevelColor: color.New(color.FgRed),
// 	WarnLevelColor:  color.New(color.FgYellow),
// 	InfoLevelColor:  color.New(color.FgCyan),
// 	DebugLevelColor: color.New(color.FgMagenta),
// 	TraceLevelColor: color.New(color.FgHiBlack),

// 	MsgColor:    color.New(color.FgHiWhite),
// 	KeyColor:    color.New(color.FgCyan),
// 	NilColor:    color.New(color.FgWhite),
// 	TrueColor:   color.New(color.FgGreen),
// 	FalseColor:  color.New(color.FgYellow),
// 	IntColor:    color.New(color.FgWhite),
// 	UintColor:   color.New(color.FgWhite),
// 	FloatColor:  color.New(color.FgWhite),
// 	UUIDColor:   color.New(color.FgWhite),
// 	StringColor: color.New(color.FgWhite),
// 	ErrorColor:  color.New(color.FgRed),
// }

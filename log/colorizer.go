package log

import (
	"os"

	"github.com/muesli/termenv"

	"github.com/domonda/golog"
)

func NewStyledColorizer() *golog.StyledColorizer {
	profile := termenv.ColorProfile()
	if profile == termenv.Monochrome && os.Getenv("COLORTERM") == "" {
		// $COLORTERM was not set, we assume TrueColor works anyways
		profile = termenv.TrueColor
	}

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
		ErrorStyle:  termenv.String().Foreground(profile.Color("#FFFFFF")).Background(profile.Color("#B00000")),
	}
}

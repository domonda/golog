package log

import (
	"github.com/muesli/termenv"

	"github.com/domonda/golog"
)

func NewStyledColorizer() *golog.StyledColorizer {
	profile := termenv.ColorProfile()

	return &golog.StyledColorizer{
		TimespampStyle: termenv.Style{}.Foreground(profile.Color("#A0A0A0")),

		OtherLevelStyle: termenv.Style{}.Foreground(profile.Color("#808080")),
		FatalLevelStyle: termenv.Style{}.Foreground(profile.Color("#F00000")).Bold(),
		ErrorLevelStyle: termenv.Style{}.Foreground(profile.Color("#F00000")),
		WarnLevelStyle:  termenv.Style{}.Foreground(profile.Color("#FFAA00")),
		InfoLevelStyle:  termenv.Style{}.Foreground(profile.Color("#00AAAA")),
		DebugLevelStyle: termenv.Style{}.Foreground(profile.Color("#A0A0F0")),
		TraceLevelStyle: termenv.Style{}.Foreground(profile.Color("#808080")),

		MsgStyle:    termenv.Style{}.Foreground(profile.Color("#FFFFFF")),
		KeyStyle:    termenv.Style{}.Foreground(profile.Color("#00DDDD")),
		NilStyle:    termenv.Style{}.Foreground(profile.Color("#F0F0F0")).Italic(),
		TrueStyle:   termenv.Style{}.Foreground(profile.Color("#00CC00")),
		FalseStyle:  termenv.Style{}.Foreground(profile.Color("#BB0000")),
		IntStyle:    termenv.Style{}.Foreground(profile.Color("#F0F0F0")),
		UintStyle:   termenv.Style{}.Foreground(profile.Color("#F0F0F0")),
		FloatStyle:  termenv.Style{}.Foreground(profile.Color("#F0F0F0")),
		UUIDStyle:   termenv.Style{}.Foreground(profile.Color("#F0F0F0")),
		StringStyle: termenv.Style{}.Foreground(profile.Color("#F0F0F0")),
		ErrorStyle:  termenv.Style{}.Foreground(profile.Color("#FFFFFF")).Background(profile.Color("#B00000")),
	}
}

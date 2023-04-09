package goslog

import (
	"reflect"
	"testing"

	"github.com/domonda/golog"
	"golang.org/x/exp/slog"
)

func TestConvertDefaultLevels(t *testing.T) {
	tests := []struct {
		name string
		l    slog.Level
		want golog.Level
	}{
		{name: "Debug", l: slog.LevelDebug, want: golog.DefaultLevels.Debug},
		{name: "Debug-1", l: slog.LevelDebug - 1, want: golog.DefaultLevels.Debug - 1},
		{name: "Info", l: slog.LevelInfo, want: golog.DefaultLevels.Info},
		{name: "Warn", l: slog.LevelWarn, want: golog.DefaultLevels.Warn},
		{name: "Error", l: slog.LevelError, want: golog.DefaultLevels.Error},
		{name: "0", l: 0, want: 0},
		{name: "1000", l: 1000, want: golog.LevelInvalid},
		{name: "-1000", l: -1000, want: golog.LevelInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertDefaultLevels(tt.l); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertDefaultLevels(%d) = %v, want %v", tt.l, got, tt.want)
			}
		})
	}
}
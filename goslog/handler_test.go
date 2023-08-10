package goslog

import (
	"log/slog"
	"reflect"
	"testing"
	"testing/slogtest"

	"github.com/stretchr/testify/require"

	"github.com/domonda/golog"
)

func TestHandler(t *testing.T) {
	var rec recorder
	config := golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, &rec)

	handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)

	err := slogtest.TestHandler(handler, func() []map[string]any {
		return rec.Result
	})
	require.NoError(t, err)
}

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

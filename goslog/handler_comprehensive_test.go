package goslog

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/golog"
)

// TestHandler_BasicLogging tests basic logging functionality
func TestHandler_BasicLogging(t *testing.T) {
	var rec recorder
	config := golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, &rec)
	handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)
	logger := slog.New(handler)

	logger.Info("test message")

	require.Len(t, rec.Result, 1)
	assert.Equal(t, "test message", rec.Result[0][slog.MessageKey])
	assert.Equal(t, slog.LevelInfo, rec.Result[0][slog.LevelKey])
}

// TestHandler_AllLevels tests logging at all standard levels
func TestHandler_AllLevels(t *testing.T) {
	tests := []struct {
		name          string
		logFunc       func(*slog.Logger, string)
		wantGologLevel golog.Level
	}{
		{
			name:          "Debug",
			logFunc:       func(l *slog.Logger, msg string) { l.Debug(msg) },
			wantGologLevel: golog.DefaultLevels.Debug,
		},
		{
			name:          "Info",
			logFunc:       func(l *slog.Logger, msg string) { l.Info(msg) },
			wantGologLevel: golog.DefaultLevels.Info,
		},
		{
			name:          "Warn",
			logFunc:       func(l *slog.Logger, msg string) { l.Warn(msg) },
			wantGologLevel: golog.DefaultLevels.Warn,
		},
		{
			name:          "Error",
			logFunc:       func(l *slog.Logger, msg string) { l.Error(msg) },
			wantGologLevel: golog.DefaultLevels.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rec recorder
			config := golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, &rec)
			handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)
			logger := slog.New(handler)

			tt.logFunc(logger, "test message")

			require.Len(t, rec.Result, 1)
			// The recorder stores golog levels (converted from slog levels)
			assert.Equal(t, slog.Level(tt.wantGologLevel), rec.Result[0][slog.LevelKey])
		})
	}
}

// TestHandler_Attributes tests various attribute types
func TestHandler_Attributes(t *testing.T) {
	var rec recorder
	config := golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, &rec)
	handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)
	logger := slog.New(handler)

	now := time.Now()
	duration := 150 * time.Millisecond

	logger.Info("test",
		slog.String("string_key", "string_value"),
		slog.Int("int_key", 42),
		slog.Int64("int64_key", int64(123)),
		slog.Uint64("uint64_key", uint64(456)),
		slog.Float64("float_key", 3.14),
		slog.Bool("bool_key", true),
		slog.Time("time_key", now),
		slog.Duration("duration_key", duration),
	)

	require.Len(t, rec.Result, 1)
	result := rec.Result[0]

	assert.Equal(t, "string_value", result["string_key"])
	assert.Equal(t, int64(42), result["int_key"])
	assert.Equal(t, int64(123), result["int64_key"])
	assert.Equal(t, uint64(456), result["uint64_key"])
	assert.Equal(t, 3.14, result["float_key"])
	assert.Equal(t, true, result["bool_key"])
	// Time and Duration are formatted as strings by golog
	assert.Contains(t, result, "time_key")
	assert.Contains(t, result, "duration_key")
}

// TestHandler_Groups tests attribute grouping
func TestHandler_Groups(t *testing.T) {
	var rec recorder
	config := golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, &rec)
	handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)
	logger := slog.New(handler)

	logger.Info("test",
		slog.Group("request",
			slog.String("method", "GET"),
			slog.String("path", "/api/users"),
			slog.Int("status", 200),
		),
		slog.String("outside", "value"),
	)

	require.Len(t, rec.Result, 1)
	result := rec.Result[0]

	// Check that group is nested
	requestGroup, ok := result["request"].(map[string]any)
	require.True(t, ok, "request should be a map")
	assert.Equal(t, "GET", requestGroup["method"])
	assert.Equal(t, "/api/users", requestGroup["path"])
	assert.Equal(t, int64(200), requestGroup["status"])

	// Check outside attribute
	assert.Equal(t, "value", result["outside"])
}

// TestHandler_NestedGroups tests nested group structures
func TestHandler_NestedGroups(t *testing.T) {
	var rec recorder
	config := golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, &rec)
	handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)
	logger := slog.New(handler)

	logger.Info("test",
		slog.Group("outer",
			slog.String("outer_key", "outer_value"),
			slog.Group("inner",
				slog.String("inner_key", "inner_value"),
			),
		),
	)

	require.Len(t, rec.Result, 1)
	result := rec.Result[0]

	outerGroup, ok := result["outer"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "outer_value", outerGroup["outer_key"])

	innerGroup, ok := outerGroup["inner"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "inner_value", innerGroup["inner_key"])
}

// TestHandler_WithAttrs tests handler with pre-configured attributes
func TestHandler_WithAttrs(t *testing.T) {
	var rec recorder
	config := golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, &rec)
	handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)

	// Create handler with pre-configured attributes
	handlerWithAttrs := handler.WithAttrs([]slog.Attr{
		slog.String("service", "api"),
		slog.Int("version", 2),
	})

	logger := slog.New(handlerWithAttrs)
	logger.Info("test", slog.String("key", "value"))

	require.Len(t, rec.Result, 1)
	result := rec.Result[0]

	// Check pre-configured attributes
	assert.Equal(t, "api", result["service"])
	assert.Equal(t, int64(2), result["version"])
	// Check regular attribute
	assert.Equal(t, "value", result["key"])
}

// TestHandler_WithGroup tests handler with group prefix
func TestHandler_WithGroup(t *testing.T) {
	var rec recorder
	config := golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, &rec)
	handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)

	// Create handler with group prefix
	handlerWithGroup := handler.WithGroup("mygroup")

	logger := slog.New(handlerWithGroup)
	logger.Info("test",
		slog.String("key1", "value1"),
		slog.String("key2", "value2"),
	)

	require.Len(t, rec.Result, 1)
	result := rec.Result[0]

	// All attributes should be under the group
	group, ok := result["mygroup"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "value1", group["key1"])
	assert.Equal(t, "value2", group["key2"])
}

// TestHandler_WithGroupAndAttrs tests combining WithGroup and WithAttrs
func TestHandler_WithGroupAndAttrs(t *testing.T) {
	var rec recorder
	config := golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, &rec)
	handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)

	// First add a group, then add attributes
	handler = handler.WithGroup("service").
		WithAttrs([]slog.Attr{
			slog.String("name", "api"),
			slog.String("env", "prod"),
		})

	logger := slog.New(handler)
	logger.Info("test", slog.String("action", "start"))

	require.Len(t, rec.Result, 1)
	result := rec.Result[0]

	// Attributes should be under the group
	group, ok := result["service"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "api", group["name"])
	assert.Equal(t, "prod", group["env"])
	assert.Equal(t, "start", group["action"])
}

// TestHandler_MultipleGroups tests multiple group nesting
func TestHandler_MultipleGroups(t *testing.T) {
	var rec recorder
	config := golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, &rec)
	handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)

	// Create nested groups
	handler = handler.WithGroup("level1").WithGroup("level2")

	logger := slog.New(handler)
	logger.Info("test", slog.String("key", "value"))

	require.Len(t, rec.Result, 1)
	result := rec.Result[0]

	// Navigate nested groups
	level1, ok := result["level1"].(map[string]any)
	require.True(t, ok)
	level2, ok := level1["level2"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "value", level2["key"])
}

// TestHandler_Enabled tests the Enabled method
func TestHandler_Enabled(t *testing.T) {
	tests := []struct {
		name         string
		filterLevel  golog.Level
		checkLevel   slog.Level
		wantEnabled  bool
	}{
		{
			name:        "Info enabled when filter is Debug",
			filterLevel: golog.DefaultLevels.Debug,
			checkLevel:  slog.LevelInfo,
			wantEnabled: true,
		},
		{
			name:        "Debug disabled when filter is Info",
			filterLevel: golog.DefaultLevels.Info,
			checkLevel:  slog.LevelDebug,
			wantEnabled: false,
		},
		{
			name:        "Error enabled when filter is Warn",
			filterLevel: golog.DefaultLevels.Warn,
			checkLevel:  slog.LevelError,
			wantEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rec recorder
			config := golog.NewConfig(
				&golog.DefaultLevels,
				golog.LevelFilterOutBelow(tt.filterLevel),
				&rec,
			)
			handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)

			enabled := handler.Enabled(context.Background(), tt.checkLevel)
			assert.Equal(t, tt.wantEnabled, enabled)
		})
	}
}

// TestHandler_ContextLogging tests context-aware logging
func TestHandler_ContextLogging(t *testing.T) {
	var rec recorder
	config := golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, &rec)
	handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)
	logger := slog.New(handler)

	ctx := context.Background()
	logger.InfoContext(ctx, "test message", slog.String("key", "value"))

	require.Len(t, rec.Result, 1)
	assert.Equal(t, "test message", rec.Result[0][slog.MessageKey])
	assert.Equal(t, "value", rec.Result[0]["key"])
}

// TestHandler_EmptyGroup tests handling of empty groups
func TestHandler_EmptyGroup(t *testing.T) {
	var rec recorder
	config := golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, &rec)
	handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)

	// WithGroup with empty string should return the same handler
	handlerWithEmptyGroup := handler.WithGroup("")
	assert.Equal(t, handler, handlerWithEmptyGroup)

	logger := slog.New(handler)
	logger.Info("test", slog.String("key", "value"))

	require.Len(t, rec.Result, 1)
	// Attributes should not be grouped
	assert.Equal(t, "value", rec.Result[0]["key"])
}

// TestHandler_EmptyAttrs tests handling of empty attributes
func TestHandler_EmptyAttrs(t *testing.T) {
	var rec recorder
	config := golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, &rec)
	handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)

	// WithAttrs with empty slice should return the same handler
	handlerWithEmptyAttrs := handler.WithAttrs([]slog.Attr{})
	assert.Equal(t, handler, handlerWithEmptyAttrs)
}

// TestHandler_AnyValue tests logging of generic any values
func TestHandler_AnyValue(t *testing.T) {
	var rec recorder
	config := golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, &rec)
	handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)
	logger := slog.New(handler)

	type CustomStruct struct {
		Field1 string
		Field2 int
	}

	custom := CustomStruct{Field1: "test", Field2: 42}
	logger.Info("test", slog.Any("custom", custom))

	require.Len(t, rec.Result, 1)
	// Any value should be present (exact format may vary)
	assert.Contains(t, rec.Result[0], "custom")
}

// testLogValuerImpl is a test type implementing slog.LogValuer
type testLogValuerImpl struct {
	computed *bool
}

func (v testLogValuerImpl) LogValue() slog.Value {
	*v.computed = true
	return slog.StringValue("computed_value")
}

// TestHandler_LogValuer tests lazy evaluation with LogValuer
func TestHandler_LogValuer(t *testing.T) {
	var rec recorder
	config := golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, &rec)
	handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)
	logger := slog.New(handler)

	computed := false
	valuer := testLogValuerImpl{computed: &computed}

	// Log with the LogValuer
	logger.Info("test", slog.Any("lazy", valuer))

	// Value should have been computed
	assert.True(t, computed)

	require.Len(t, rec.Result, 1)
	assert.Equal(t, "computed_value", rec.Result[0]["lazy"])
}

// TestHandler_MultipleMessages tests multiple log messages
func TestHandler_MultipleMessages(t *testing.T) {
	var rec recorder
	config := golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, &rec)
	handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)
	logger := slog.New(handler)

	logger.Info("message 1")
	logger.Warn("message 2")
	logger.Error("message 3")

	require.Len(t, rec.Result, 3)
	assert.Equal(t, "message 1", rec.Result[0][slog.MessageKey])
	assert.Equal(t, "message 2", rec.Result[1][slog.MessageKey])
	assert.Equal(t, "message 3", rec.Result[2][slog.MessageKey])
}

// TestHandler_WithRealWriters tests with actual golog writers
func TestHandler_WithRealWriters(t *testing.T) {
	// This test uses real JSON writer to ensure integration works
	// We won't verify output, just that it doesn't panic
	config := golog.NewConfig(
		&golog.DefaultLevels,
		golog.AllLevelsActive,
		golog.NewJSONWriterConfig(os.Stdout, nil),
	)
	handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)
	logger := slog.New(handler)

	// Should not panic
	logger.Info("test message",
		slog.String("key1", "value1"),
		slog.Group("group1",
			slog.String("nested", "value"),
		),
	)
}

// TestHandler_CustomLevelConverter tests custom level conversion
func TestHandler_CustomLevelConverter(t *testing.T) {
	var rec recorder
	config := golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, &rec)

	// Custom converter that always returns INFO
	customConverter := func(level slog.Level) golog.Level {
		return golog.DefaultLevels.Info
	}

	handler := Handler(golog.NewLogger(config), customConverter)
	logger := slog.New(handler)

	// Log at different levels
	logger.Debug("debug message")
	logger.Error("error message")

	require.Len(t, rec.Result, 2)
	// Both should be logged as INFO due to custom converter
	assert.Equal(t, slog.LevelInfo, rec.Result[0][slog.LevelKey])
	assert.Equal(t, slog.LevelInfo, rec.Result[1][slog.LevelKey])
}

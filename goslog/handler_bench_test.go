package goslog

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/domonda/golog"
)

// BenchmarkHandler_SimpleAttrs benchmarks logging simple attributes
func BenchmarkHandler_SimpleAttrs(b *testing.B) {
	config := golog.NewConfig(
		&golog.DefaultLevels,
		golog.AllLevelsActive,
		golog.NewJSONWriterConfig(io.Discard, nil),
	)
	handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)
	logger := slog.New(handler)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.InfoContext(ctx, "test message",
			"string", "value",
			"int", 42,
			"bool", true,
			"float", 3.14,
		)
	}
}

// BenchmarkHandler_GroupedAttrs benchmarks logging with groups
func BenchmarkHandler_GroupedAttrs(b *testing.B) {
	config := golog.NewConfig(
		&golog.DefaultLevels,
		golog.AllLevelsActive,
		golog.NewJSONWriterConfig(io.Discard, nil),
	)
	handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)
	logger := slog.New(handler)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.InfoContext(ctx, "test message",
			slog.Group("user",
				"id", 123,
				"name", "john",
				"email", "john@example.com",
			),
			slog.Group("request",
				"method", "GET",
				"path", "/api/users",
			),
		)
	}
}

// BenchmarkHandler_ManyAttrs benchmarks logging many attributes
func BenchmarkHandler_ManyAttrs(b *testing.B) {
	config := golog.NewConfig(
		&golog.DefaultLevels,
		golog.AllLevelsActive,
		golog.NewJSONWriterConfig(io.Discard, nil),
	)
	handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)
	logger := slog.New(handler)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.InfoContext(ctx, "test message",
			"attr1", "value1",
			"attr2", "value2",
			"attr3", "value3",
			"attr4", 42,
			"attr5", 3.14,
			"attr6", true,
			"attr7", "value7",
			"attr8", "value8",
			"attr9", "value9",
			"attr10", "value10",
		)
	}
}

// BenchmarkHandler_WithAttrs benchmarks handler with pre-configured attributes
func BenchmarkHandler_WithAttrs(b *testing.B) {
	config := golog.NewConfig(
		&golog.DefaultLevels,
		golog.AllLevelsActive,
		golog.NewJSONWriterConfig(io.Discard, nil),
	)
	handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)

	// Pre-configure some attributes
	handler = handler.WithAttrs([]slog.Attr{
		slog.String("service", "api"),
		slog.String("env", "prod"),
		slog.Int("version", 2),
	})

	logger := slog.New(handler)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.InfoContext(ctx, "test message", "key", "value")
	}
}

// BenchmarkHandler_WithGroup benchmarks handler with group prefix
func BenchmarkHandler_WithGroup(b *testing.B) {
	config := golog.NewConfig(
		&golog.DefaultLevels,
		golog.AllLevelsActive,
		golog.NewJSONWriterConfig(io.Discard, nil),
	)
	handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)

	// Add a group prefix
	handler = handler.WithGroup("service")

	logger := slog.New(handler)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.InfoContext(ctx, "test message",
			"key1", "value1",
			"key2", "value2",
			"key3", "value3",
		)
	}
}

// BenchmarkHandler_NestedGroups benchmarks deeply nested groups
func BenchmarkHandler_NestedGroups(b *testing.B) {
	config := golog.NewConfig(
		&golog.DefaultLevels,
		golog.AllLevelsActive,
		golog.NewJSONWriterConfig(io.Discard, nil),
	)
	handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)
	logger := slog.New(handler)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.InfoContext(ctx, "test message",
			slog.Group("level1",
				slog.String("key1", "value1"),
				slog.Group("level2",
					slog.String("key2", "value2"),
					slog.Group("level3",
						slog.String("key3", "value3"),
					),
				),
			),
		)
	}
}

// BenchmarkHandler_Enabled benchmarks the Enabled check
func BenchmarkHandler_Enabled(b *testing.B) {
	config := golog.NewConfig(
		&golog.DefaultLevels,
		golog.AllLevelsActive,
		golog.NewJSONWriterConfig(io.Discard, nil),
	)
	handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = handler.Enabled(ctx, slog.LevelInfo)
	}
}

// BenchmarkHandler_DisabledLevel benchmarks logging at a disabled level
func BenchmarkHandler_DisabledLevel(b *testing.B) {
	config := golog.NewConfig(
		&golog.DefaultLevels,
		golog.LevelFilterOutBelow(golog.DefaultLevels.Warn), // Only WARN and above
		golog.NewJSONWriterConfig(io.Discard, nil),
	)
	handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)
	logger := slog.New(handler)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Debug is disabled, should be fast
		logger.DebugContext(ctx, "test message", "key", "value")
	}
}

// BenchmarkConvertDefaultLevels benchmarks level conversion
func BenchmarkConvertDefaultLevels(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = ConvertDefaultLevels(slog.LevelInfo)
	}
}

// Benchmark_prefixKey benchmarks the prefixKey function
func Benchmark_prefixKey(b *testing.B) {
	b.Run("NoGroup", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = prefixKey("", "key")
		}
	})

	b.Run("WithGroup", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = prefixKey("group", "key")
		}
	})

	b.Run("NestedGroup", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = prefixKey("group1.group2.group3", "key")
		}
	})
}

// BenchmarkHandler_TextWriter benchmarks with text writer
func BenchmarkHandler_TextWriter(b *testing.B) {
	config := golog.NewConfig(
		&golog.DefaultLevels,
		golog.AllLevelsActive,
		golog.NewTextWriterConfig(io.Discard, nil, nil),
	)
	handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)
	logger := slog.New(handler)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.InfoContext(ctx, "test message",
			"string", "value",
			"int", 42,
			"bool", true,
		)
	}
}

// BenchmarkHandler_MultipleWriters benchmarks with multiple output writers
func BenchmarkHandler_MultipleWriters(b *testing.B) {
	config := golog.NewConfig(
		&golog.DefaultLevels,
		golog.AllLevelsActive,
		golog.NewJSONWriterConfig(io.Discard, nil),
		golog.NewTextWriterConfig(io.Discard, nil, nil),
	)
	handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)
	logger := slog.New(handler)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.InfoContext(ctx, "test message", "key", "value")
	}
}

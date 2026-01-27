package golog

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	t.Run("nil config returns nil logger", func(t *testing.T) {
		logger := NewLogger(nil)
		assert.Nil(t, logger)
	})

	t.Run("valid config returns logger", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
		logger := NewLogger(config)
		require.NotNil(t, logger)
		assert.NotNil(t, logger.Config())
	})

	t.Run("with per-message attribs", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
		attrib := NewString("key", "value")
		logger := NewLogger(config, attrib)
		require.NotNil(t, logger)
		assert.Len(t, logger.Attribs(), 1)
	})
}

func TestNewLoggerWithPrefix(t *testing.T) {
	t.Run("nil config returns nil logger", func(t *testing.T) {
		logger := NewLoggerWithPrefix(nil, "prefix")
		assert.Nil(t, logger)
	})

	t.Run("valid config with prefix", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
		logger := NewLoggerWithPrefix(config, "myprefix")
		require.NotNil(t, logger)
		assert.Equal(t, "myprefix", logger.Prefix())
	})
}

func TestLogger_Clone(t *testing.T) {
	t.Run("nil logger clone returns nil", func(t *testing.T) {
		var logger *Logger
		clone := logger.Clone()
		assert.Nil(t, clone)
	})

	t.Run("clone creates independent copy", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
		attrib := NewString("key", "value")
		logger := NewLoggerWithPrefix(config, "prefix", attrib)
		clone := logger.Clone()

		require.NotNil(t, clone)
		assert.Equal(t, logger.Prefix(), clone.Prefix())
		assert.Equal(t, len(logger.Attribs()), len(clone.Attribs()))
	})
}

func TestLogger_WithPrefix(t *testing.T) {
	t.Run("nil logger returns nil", func(t *testing.T) {
		var logger *Logger
		result := logger.WithPrefix("prefix")
		assert.Nil(t, result)
	})

	t.Run("sets new prefix", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
		logger := NewLogger(config)
		newLogger := logger.WithPrefix("newprefix")

		require.NotNil(t, newLogger)
		assert.Equal(t, "newprefix", newLogger.Prefix())
		assert.Equal(t, "", logger.Prefix()) // Original unchanged
	})
}

func TestLogger_Prefix(t *testing.T) {
	t.Run("nil logger returns empty string", func(t *testing.T) {
		var logger *Logger
		assert.Equal(t, "", logger.Prefix())
	})

	t.Run("returns prefix", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
		logger := NewLoggerWithPrefix(config, "test")
		assert.Equal(t, "test", logger.Prefix())
	})
}

func TestLogger_Config(t *testing.T) {
	t.Run("nil logger returns nil config", func(t *testing.T) {
		var logger *Logger
		assert.Nil(t, logger.Config())
	})

	t.Run("returns config", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
		logger := NewLogger(config)
		assert.NotNil(t, logger.Config())
	})
}

func TestLogger_Attribs(t *testing.T) {
	t.Run("nil logger returns nil attribs", func(t *testing.T) {
		var logger *Logger
		assert.Nil(t, logger.Attribs())
	})

	t.Run("returns attribs", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
		logger := NewLogger(config, NewString("key", "value"))
		attribs := logger.Attribs()
		require.Len(t, attribs, 1)
		assert.Equal(t, "key", attribs[0].Key())
	})
}

func TestLogger_RemoveAttribs(t *testing.T) {
	t.Run("nil logger does not panic", func(t *testing.T) {
		var logger *Logger
		assert.NotPanics(t, func() {
			logger.RemoveAttribs()
		})
	})

	t.Run("removes attribs", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
		logger := NewLogger(config, NewString("key", "value"))
		require.Len(t, logger.Attribs(), 1)
		logger.RemoveAttribs()
		assert.Nil(t, logger.Attribs())
	})
}

func TestLogger_WithClonedAttribs(t *testing.T) {
	t.Run("nil logger returns nil", func(t *testing.T) {
		var logger *Logger
		result := logger.WithClonedAttribs(NewString("key", "value"))
		assert.Nil(t, result)
	})

	t.Run("empty attribs returns same logger", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
		logger := NewLogger(config)
		result := logger.WithClonedAttribs()
		assert.Same(t, logger, result)
	})

	t.Run("adds new attribs", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
		logger := NewLogger(config)
		newLogger := logger.WithClonedAttribs(NewString("key", "value"))

		require.NotNil(t, newLogger)
		assert.Len(t, newLogger.Attribs(), 1)
		assert.Nil(t, logger.Attribs()) // Original unchanged
	})
}

func TestLogger_WithCtx(t *testing.T) {
	t.Run("adds context attribs", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
		logger := NewLogger(config)

		ctx := ContextWithAttribs(context.Background(), NewString("ctxKey", "ctxValue"))
		newLogger := logger.WithCtx(ctx)

		require.NotNil(t, newLogger)
		assert.Len(t, newLogger.Attribs(), 1)
	})
}

func TestLogger_WithLevelFilter(t *testing.T) {
	t.Run("nil logger returns nil", func(t *testing.T) {
		var logger *Logger
		result := logger.WithLevelFilter(AllLevelsActive)
		assert.Nil(t, result)
	})

	t.Run("creates logger with filter", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
		logger := NewLogger(config)
		filteredLogger := logger.WithLevelFilter(LevelFilterOutBelow(DefaultLevels.Error))

		require.NotNil(t, filteredLogger)
		assert.True(t, filteredLogger.IsActive(context.Background(), DefaultLevels.Error))
		assert.False(t, filteredLogger.IsActive(context.Background(), DefaultLevels.Info))
	})
}

func TestLogger_WithAdditionalWriterConfigs(t *testing.T) {
	t.Run("nil logger returns nil", func(t *testing.T) {
		var logger *Logger
		buf := bytes.NewBuffer(nil)
		result := logger.WithAdditionalWriterConfigs(NewTextWriterConfig(buf, nil, nil))
		assert.Nil(t, result)
	})

	t.Run("empty configs returns same logger", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
		logger := NewLogger(config)
		result := logger.WithAdditionalWriterConfigs()
		assert.Same(t, logger, result)
	})

	t.Run("creates logger with additional writers when adding unique writers", func(t *testing.T) {
		buf1 := bytes.NewBuffer(nil)
		buf2 := bytes.NewBuffer(nil)
		config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf1, nil, nil))
		logger := NewLogger(config)
		newLogger := logger.WithAdditionalWriterConfigs(NewTextWriterConfig(buf2, nil, nil))

		require.NotNil(t, newLogger)
		assert.Len(t, newLogger.Config().WriterConfigs(), 2, "should have both parent and new writer")
	})
}

func TestLogger_IsActive(t *testing.T) {
	t.Run("nil logger returns false", func(t *testing.T) {
		var logger *Logger
		assert.False(t, logger.IsActive(context.Background(), DefaultLevels.Info))
	})

	t.Run("respects level filter", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewConfig(&DefaultLevels, LevelFilterOutBelow(DefaultLevels.Warn), NewTextWriterConfig(buf, nil, nil))
		logger := NewLogger(config)

		assert.True(t, logger.IsActive(context.Background(), DefaultLevels.Warn))
		assert.True(t, logger.IsActive(context.Background(), DefaultLevels.Error))
		assert.False(t, logger.IsActive(context.Background(), DefaultLevels.Info))
		assert.False(t, logger.IsActive(context.Background(), DefaultLevels.Debug))
	})

	t.Run("respects context level decider", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
		logger := NewLogger(config)

		ctx := ContextWithoutLogging(context.Background())
		assert.False(t, logger.IsActive(ctx, DefaultLevels.Info))
	})
}

func TestLogger_Flush(t *testing.T) {
	t.Run("nil logger does not panic", func(t *testing.T) {
		var logger *Logger
		assert.NotPanics(t, func() {
			logger.Flush()
		})
	})

	t.Run("flushes writers", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
		logger := NewLogger(config)
		assert.NotPanics(t, func() {
			logger.Flush()
		})
	})
}

func TestLogger_With(t *testing.T) {
	t.Run("nil logger returns nil message", func(t *testing.T) {
		var logger *Logger
		msg := logger.With()
		assert.Nil(t, msg)
	})

	t.Run("creates message for sub-logger building", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
		logger := NewLogger(config)

		subLogger := logger.With().
			Str("requestID", "12345").
			SubLogger()

		require.NotNil(t, subLogger)
		assert.Len(t, subLogger.Attribs(), 1)
		assert.Equal(t, "requestID", subLogger.Attribs()[0].Key())
	})
}

func TestLogger_NewMessage(t *testing.T) {
	t.Run("nil logger returns nil message", func(t *testing.T) {
		var logger *Logger
		msg := logger.NewMessage(context.Background(), DefaultLevels.Info, "test")
		assert.Nil(t, msg)
	})

	t.Run("filtered level returns nil message", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewConfig(&DefaultLevels, LevelFilterOutBelow(DefaultLevels.Warn), NewTextWriterConfig(buf, nil, nil))
		logger := NewLogger(config)

		msg := logger.NewMessage(context.Background(), DefaultLevels.Info, "test")
		assert.Nil(t, msg)
	})

	t.Run("creates message for active level", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
		logger := NewLogger(config)

		msg := logger.NewMessage(context.Background(), DefaultLevels.Info, "test message")
		require.NotNil(t, msg)
		msg.Log()

		assert.Contains(t, buf.String(), "test message")
	})
}

func TestLogger_NewMessageAt(t *testing.T) {
	timestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	t.Run("with nil context uses background context", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
		logger := NewLogger(config)

		var nilCtx context.Context
		msg := logger.NewMessageAt(nilCtx, timestamp, DefaultLevels.Info, "test")
		require.NotNil(t, msg)
		msg.Log()

		assert.Contains(t, buf.String(), "test")
	})

	t.Run("uses provided timestamp", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		format := &Format{
			TimestampFormat: "2006-01-02 15:04:05",
			TimestampKey:    "time",
			LevelKey:        "level",
			MessageKey:      "message",
		}
		config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, format, nil))
		logger := NewLogger(config)

		msg := logger.NewMessageAt(context.Background(), timestamp, DefaultLevels.Info, "test")
		require.NotNil(t, msg)
		msg.Log()

		assert.Contains(t, buf.String(), "2024-01-15 10:30:00")
	})
}

func TestLogger_NewMessagef(t *testing.T) {
	t.Run("formats message", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
		logger := NewLogger(config)

		msg := logger.NewMessagef(context.Background(), DefaultLevels.Info, "hello %s %d", "world", 42)
		require.NotNil(t, msg)
		msg.Log()

		assert.Contains(t, buf.String(), "hello world 42")
	})
}

func TestLogger_LevelMethods(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
	logger := NewLogger(config)
	ctx := context.Background()
	timestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		logFunc  func() *Message
		expected string
	}{
		{"Fatal", func() *Message { return logger.Fatal("fatal msg") }, "FATAL"},
		{"FatalCtx", func() *Message { return logger.FatalCtx(ctx, "fatal ctx msg") }, "FATAL"},
		{"Fatalf", func() *Message { return logger.Fatalf("fatal %s", "formatted") }, "FATAL"},
		{"FatalfCtx", func() *Message { return logger.FatalfCtx(ctx, "fatal ctx %s", "formatted") }, "FATAL"},
		{"Error", func() *Message { return logger.Error("error msg") }, "ERROR"},
		{"ErrorAt", func() *Message { return logger.ErrorAt(timestamp, "error at msg") }, "ERROR"},
		{"ErrorCtx", func() *Message { return logger.ErrorCtx(ctx, "error ctx msg") }, "ERROR"},
		{"Errorf", func() *Message { return logger.Errorf("error %s", "formatted") }, "ERROR"},
		{"ErrorfCtx", func() *Message { return logger.ErrorfCtx(ctx, "error ctx %s", "formatted") }, "ERROR"},
		{"Warn", func() *Message { return logger.Warn("warn msg") }, "WARN"},
		{"WarnAt", func() *Message { return logger.WarnAt(timestamp, "warn at msg") }, "WARN"},
		{"WarnCtx", func() *Message { return logger.WarnCtx(ctx, "warn ctx msg") }, "WARN"},
		{"Warnf", func() *Message { return logger.Warnf("warn %s", "formatted") }, "WARN"},
		{"WarnfCtx", func() *Message { return logger.WarnfCtx(ctx, "warn ctx %s", "formatted") }, "WARN"},
		{"Info", func() *Message { return logger.Info("info msg") }, "INFO"},
		{"InfoAt", func() *Message { return logger.InfoAt(timestamp, "info at msg") }, "INFO"},
		{"InfoCtx", func() *Message { return logger.InfoCtx(ctx, "info ctx msg") }, "INFO"},
		{"Infof", func() *Message { return logger.Infof("info %s", "formatted") }, "INFO"},
		{"InfofCtx", func() *Message { return logger.InfofCtx(ctx, "info ctx %s", "formatted") }, "INFO"},
		{"Debug", func() *Message { return logger.Debug("debug msg") }, "DEBUG"},
		{"DebugAt", func() *Message { return logger.DebugAt(timestamp, "debug at msg") }, "DEBUG"},
		{"DebugCtx", func() *Message { return logger.DebugCtx(ctx, "debug ctx msg") }, "DEBUG"},
		{"Debugf", func() *Message { return logger.Debugf("debug %s", "formatted") }, "DEBUG"},
		{"DebugfCtx", func() *Message { return logger.DebugfCtx(ctx, "debug ctx %s", "formatted") }, "DEBUG"},
		{"Trace", func() *Message { return logger.Trace("trace msg") }, "TRACE"},
		{"TraceAt", func() *Message { return logger.TraceAt(timestamp, "trace at msg") }, "TRACE"},
		{"TraceCtx", func() *Message { return logger.TraceCtx(ctx, "trace ctx msg") }, "TRACE"},
		{"Tracef", func() *Message { return logger.Tracef("trace %s", "formatted") }, "TRACE"},
		{"TracefCtx", func() *Message { return logger.TracefCtx(ctx, "trace ctx %s", "formatted") }, "TRACE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			msg := tt.logFunc()
			require.NotNil(t, msg)
			msg.Log()
			assert.Contains(t, buf.String(), tt.expected)
		})
	}
}

func TestLogger_LevelWriters(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
	logger := NewLogger(config)

	tests := []struct {
		name       string
		writerFunc func() *LevelWriter
	}{
		{"FatalWriter", logger.FatalWriter},
		{"ErrorWriter", logger.ErrorWriter},
		{"WarnWriter", logger.WarnWriter},
		{"InfoWriter", logger.InfoWriter},
		{"DebugWriter", logger.DebugWriter},
		{"TraceWriter", logger.TraceWriter},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := tt.writerFunc()
			require.NotNil(t, writer)
		})
	}
}

func TestLogger_NewLevelWriter(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
	logger := NewLogger(config)

	writer := logger.NewLevelWriter(DefaultLevels.Info)
	require.NotNil(t, writer)

	n, err := writer.Write([]byte("test message"))
	require.NoError(t, err)
	assert.Greater(t, n, 0)
	assert.Contains(t, buf.String(), "test message")
}

func TestLogger_ContextWithWriterConfigs(t *testing.T) {
	buf1 := bytes.NewBuffer(nil)
	buf2 := bytes.NewBuffer(nil)

	config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf1, nil, nil))
	logger := NewLogger(config)

	// Add additional writer config to context
	ctx := ContextWithAdditionalWriterConfigs(context.Background(), NewTextWriterConfig(buf2, nil, nil))

	logger.NewMessage(ctx, DefaultLevels.Info, "test message").Log()

	// Both buffers should contain the message
	assert.Contains(t, buf1.String(), "test message")
	assert.Contains(t, buf2.String(), "test message")
}

func TestLogger_PrefixInOutput(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	format := &Format{
		TimestampFormat: "2006-01-02 15:04:05",
		TimestampKey:    "time",
		LevelKey:        "level",
		PrefixFmt:       "[%s] %s",
		MessageKey:      "message",
	}
	config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, format, nil))
	logger := NewLoggerWithPrefix(config, "MyApp")

	logger.Info("hello world").Log()

	output := buf.String()
	assert.Contains(t, output, "[MyApp]")
	assert.Contains(t, output, "hello world")
}

func TestLogger_AttribsLoggedCorrectly(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
	logger := NewLogger(config, NewString("service", "api"))

	logger.Info("test").Str("key", "value").Log()

	output := buf.String()
	assert.Contains(t, output, `service="api"`)
	assert.Contains(t, output, `key="value"`)
}

func TestLogger_ContextAttribsLoggedAfterLoggerAttribs(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
	logger := NewLogger(config, NewString("logger_attr", "logger_value"))

	ctx := ContextWithAttribs(context.Background(), NewString("ctx_attr", "ctx_value"))

	logger.NewMessage(ctx, DefaultLevels.Info, "test").Log()

	output := buf.String()
	// Both should be present
	assert.Contains(t, output, `logger_attr="logger_value"`)
	assert.Contains(t, output, `ctx_attr="ctx_value"`)

	// Logger attribs should come before context attribs
	loggerIdx := strings.Index(output, "logger_attr")
	ctxIdx := strings.Index(output, "ctx_attr")
	assert.Less(t, loggerIdx, ctxIdx, "logger attribs should come before context attribs")
}

func TestLogger_DuplicateKeyHandling(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
	logger := NewLogger(config, NewString("key", "original"))

	// Context attribs with duplicate keys are filtered out to prevent key collisions.
	// Logger attribs take precedence over context attribs.
	ctx := ContextWithAttribs(context.Background(), NewString("key", "from_context"))

	logger.NewMessage(ctx, DefaultLevels.Info, "test").Log()

	output := buf.String()
	// Only the logger attrib value should be present
	assert.Contains(t, output, `key="original"`)
	// The context value with duplicate key should NOT appear
	assert.NotContains(t, output, `key="from_context"`)
	// Verify only one occurrence of the key
	assert.Equal(t, 1, strings.Count(output, "key="), "key should appear exactly once")
}

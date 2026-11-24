package golog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/golog/mempool"
)

func TestNewLogger(t *testing.T) {
	t.Run("nil config returns nil logger", func(t *testing.T) {
		logger := NewLogger(nil)
		assert.Nil(t, logger)
	})

	t.Run("valid config returns logger", func(t *testing.T) {
		config, _ := newTextTestConfig()
		logger := NewLogger(config)
		require.NotNil(t, logger)
		assert.NotNil(t, logger.Config())
	})

	t.Run("with per-message attribs", func(t *testing.T) {
		config, _ := newTextTestConfig()
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
		config, _ := newTextTestConfig()
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
		config, _ := newTextTestConfig()
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
		config, _ := newTextTestConfig()
		logger := NewLogger(config)
		newLogger := logger.WithPrefix("newprefix")

		require.NotNil(t, newLogger)
		assert.Equal(t, "newprefix", newLogger.Prefix())
		assert.Equal(t, "", logger.Prefix()) // Original unchanged
	})

	// Sharing the attribs with the parent logger would let
	// Logger.RemoveAttribs on either logger nil out the attribs of the other
	// one and return their slice to the mempool while it's still referenced.
	t.Run("clones attribs", func(t *testing.T) {
		DrainAllMemPools()
		mempool.RegisterCallbacksWriterForTest(t, io.Discard)

		config, buf := newTextTestConfig()
		attrib := NewString("key", "value")
		logger := NewLogger(config, attrib)
		prefixed := logger.WithPrefix("newprefix")
		require.Len(t, prefixed.Attribs(), 1)
		require.NotSame(t, logger.Attribs()[0], prefixed.Attribs()[0], "prefixed logger has a clone of the attrib")

		// Removing the attribs of the parent returns them to the mempool,
		// which hands them out again for the next attrib of the same type,
		// so a prefixed logger that shared them would log this value.
		logger.RemoveAttribs()
		recycled := NewString("recycled", "recycled")
		require.Len(t, prefixed.Attribs(), 1, "prefixed logger kept its attrib")

		prefixed.Info("test").Log()
		assert.Contains(t, buf.String(), `key="value"`)
		assert.NotContains(t, buf.String(), "recycled")

		recycled.Free()
		prefixed.RemoveAttribs()
		attrib.Free()
		assert.Zero(t, mempool.NumOutstanding(), "every pooled attrib and slice was returned")
	})
}

// TestNewLogger_ClonesAttribs tests that the constructors store clones of the
// passed attribs instead of the attribs and the variadic slice of the caller.
// Sharing an attrib lets Logger.RemoveAttribs of one logger return an attrib
// to the mempool that another owner still uses, and sharing the variadic
// slice puts a slice that never came from the mempool into it, where it is
// too small to be reused but is taken out and put back on every call.
func TestNewLogger_ClonesAttribs(t *testing.T) {
	DrainAllMemPools()
	mempool.RegisterCallbacksWriterForTest(t, io.Discard)

	config, buf := newTextTestConfig()
	attrib := NewString("key", "value")

	logger1 := NewLogger(config, attrib)
	logger2 := NewLoggerWithPrefix(config, "prefix", attrib)
	require.NotSame(t, attrib, logger1.Attribs()[0], "logger has a clone of the passed attrib")
	require.NotSame(t, attrib, logger2.Attribs()[0], "logger has a clone of the passed attrib")
	require.NotSame(t, logger1.Attribs()[0], logger2.Attribs()[0], "every logger has its own clone")
	require.Greater(t, cap(logger1.Attribs()), len(logger1.Attribs()), "pooled slice with spare capacity")

	// Removing the attribs of one owner must not affect the other ones
	logger1.RemoveAttribs()
	logger2.Info("test").Log()
	assert.Contains(t, buf.String(), `key="value"`)
	// Not comparing against NewString, that would allocate another pooled attrib
	assert.Equal(t, "key", attrib.Key(), "the attrib of the caller is unchanged")
	assert.Equal(t, "value", attrib.ValueString(), "the attrib of the caller is unchanged")

	logger2.RemoveAttribs()
	attrib.Free()
	assert.Zero(t, mempool.NumOutstanding(), "every pooled attrib and slice was returned")
}

// TestLogger_DerivedLoggersKeepCtxAttribs tests that every method deriving a
// logger from a sub-logger of Message.SubLoggerContext passes on clones of
// the attribs it inherited from the context. Dropping them would silently
// lose request scoped attribs like a request ID in the derived logger, and
// sharing them would let Logger.RemoveAttribs on either logger return them
// to the mempool while the other one still logs them.
func TestLogger_DerivedLoggersKeepCtxAttribs(t *testing.T) {
	config, buf := newTextTestConfig()
	extraConfig, _ := newTextTestConfig()
	ctx := ContextWithAttribs(context.Background(), NewString("ctxStr", "ctx"))

	derive := map[string]func(*Logger) *Logger{
		"Clone":           (*Logger).Clone,
		"WithLevelFilter": func(l *Logger) *Logger { return l.WithLevelFilter(AllLevelsActive) },
		"WithAdditionalWriterConfigs": func(l *Logger) *Logger {
			return l.WithAdditionalWriterConfigs(extraConfig.WriterConfigs()...)
		},
		"WithPrefix": func(l *Logger) *Logger { return l.WithPrefix("derived") },
	}
	for name, derive := range derive {
		t.Run(name, func(t *testing.T) {
			subLog, _ := NewLogger(config).With().SubLoggerContext(ctx)
			require.Len(t, subLog.CtxAttribs(), 1)

			derived := derive(subLog)
			require.Equal(t, subLog.CtxAttribs(), derived.CtxAttribs(), "derived logger has the inherited ctx attribs")
			require.NotSame(t, subLog.CtxAttribs()[0], derived.CtxAttribs()[0], "derived logger has clones of the inherited ctx attribs")

			derived.RemoveAttribs()
			buf.Reset()
			subLog.Info("test").Log()
			assert.Contains(t, buf.String(), `ctxStr="ctx"`, "removing the attribs of the derived logger doesn't affect the sub-logger")
			subLog.RemoveAttribs()
		})
	}

	t.Run("nil logger", func(t *testing.T) {
		var logger *Logger
		assert.Nil(t, logger.CtxAttribs())
	})
}

// TestLogger_NewMessageAt_InheritedCtxAttribsPoolBalance tests that a message
// of a sub-logger with inherited ctx attribs returns everything it takes from
// the mempool. The message grows its attribs for the logged inherited ones,
// and appending without growing the pooled slice first would let append
// allocate a backing array that the message then puts into the mempool.
func TestLogger_NewMessageAt_InheritedCtxAttribsPoolBalance(t *testing.T) {
	config, _ := newTextTestConfig()
	ctx := ContextWithAttribs(context.Background(), NewString("ctxStr", "ctx"), NewInt("ctxInt", 1))
	// Without logger attribs the message has no pooled slice of its own,
	// so append would allocate a backing array the mempool never handed out
	subLog, _ := NewLogger(config).With().SubLoggerContext(ctx)

	DrainAllMemPools()
	mempool.RegisterCallbacksWriterForTest(t, io.Discard)

	subLog.Info("test").Log()
	assert.Zero(t, mempool.NumOutstanding(), "every pooled attrib and slice of the message was returned")
}

func TestLogger_Prefix(t *testing.T) {
	t.Run("nil logger returns empty string", func(t *testing.T) {
		var logger *Logger
		assert.Equal(t, "", logger.Prefix())
	})

	t.Run("returns prefix", func(t *testing.T) {
		config, _ := newTextTestConfig()
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
		config, _ := newTextTestConfig()
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
		config, _ := newTextTestConfig()
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
		config, _ := newTextTestConfig()
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
		config, _ := newTextTestConfig()
		logger := NewLogger(config)
		result := logger.WithClonedAttribs()
		assert.Same(t, logger, result)
	})

	t.Run("adds new attribs", func(t *testing.T) {
		config, _ := newTextTestConfig()
		logger := NewLogger(config)
		newLogger := logger.WithClonedAttribs(NewString("key", "value"))

		require.NotNil(t, newLogger)
		assert.Len(t, newLogger.Attribs(), 1)
		assert.Nil(t, logger.Attribs()) // Original unchanged
	})
}

func TestLogger_WithCtx(t *testing.T) {
	t.Run("adds context attribs", func(t *testing.T) {
		config, _ := newTextTestConfig()
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
		config, _ := newTextTestConfig()
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
		config, _ := newTextTestConfig()
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
		config, _ := newTextTestConfig()
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
		config, _ := newTextTestConfig()
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
		config, _ := newTextTestConfig()
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
		config, buf := newTextTestConfig()
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
		config, buf := newTextTestConfig()
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
		config, buf := newTextTestConfig()
		logger := NewLogger(config)

		msg := logger.NewMessagef(context.Background(), DefaultLevels.Info, "hello %s %d", "world", 42)
		require.NotNil(t, msg)
		msg.Log()

		assert.Contains(t, buf.String(), "hello world 42")
	})
}

func TestLogger_LevelMethods(t *testing.T) {
	config, buf := newTextTestConfig()
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
	config, _ := newTextTestConfig()
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
	config, buf := newTextTestConfig()
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
	config, buf := newTextTestConfig()
	logger := NewLogger(config, NewString("service", "api"))

	logger.Info("test").Str("key", "value").Log()

	output := buf.String()
	assert.Contains(t, output, `service="api"`)
	assert.Contains(t, output, `key="value"`)
}

func TestLogger_ContextAttribsLoggedAfterLoggerAttribs(t *testing.T) {
	config, buf := newTextTestConfig()
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
	config, buf := newTextTestConfig()
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

// TestLogger_NilSafeLevelMethods verifies the nil-safe contract documented in
// doc.go: a nil *Logger must not panic on the level shortcut methods. Each
// method reads the level from l.config, so a missing nil guard dereferences a
// nil pointer. See https://github.com/domonda/golog issue: nil logger panic.
func TestLogger_NilSafeLevelMethods(t *testing.T) {
	var logger *Logger
	ctx := context.Background()
	now := time.Now()
	err := context.Canceled

	assert.NotPanics(t, func() {
		// Base level methods (the documented example: log.Info(...).Log()).
		logger.Trace("msg").Log()
		logger.Debug("msg").Log()
		logger.Info("msg").Log()
		logger.Warn("msg").Log()
		logger.Error("msg").Err(err).Log()
		logger.Fatal("msg").Log()

		// At variants.
		logger.TraceAt(now, "msg").Log()
		logger.DebugAt(now, "msg").Log()
		logger.InfoAt(now, "msg").Log()
		logger.WarnAt(now, "msg").Log()
		logger.ErrorAt(now, "msg").Log()

		// Ctx variants.
		logger.TraceCtx(ctx, "msg").Log()
		logger.DebugCtx(ctx, "msg").Log()
		logger.InfoCtx(ctx, "msg").Log()
		logger.WarnCtx(ctx, "msg").Log()
		logger.ErrorCtx(ctx, "msg").Log()
		logger.FatalCtx(ctx, "msg").Log()

		// Formatted variants.
		logger.Tracef("msg %d", 1).Log()
		logger.Debugf("msg %d", 1).Log()
		logger.Infof("msg %d", 1).Log()
		logger.Warnf("msg %d", 1).Log()
		logger.Errorf("msg %w", err).Log()
		logger.Fatalf("msg %d", 1).Log()

		// Formatted context variants.
		logger.TracefCtx(ctx, "msg %d", 1).Log()
		logger.DebugfCtx(ctx, "msg %d", 1).Log()
		logger.InfofCtx(ctx, "msg %d", 1).Log()
		logger.WarnfCtx(ctx, "msg %d", 1).Log()
		logger.ErrorfCtx(ctx, "msg %w", err).Log()
		logger.FatalfCtx(ctx, "msg %d", 1).Log()
	})

	// A nil logger must return a nil Message so chained field methods stay safe.
	assert.Nil(t, logger.Info("msg"))
}

// concurrentTestFields logs fields that all derive from the goroutine
// number g and the message number i, so that the output of every message
// can be checked for values of another message, which would happen
// if a pooled Message, Writer or Attrib were used by two messages at once.
func concurrentTestFields(m *Message, g, i int) *Message {
	gi := fmt.Sprintf("%d-%d", g, i)
	return m.
		Int("g", g).
		Int("i", i).
		Str("gi", gi).
		Ints("gis", []int{g, i}).
		Strs("strs", []string{gi, gi}).
		Err(errors.New(gi)).
		Any("any", struct{ G, I int }{g, i}).
		Loggable(Attribs{NewString("loggable", gi)})
}

// requireConcurrentTestJSON checks that a JSON line has the fields
// of concurrentTestFields consistent with each other
// and returns the goroutine and message numbers.
func requireConcurrentTestJSON(t *testing.T, line string) (g, i int) {
	t.Helper()
	var m struct {
		G        int      `json:"g"`
		I        int      `json:"i"`
		GI       string   `json:"gi"`
		GIs      []int    `json:"gis"`
		Strs     []string `json:"strs"`
		Error    string   `json:"error"`
		Any      struct{ G, I int }
		Loggable string `json:"loggable"`
	}
	require.NoError(t, json.Unmarshal([]byte(line), &m), line)
	gi := fmt.Sprintf("%d-%d", m.G, m.I)
	require.Equal(t, gi, m.GI, line)
	require.Equal(t, []int{m.G, m.I}, m.GIs, line)
	require.Equal(t, []string{gi, gi}, m.Strs, line)
	require.Equal(t, gi, m.Error, line)
	require.Equal(t, struct{ G, I int }{m.G, m.I}, m.Any, line)
	require.Equal(t, gi, m.Loggable, line)
	return m.G, m.I
}

// requireAllConcurrentTestMessages checks that the output has exactly one
// line for each combination of goroutine and message number
// and returns the lines.
func requireAllConcurrentTestMessages(t *testing.T, output string, numGoroutines, numMessages int) []string {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.Len(t, lines, numGoroutines*numMessages)
	seen := make(map[[2]int]bool, len(lines))
	for _, line := range lines {
		g, i := requireConcurrentTestJSON(t, line)
		require.False(t, seen[[2]int{g, i}], "duplicate message %d-%d", g, i)
		seen[[2]int{g, i}] = true
	}
	return lines
}

// TestLogger_Concurrent logs with one Logger from many goroutines
// to all writer types, for go test -race. It also checks that every
// message has only its own values and that all pooled objects
// were returned to the mempool exactly once.
func TestLogger_Concurrent(t *testing.T) {
	const (
		numGoroutines = 16
		numMessages   = 64
	)
	var (
		jsonOut, textOut lockedBuffer
		callbackMtx      sync.Mutex
		callbackErrs     []string
		numCallbacks     int
	)
	callback := func(_ time.Time, _ Level, _, _ string, attribs Attribs) {
		// The attribs are freed after the callback,
		// so they must be checked within the callback
		g := attribs.Get("g").(*Int).val
		i := attribs.Get("i").(*Int).val
		gi := attribs.Get("gi").ValueString()

		callbackMtx.Lock()
		defer callbackMtx.Unlock()

		numCallbacks++
		if want := fmt.Sprintf("%d-%d", g, i); gi != want {
			callbackErrs = append(callbackErrs, fmt.Sprintf("gi=%s in message %s", gi, want))
		}
	}
	config := NewConfig(
		&DefaultLevels, // Level names of different lengths use the padded levels cache of the TextWriterConfig
		AllLevelsActive,
		NewJSONWriterConfig(&jsonOut, nil),
		NewTextWriterConfig(&textOut, nil, nil),
		NewCallbackWriterConfig(callback),
	)
	logger := NewLogger(config, NewString("service", "test"))
	sharedCtx := ContextWithAttribs(context.Background(), NewString("shared", "ctx"))
	ctxLogger, _ := logger.With().Str("sub", "ctx").SubLoggerContext(sharedCtx)

	mempool.RegisterCallbacksWriterForTest(t, io.Discard)

	var wg sync.WaitGroup
	for g := range numGoroutines {
		wg.Go(func() {
			for i := range numMessages {
				switch i % 4 {
				case 0:
					concurrentTestFields(logger.Info("msg"), g, i).Log()
				case 1:
					// The message borrows the attribs of the shared context
					concurrentTestFields(logger.WarnCtx(sharedCtx, "msg"), g, i).Log()
				case 2:
					// Logged with attribs inherited from the shared context
					concurrentTestFields(ctxLogger.Error("msg"), g, i).Log()
				case 3:
					concurrentTestFields(logger.Debug("msg").Ctx(sharedCtx), g, i).Log()
				}
			}
		})
	}
	wg.Wait()

	require.Zero(t, mempool.NumOutstanding(), "every pooled object was returned exactly once")

	lines := requireAllConcurrentTestMessages(t, string(jsonOut.Bytes()), numGoroutines, numMessages)
	for _, line := range lines {
		require.Contains(t, line, `"service":"test"`)
		if strings.Contains(line, `"level":"WARN"`) || strings.Contains(line, `"level":"DEBUG"`) {
			require.Contains(t, line, `"shared":"ctx"`)
		}
		if strings.Contains(line, `"level":"ERROR"`) {
			require.Contains(t, line, `"sub":"ctx"`)
			require.Contains(t, line, `"shared":"ctx"`)
		}
	}

	textLines := strings.Split(strings.TrimSpace(string(textOut.Bytes())), "\n")
	require.Len(t, textLines, numGoroutines*numMessages)
	textFields := regexp.MustCompile(` g=(\d+) i=(\d+) gi="(\d+-\d+)"`)
	for _, line := range textLines {
		match := textFields.FindStringSubmatch(line)
		require.NotNil(t, match, line)
		require.Equal(t, match[1]+"-"+match[2], match[3], line)
	}

	require.Equal(t, numGoroutines*numMessages, numCallbacks)
	require.Empty(t, callbackErrs)
}

// TestLogger_ConcurrentDerived derives loggers and contexts from a shared
// logger and context in many goroutines at once, for go test -race.
func TestLogger_ConcurrentDerived(t *testing.T) {
	const numGoroutines = 32
	var out, extraOut, ctxOut lockedBuffer
	logger := NewLogger(
		NewConfig(&DefaultLevels, AllLevelsActive, NewJSONWriterConfig(&out, nil)),
		NewString("service", "test"),
	)
	extraWriter := NewJSONWriterConfig(&extraOut, nil)
	ctxWriter := NewJSONWriterConfig(&ctxOut, nil)
	sharedCtx := ContextWithAttribs(context.Background(), NewString("shared", "ctx"))

	var wg sync.WaitGroup
	for g := range numGoroutines {
		wg.Go(func() {
			l := logger.With().Int("g", g).SubLogger()
			l = l.WithPrefix(fmt.Sprintf("prefix%d", g))
			l = l.WithLevelFilter(AllLevelsActive)
			l = l.WithAdditionalWriterConfigs(extraWriter)
			l = l.WithCtx(sharedCtx)
			l = l.WithClonedAttribs(NewInt("cloned", int64(g)))
			l, ctx := l.With().Int("sub", g).SubLoggerContext(sharedCtx)
			ctx = ContextWithAdditionalWriterConfigs(ctx, ctxWriter)
			ctx = ContextWithAttribs(ctx, NewInt("ctx", int64(g)))
			l = l.Clone()
			concurrentTestFields(l.InfoCtx(ctx, "msg"), g, 0).Log()
			l.RemoveAttribs()
		})
	}
	wg.Wait()

	for _, output := range []string{string(out.Bytes()), string(extraOut.Bytes()), string(ctxOut.Bytes())} {
		lines := requireAllConcurrentTestMessages(t, output, numGoroutines, 1)
		for _, line := range lines {
			g, _ := requireConcurrentTestJSON(t, line)
			require.Contains(t, line, fmt.Sprintf(`"message":"prefix%d: msg"`, g))
			for _, key := range []string{"cloned", "sub", "ctx"} {
				require.Contains(t, line, fmt.Sprintf(`"%s":%d`, key, g))
			}
			require.Contains(t, line, `"service":"test"`)
			require.Contains(t, line, `"shared":"ctx"`)
		}
	}
}

// TestLogger_ConcurrentLevelWriter writes to a LevelWriter
// and its standard library logger from many goroutines.
func TestLogger_ConcurrentLevelWriter(t *testing.T) {
	const numGoroutines = 16
	var out lockedBuffer
	logger := NewLogger(NewConfig(&DefaultLevels, AllLevelsActive, NewJSONWriterConfig(&out, nil)))
	writer := logger.InfoWriter()
	stdLogger := writer.StdLogger()

	var wg sync.WaitGroup
	for g := range numGoroutines {
		wg.Go(func() {
			_, _ = fmt.Fprintf(writer, "write %d\n", g)
			stdLogger.Printf("std %d", g)
		})
	}
	wg.Wait()

	output := string(out.Bytes())
	require.Len(t, strings.Split(strings.TrimSpace(output), "\n"), 2*numGoroutines)
	for g := range numGoroutines {
		require.Contains(t, output, fmt.Sprintf(`"message":"write %d"`, g))
		require.Contains(t, output, fmt.Sprintf(`"message":"std %d"`, g))
	}
}

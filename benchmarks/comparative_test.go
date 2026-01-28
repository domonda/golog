package benchmarks

import (
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/domonda/golog"
	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Common test data for consistent benchmarking
var (
	testMessage      = "test log message"
	testStringKey    = "string_key"
	testStringValue  = "string_value"
	testIntKey       = "int_key"
	testIntValue     = 42
	testFloatKey     = "float_key"
	testFloatValue   = 3.14159
	testBoolKey      = "bool_key"
	testBoolValue    = true
	testErrorKey     = "error"
	testErrorValue   = errors.New("test error")
	testTimeKey      = "timestamp"
	testTimeValue    = time.Now()
	testUserIDKey    = "user_id"
	testUserIDValue  = "user_12345"
	testRequestIDKey = "request_id"
	testRequestID    = "req_abcdef123456"
)

// BenchmarkSimpleMessage benchmarks logging a simple message without fields
func BenchmarkSimpleMessage(b *testing.B) {
	b.Run("golog", func(b *testing.B) {
		logger := golog.NewLogger(golog.NewConfig(
			&golog.DefaultLevels,
			golog.AllLevelsActive,
			golog.NewJSONWriterConfig(io.Discard, nil),
		))

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.Info(testMessage).Log()
		}
	})

	b.Run("zerolog", func(b *testing.B) {
		logger := zerolog.New(io.Discard)

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.Info().Msg(testMessage)
		}
	})

	b.Run("zap", func(b *testing.B) {
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(io.Discard),
			zapcore.InfoLevel,
		)
		logger := zap.New(core)

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.Info(testMessage)
		}
	})

	b.Run("slog", func(b *testing.B) {
		logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.Info(testMessage)
		}
	})

	b.Run("logrus", func(b *testing.B) {
		logger := logrus.New()
		logger.SetOutput(io.Discard)
		logger.SetFormatter(&logrus.JSONFormatter{})

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.Info(testMessage)
		}
	})
}

// BenchmarkWithFields benchmarks logging with several structured fields
func BenchmarkWithFields(b *testing.B) {
	b.Run("golog", func(b *testing.B) {
		logger := golog.NewLogger(golog.NewConfig(
			&golog.DefaultLevels,
			golog.AllLevelsActive,
			golog.NewJSONWriterConfig(io.Discard, nil),
		))

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.Info(testMessage).
				Str(testStringKey, testStringValue).
				Int(testIntKey, testIntValue).
				Float(testFloatKey, testFloatValue).
				Bool(testBoolKey, testBoolValue).
				Log()
		}
	})

	b.Run("zerolog", func(b *testing.B) {
		logger := zerolog.New(io.Discard)

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.Info().
				Str(testStringKey, testStringValue).
				Int(testIntKey, testIntValue).
				Float64(testFloatKey, testFloatValue).
				Bool(testBoolKey, testBoolValue).
				Msg(testMessage)
		}
	})

	b.Run("zap", func(b *testing.B) {
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(io.Discard),
			zapcore.InfoLevel,
		)
		logger := zap.New(core)

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.Info(testMessage,
				zap.String(testStringKey, testStringValue),
				zap.Int(testIntKey, testIntValue),
				zap.Float64(testFloatKey, testFloatValue),
				zap.Bool(testBoolKey, testBoolValue),
			)
		}
	})

	b.Run("slog", func(b *testing.B) {
		logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.Info(testMessage,
				testStringKey, testStringValue,
				testIntKey, testIntValue,
				testFloatKey, testFloatValue,
				testBoolKey, testBoolValue,
			)
		}
	})

	b.Run("logrus", func(b *testing.B) {
		logger := logrus.New()
		logger.SetOutput(io.Discard)
		logger.SetFormatter(&logrus.JSONFormatter{})

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.WithFields(logrus.Fields{
				testStringKey: testStringValue,
				testIntKey:    testIntValue,
				testFloatKey:  testFloatValue,
				testBoolKey:   testBoolValue,
			}).Info(testMessage)
		}
	})
}

// BenchmarkWithManyFields benchmarks logging with many fields (10 fields)
func BenchmarkWithManyFields(b *testing.B) {
	b.Run("golog", func(b *testing.B) {
		logger := golog.NewLogger(golog.NewConfig(
			&golog.DefaultLevels,
			golog.AllLevelsActive,
			golog.NewJSONWriterConfig(io.Discard, nil),
		))

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.Info(testMessage).
				Str("field1", "value1").
				Str("field2", "value2").
				Int("field3", 100).
				Int("field4", 200).
				Float("field5", 1.23).
				Float("field6", 4.56).
				Bool("field7", true).
				Bool("field8", false).
				Str("field9", "value9").
				Str("field10", "value10").
				Log()
		}
	})

	b.Run("zerolog", func(b *testing.B) {
		logger := zerolog.New(io.Discard)

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.Info().
				Str("field1", "value1").
				Str("field2", "value2").
				Int("field3", 100).
				Int("field4", 200).
				Float64("field5", 1.23).
				Float64("field6", 4.56).
				Bool("field7", true).
				Bool("field8", false).
				Str("field9", "value9").
				Str("field10", "value10").
				Msg(testMessage)
		}
	})

	b.Run("zap", func(b *testing.B) {
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(io.Discard),
			zapcore.InfoLevel,
		)
		logger := zap.New(core)

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.Info(testMessage,
				zap.String("field1", "value1"),
				zap.String("field2", "value2"),
				zap.Int("field3", 100),
				zap.Int("field4", 200),
				zap.Float64("field5", 1.23),
				zap.Float64("field6", 4.56),
				zap.Bool("field7", true),
				zap.Bool("field8", false),
				zap.String("field9", "value9"),
				zap.String("field10", "value10"),
			)
		}
	})

	b.Run("slog", func(b *testing.B) {
		logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.Info(testMessage,
				"field1", "value1",
				"field2", "value2",
				"field3", 100,
				"field4", 200,
				"field5", 1.23,
				"field6", 4.56,
				"field7", true,
				"field8", false,
				"field9", "value9",
				"field10", "value10",
			)
		}
	})

	b.Run("logrus", func(b *testing.B) {
		logger := logrus.New()
		logger.SetOutput(io.Discard)
		logger.SetFormatter(&logrus.JSONFormatter{})

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.WithFields(logrus.Fields{
				"field1":  "value1",
				"field2":  "value2",
				"field3":  100,
				"field4":  200,
				"field5":  1.23,
				"field6":  4.56,
				"field7":  true,
				"field8":  false,
				"field9":  "value9",
				"field10": "value10",
			}).Info(testMessage)
		}
	})
}

// BenchmarkWithAccumulatedContext benchmarks logger with pre-configured context
func BenchmarkWithAccumulatedContext(b *testing.B) {
	b.Run("golog", func(b *testing.B) {
		baseLogger := golog.NewLogger(golog.NewConfig(
			&golog.DefaultLevels,
			golog.AllLevelsActive,
			golog.NewJSONWriterConfig(io.Discard, nil),
		))
		// Create a derived logger with accumulated context
		logger := baseLogger.With().
			Str(testUserIDKey, testUserIDValue).
			Str(testRequestIDKey, testRequestID).
			SubLogger()

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.Info(testMessage).
				Str(testStringKey, testStringValue).
				Int(testIntKey, testIntValue).
				Log()
		}
	})

	b.Run("zerolog", func(b *testing.B) {
		logger := zerolog.New(io.Discard).With().
			Str(testUserIDKey, testUserIDValue).
			Str(testRequestIDKey, testRequestID).
			Logger()

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.Info().
				Str(testStringKey, testStringValue).
				Int(testIntKey, testIntValue).
				Msg(testMessage)
		}
	})

	b.Run("zap", func(b *testing.B) {
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(io.Discard),
			zapcore.InfoLevel,
		)
		logger := zap.New(core).With(
			zap.String(testUserIDKey, testUserIDValue),
			zap.String(testRequestIDKey, testRequestID),
		)

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.Info(testMessage,
				zap.String(testStringKey, testStringValue),
				zap.Int(testIntKey, testIntValue),
			)
		}
	})

	b.Run("slog", func(b *testing.B) {
		logger := slog.New(slog.NewJSONHandler(io.Discard, nil)).With(
			testUserIDKey, testUserIDValue,
			testRequestIDKey, testRequestID,
		)

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.Info(testMessage,
				testStringKey, testStringValue,
				testIntKey, testIntValue,
			)
		}
	})

	b.Run("logrus", func(b *testing.B) {
		baseLogger := logrus.New()
		baseLogger.SetOutput(io.Discard)
		baseLogger.SetFormatter(&logrus.JSONFormatter{})
		logger := baseLogger.WithFields(logrus.Fields{
			testUserIDKey:    testUserIDValue,
			testRequestIDKey: testRequestID,
		})

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.WithFields(logrus.Fields{
				testStringKey: testStringValue,
				testIntKey:    testIntValue,
			}).Info(testMessage)
		}
	})
}

// BenchmarkDisabled benchmarks the overhead when logging is disabled
func BenchmarkDisabled(b *testing.B) {
	b.Run("golog", func(b *testing.B) {
		logger := golog.NewLogger(golog.NewConfig(
			&golog.DefaultLevels,
			golog.LevelFilterOutBelow(golog.DefaultLevels.Warn), // Only WARN and above
			golog.NewJSONWriterConfig(io.Discard, nil),
		))

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			// Debug is disabled
			logger.Debug(testMessage).
				Str(testStringKey, testStringValue).
				Int(testIntKey, testIntValue).
				Log()
		}
	})

	b.Run("zerolog", func(b *testing.B) {
		logger := zerolog.New(io.Discard).Level(zerolog.WarnLevel)

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			// Debug is disabled
			logger.Debug().
				Str(testStringKey, testStringValue).
				Int(testIntKey, testIntValue).
				Msg(testMessage)
		}
	})

	b.Run("zap", func(b *testing.B) {
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(io.Discard),
			zapcore.WarnLevel, // Only WARN and above
		)
		logger := zap.New(core)

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			// Debug is disabled
			logger.Debug(testMessage,
				zap.String(testStringKey, testStringValue),
				zap.Int(testIntKey, testIntValue),
			)
		}
	})

	b.Run("slog", func(b *testing.B) {
		logger := slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelWarn, // Only WARN and above
		}))

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			// Debug is disabled
			logger.Debug(testMessage,
				testStringKey, testStringValue,
				testIntKey, testIntValue,
			)
		}
	})

	b.Run("logrus", func(b *testing.B) {
		logger := logrus.New()
		logger.SetOutput(io.Discard)
		logger.SetFormatter(&logrus.JSONFormatter{})
		logger.SetLevel(logrus.WarnLevel) // Only WARN and above

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			// Debug is disabled
			logger.WithFields(logrus.Fields{
				testStringKey: testStringValue,
				testIntKey:    testIntValue,
			}).Debug(testMessage)
		}
	})
}

// BenchmarkComplexFields benchmarks logging with complex types (errors, time)
func BenchmarkComplexFields(b *testing.B) {
	b.Run("golog", func(b *testing.B) {
		logger := golog.NewLogger(golog.NewConfig(
			&golog.DefaultLevels,
			golog.AllLevelsActive,
			golog.NewJSONWriterConfig(io.Discard, nil),
		))

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.Error(testMessage).
				Str(testStringKey, testStringValue).
				Int(testIntKey, testIntValue).
				Err(testErrorValue).
				Time(testTimeKey, testTimeValue).
				Log()
		}
	})

	b.Run("zerolog", func(b *testing.B) {
		logger := zerolog.New(io.Discard)

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.Error().
				Str(testStringKey, testStringValue).
				Int(testIntKey, testIntValue).
				Err(testErrorValue).
				Time(testTimeKey, testTimeValue).
				Msg(testMessage)
		}
	})

	b.Run("zap", func(b *testing.B) {
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(io.Discard),
			zapcore.InfoLevel,
		)
		logger := zap.New(core)

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.Error(testMessage,
				zap.String(testStringKey, testStringValue),
				zap.Int(testIntKey, testIntValue),
				zap.Error(testErrorValue),
				zap.Time(testTimeKey, testTimeValue),
			)
		}
	})

	b.Run("slog", func(b *testing.B) {
		logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.Error(testMessage,
				testStringKey, testStringValue,
				testIntKey, testIntValue,
				testErrorKey, testErrorValue,
				testTimeKey, testTimeValue,
			)
		}
	})

	b.Run("logrus", func(b *testing.B) {
		logger := logrus.New()
		logger.SetOutput(io.Discard)
		logger.SetFormatter(&logrus.JSONFormatter{})

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.WithFields(logrus.Fields{
				testStringKey: testStringValue,
				testIntKey:    testIntValue,
				testErrorKey:  testErrorValue,
				testTimeKey:   testTimeValue,
			}).Error(testMessage)
		}
	})
}

// BenchmarkTextOutput benchmarks text (non-JSON) output format
func BenchmarkTextOutput(b *testing.B) {
	b.Run("golog", func(b *testing.B) {
		logger := golog.NewLogger(golog.NewConfig(
			&golog.DefaultLevels,
			golog.AllLevelsActive,
			golog.NewTextWriterConfig(io.Discard, nil, nil),
		))

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.Info(testMessage).
				Str(testStringKey, testStringValue).
				Int(testIntKey, testIntValue).
				Log()
		}
	})

	b.Run("zerolog", func(b *testing.B) {
		logger := zerolog.New(zerolog.ConsoleWriter{Out: io.Discard, NoColor: true})

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.Info().
				Str(testStringKey, testStringValue).
				Int(testIntKey, testIntValue).
				Msg(testMessage)
		}
	})

	b.Run("zap", func(b *testing.B) {
		core := zapcore.NewCore(
			zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
			zapcore.AddSync(io.Discard),
			zapcore.InfoLevel,
		)
		logger := zap.New(core)

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.Info(testMessage,
				zap.String(testStringKey, testStringValue),
				zap.Int(testIntKey, testIntValue),
			)
		}
	})

	b.Run("slog", func(b *testing.B) {
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.Info(testMessage,
				testStringKey, testStringValue,
				testIntKey, testIntValue,
			)
		}
	})

	b.Run("logrus", func(b *testing.B) {
		logger := logrus.New()
		logger.SetOutput(io.Discard)
		logger.SetFormatter(&logrus.TextFormatter{})

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			logger.WithFields(logrus.Fields{
				testStringKey: testStringValue,
				testIntKey:    testIntValue,
			}).Info(testMessage)
		}
	})
}

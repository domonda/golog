package benchmarks

import (
	"context"
	"io"
	"testing"

	"github.com/domonda/golog"
)

// BenchmarkGologFeatures measures what golog does beyond the comparative
// scenarios, all with the 4 fields of BenchmarkWithFields:
// writing every message to two writers in one call,
// adding attributes recorded in a sub-logger,
// and adding attributes from the context.Context of the message.
func BenchmarkGologFeatures(b *testing.B) {
	logField := func(m *golog.Message) {
		m.Str(testStringKey, testStringValue).
			Int(testIntKey, testIntValue).
			Float(testFloatKey, testFloatValue).
			Bool(testBoolKey, testBoolValue).
			Log()
	}
	jsonConfig := golog.NewJSONWriterConfig(io.Discard, nil)
	textConfig := golog.NewTextWriterConfig(io.Discard, nil, nil)
	newLogger := func(configs ...golog.WriterConfig) *golog.Logger {
		return golog.NewLogger(golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, configs...))
	}
	ctx := golog.ContextWithAttribs(context.Background(),
		golog.NewString(testUserIDKey, testUserIDValue),
		golog.NewString(testRequestIDKey, testRequestID),
	)

	b.Run("one-writer", func(b *testing.B) {
		cooldown()
		logger := newLogger(jsonConfig)
		b.ReportAllocs()
		for b.Loop() {
			logField(logger.Info(testMessage))
		}
	})

	b.Run("json-and-text-writers", func(b *testing.B) {
		cooldown()
		logger := newLogger(jsonConfig, textConfig)
		b.ReportAllocs()
		for b.Loop() {
			logField(logger.Info(testMessage))
		}
	})

	b.Run("context-attribs", func(b *testing.B) {
		cooldown()
		logger := newLogger(jsonConfig)
		b.ReportAllocs()
		for b.Loop() {
			logField(logger.InfoCtx(ctx, testMessage))
		}
	})

	b.Run("recorded-and-context-attribs-two-writers", func(b *testing.B) {
		cooldown()
		logger := newLogger(jsonConfig, textConfig).With().
			Str("service", "api").
			SubLogger()
		b.ReportAllocs()
		for b.Loop() {
			logField(logger.InfoCtx(ctx, testMessage))
		}
	})
}

package golog

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultFormat(t *testing.T) {
	format := NewDefaultFormat()

	assert.Equal(t, "time", format.TimestampKey)
	assert.Equal(t, "2006-01-02 15:04:05.000", format.TimestampFormat)
	assert.Equal(t, "level", format.LevelKey)
	assert.Equal(t, "%s: %s", format.PrefixFmt)
	assert.Equal(t, "message", format.MessageKey)
	assert.Nil(t, format.Location)
}

func TestFormat_Location(t *testing.T) {
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	// 2024-01-15 10:30:00 UTC == 2024-01-15 19:30:00 +09:00 (Asia/Tokyo)
	timestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	timeAttr := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	format := &Format{
		TimestampKey:    "time",
		TimestampFormat: "2006-01-02 15:04:05 -0700",
		LevelKey:        "level",
		MessageKey:      "message",
		TimeFormat:      "2006-01-02 15:04:05 -0700",
		Location:        tokyo,
	}

	t.Run("TextWriter converts line timestamp and Time attribute", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		writerConfig := NewTextWriterConfig(buf, format, NoColorizer)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, writerConfig)
		log := NewLogger(logConfig)

		log.NewMessageAt(context.Background(), timestamp, logConfig.InfoLevel(), "hello").
			Time("at", timeAttr).
			Log()

		output := buf.String()
		assert.Contains(t, output, "2024-01-15 19:30:00 +0900")
		assert.Contains(t, output, `at="2024-06-01 09:00:00 +0900"`)
	})

	t.Run("JSONWriter converts line timestamp and Time attribute", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		writerConfig := NewJSONWriterConfig(buf, format)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, writerConfig)
		log := NewLogger(logConfig)

		log.NewMessageAt(context.Background(), timestamp, logConfig.InfoLevel(), "hello").
			Time("at", timeAttr).
			Log()

		output := buf.String()
		assert.Contains(t, output, `"time":"2024-01-15 19:30:00 +0900"`)
		assert.Contains(t, output, `"at":"2024-06-01 09:00:00 +0900"`)
	})

	t.Run("nil Location preserves original location", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		f := *format
		f.Location = nil
		writerConfig := NewJSONWriterConfig(buf, &f)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, writerConfig)
		log := NewLogger(logConfig)

		log.NewMessageAt(context.Background(), timestamp, logConfig.InfoLevel(), "hello").
			Time("at", timeAttr).
			Log()

		output := buf.String()
		assert.Contains(t, output, `"time":"2024-01-15 10:30:00 +0000"`)
		assert.Contains(t, output, `"at":"2024-06-01 00:00:00 +0000"`)
	})
}

package golog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ExampleJSONWriter() {
	format := &Format{
		TimestampFormat: "2006-01-02 15:04:05",
		TimestampKey:    "time",
		LevelKey:        "level",
		MessageKey:      "message",
	}
	writerConfig := NewJSONWriterConfig(os.Stdout, format)
	config := NewConfig(&DefaultLevels, AllLevelsActive, writerConfig)
	log := NewLogger(config)

	// Use fixed time for reproducable example output
	at, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")

	log.NewMessageAt(context.Background(), at, config.InfoLevel(), "My log message").
		Int("int", 66).
		Str("str", "Hello\tWorld!\n").
		Log()

	log.NewMessageAt(context.Background(), at, config.ErrorLevel(), "This is an error").Log()

	// Output:
	// {"time":"2006-01-02 15:04:05","level":"INFO","message":"My log message","int":66,"str":"Hello\tWorld!\n"}
	// {"time":"2006-01-02 15:04:05","level":"ERROR","message":"This is an error"}
}

func TestNewJSONWriterConfig(t *testing.T) {
	t.Run("panics with nil writer", func(t *testing.T) {
		assert.PanicsWithValue(t, "nil writer", func() {
			NewJSONWriterConfig(nil, nil)
		})
	})

	t.Run("uses default format when nil", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewJSONWriterConfig(buf, nil)
		require.NotNil(t, config)
	})

	t.Run("uses provided format", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		format := &Format{
			TimestampKey:    "ts",
			TimestampFormat: time.RFC3339,
			LevelKey:        "severity",
			MessageKey:      "msg",
		}
		config := NewJSONWriterConfig(buf, format)
		require.NotNil(t, config)
	})

	t.Run("accepts level filters", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		filter := LevelFilterOutBelow(DefaultLevels.Warn)
		config := NewJSONWriterConfig(buf, nil, filter)
		require.NotNil(t, config)
	})
}

func TestJSONWriterConfig_WriterForNewMessage(t *testing.T) {
	t.Run("returns writer for active level", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewJSONWriterConfig(buf, nil)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		require.NotNil(t, writer)
	})

	t.Run("returns nil for filtered level", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		filter := LevelFilterOutBelow(DefaultLevels.Warn)
		config := NewJSONWriterConfig(buf, nil, filter)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		assert.Nil(t, writer)
	})
}

func TestJSONWriterConfig_FlushUnderlying(t *testing.T) {
	t.Run("does not panic", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewJSONWriterConfig(buf, nil)
		assert.NotPanics(t, func() {
			config.FlushUnderlying()
		})
	})
}

func TestJSONWriter_BeginMessage(t *testing.T) {
	timestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	t.Run("writes timestamp key and level", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		format := &Format{
			TimestampFormat: "2006-01-02 15:04:05",
			TimestampKey:    "time",
			LevelKey:        "level",
			MessageKey:      "message",
		}
		config := NewJSONWriterConfig(buf, format)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test message")
		writer.CommitMessage()

		output := buf.String()
		assert.Contains(t, output, `"time":"2024-01-15 10:30:00"`)
		assert.Contains(t, output, `"level":"INFO"`)
		assert.Contains(t, output, `"message":"test message"`)
	})

	t.Run("omits empty timestamp key", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		format := &Format{
			TimestampFormat: "2006-01-02 15:04:05",
			TimestampKey:    "", // Empty
			LevelKey:        "level",
			MessageKey:      "message",
		}
		config := NewJSONWriterConfig(buf, format)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.CommitMessage()

		output := buf.String()
		assert.NotContains(t, output, "time")
	})

	t.Run("omits empty level key", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		format := &Format{
			TimestampFormat: "2006-01-02 15:04:05",
			TimestampKey:    "time",
			LevelKey:        "", // Empty
			MessageKey:      "message",
		}
		config := NewJSONWriterConfig(buf, format)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.CommitMessage()

		output := buf.String()
		assert.NotContains(t, output, "level")
	})

	t.Run("handles prefix with PrefixFmt", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		format := &Format{
			TimestampFormat: "2006-01-02 15:04:05",
			TimestampKey:    "time",
			LevelKey:        "level",
			PrefixFmt:       "[%s] %s",
			MessageKey:      "message",
		}
		config := NewJSONWriterConfig(buf, format)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "MyApp", "test message")
		writer.CommitMessage()

		output := buf.String()
		assert.Contains(t, output, `"message":"[MyApp] test message"`)
	})
}

func TestJSONWriter_WriteValues(t *testing.T) {
	timestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	format := &Format{
		TimestampFormat: "2006-01-02",
		TimestampKey:    "time",
		LevelKey:        "level",
		MessageKey:      "message",
	}

	t.Run("WriteKey and WriteNil", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewJSONWriterConfig(buf, format)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.WriteKey("myKey")
		writer.WriteNil()
		writer.CommitMessage()

		var result map[string]any
		err := json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)
		assert.Nil(t, result["myKey"])
	})

	t.Run("WriteBool", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewJSONWriterConfig(buf, format)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.WriteKey("active")
		writer.WriteBool(true)
		writer.CommitMessage()

		var result map[string]any
		err := json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, true, result["active"])
	})

	t.Run("WriteInt", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewJSONWriterConfig(buf, format)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.WriteKey("count")
		writer.WriteInt(-123)
		writer.CommitMessage()

		var result map[string]any
		err := json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, float64(-123), result["count"])
	})

	t.Run("WriteUint", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewJSONWriterConfig(buf, format)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.WriteKey("count")
		writer.WriteUint(456)
		writer.CommitMessage()

		var result map[string]any
		err := json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, float64(456), result["count"])
	})

	t.Run("WriteFloat", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewJSONWriterConfig(buf, format)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.WriteKey("price")
		writer.WriteFloat(3.14)
		writer.CommitMessage()

		var result map[string]any
		err := json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, 3.14, result["price"])
	})

	t.Run("WriteString", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewJSONWriterConfig(buf, format)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.WriteKey("name")
		writer.WriteString("John")
		writer.CommitMessage()

		var result map[string]any
		err := json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "John", result["name"])
	})

	t.Run("WriteError nil", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewJSONWriterConfig(buf, format)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.WriteKey("error")
		writer.WriteError(nil)
		writer.CommitMessage()

		var result map[string]any
		err := json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, nil, result["error"])
	})

	t.Run("WriteError", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewJSONWriterConfig(buf, format)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.WriteKey("error")
		writer.WriteError(errors.New("something went wrong"))
		writer.CommitMessage()

		var result map[string]any
		err := json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "something went wrong", result["error"])
	})

	t.Run("WriteUUID", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewJSONWriterConfig(buf, format)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		uuid := MustParseUUID("a547276f-b02b-4e7d-b67e-c6deb07567da")
		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.WriteKey("id")
		writer.WriteUUID(uuid)
		writer.CommitMessage()

		var result map[string]any
		err := json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "a547276f-b02b-4e7d-b67e-c6deb07567da", result["id"])
	})

	t.Run("WriteJSON", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewJSONWriterConfig(buf, format)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.WriteKey("data")
		writer.WriteJSON([]byte(`{"nested":"value"}`))
		writer.CommitMessage()

		output := buf.String()
		assert.Contains(t, output, `"data":{"nested":"value"}`)
	})

	t.Run("WriteJSON with empty value writes null", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewJSONWriterConfig(buf, format)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.WriteKey("data")
		writer.WriteJSON([]byte{})
		writer.CommitMessage()

		var result map[string]any
		err := json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)
		assert.Nil(t, result["data"])
	})
}

func TestJSONWriter_WriteSlice(t *testing.T) {
	timestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	format := &Format{
		TimestampFormat: "2006-01-02",
		TimestampKey:    "time",
		LevelKey:        "level",
		MessageKey:      "message",
	}

	t.Run("WriteSliceKey and WriteSliceEnd", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewJSONWriterConfig(buf, format)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.WriteSliceKey("numbers")
		writer.WriteInt(1)
		writer.WriteInt(2)
		writer.WriteInt(3)
		writer.WriteSliceEnd()
		writer.CommitMessage()

		output := buf.String()
		assert.Contains(t, output, `"numbers":[1,2,3]`)
	})
}

func TestJSONWriter_String(t *testing.T) {
	t.Run("returns current buffer content", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewJSONWriterConfig(buf, nil)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info).(*JSONWriter)
		writer.BeginMessage(logConfig, time.Now(), DefaultLevels.Info, "", "test")

		str := writer.String()
		assert.Contains(t, str, "{")
		assert.Contains(t, str, "test")
	})
}

func TestJSONWriterInterface(t *testing.T) {
	// Verify interface compliance
	var _ Writer = &JSONWriter{}
	var _ WriterConfig = &JSONWriterConfig{}
}

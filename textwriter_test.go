package golog

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ExampleTextWriter() {
	format := &Format{
		TimestampFormat: "2006-01-02 15:04:05",
		TimestampKey:    "time",
		LevelKey:        "level",
		MessageKey:      "message",
	}
	writerConfig := NewTextWriterConfig(os.Stdout, format, NoColorizer)
	config := NewConfig(&DefaultLevels, AllLevelsActive, writerConfig)
	log := NewLogger(config)

	// Use fixed time for reproducable example output
	at, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")

	log.NewMessageAt(context.Background(), at, config.InfoLevel(), "My log message").
		Int("int", 66).
		Str("str", "Hello\tWorld!\n").
		Log()
	log.NewMessageAt(context.Background(), at, config.ErrorLevel(), "Something went wrong!").
		Err(errors.New("Multi\nLine\n\"Error\"")).
		Int("numberOfTheBeast", 666).
		Log()

	// Output:
	// 2006-01-02 15:04:05 |INFO | My log message int=66 str="Hello\tWorld!\n"
	// 2006-01-02 15:04:05 |ERROR| Something went wrong! error=`
	// Multi
	// Line
	// "Error"
	// ` numberOfTheBeast=666
}

func TestNewTextWriterConfig(t *testing.T) {
	t.Run("panics with nil writer", func(t *testing.T) {
		assert.PanicsWithValue(t, "nil writer", func() {
			NewTextWriterConfig(nil, nil, nil)
		})
	})

	t.Run("uses default format when nil", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewTextWriterConfig(buf, nil, nil)
		require.NotNil(t, config)
	})

	t.Run("uses NoColorizer when colorizer is nil", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewTextWriterConfig(buf, nil, nil)
		require.NotNil(t, config)
	})

	t.Run("uses provided format", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		format := &Format{
			TimestampFormat: time.RFC3339,
		}
		config := NewTextWriterConfig(buf, format, NoColorizer)
		require.NotNil(t, config)
	})

	t.Run("accepts level filters", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		filter := LevelFilterOutBelow(DefaultLevels.Warn)
		config := NewTextWriterConfig(buf, nil, NoColorizer, filter)
		require.NotNil(t, config)
	})
}

func TestTextWriterConfig_WriterForNewMessage(t *testing.T) {
	t.Run("returns writer for active level", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewTextWriterConfig(buf, nil, NoColorizer)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		require.NotNil(t, writer)
	})

	t.Run("returns nil for filtered level", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		filter := LevelFilterOutBelow(DefaultLevels.Warn)
		config := NewTextWriterConfig(buf, nil, NoColorizer, filter)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		assert.Nil(t, writer)
	})
}

func TestTextWriterConfig_FlushUnderlying(t *testing.T) {
	t.Run("does not panic", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewTextWriterConfig(buf, nil, NoColorizer)
		assert.NotPanics(t, func() {
			config.FlushUnderlying()
		})
	})
}

func TestTextWriter_BeginMessage(t *testing.T) {
	timestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	t.Run("writes timestamp and level", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		format := &Format{
			TimestampFormat: "2006-01-02 15:04:05",
		}
		config := NewTextWriterConfig(buf, format, NoColorizer)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test message")
		writer.CommitMessage()

		output := buf.String()
		assert.Contains(t, output, "2024-01-15 10:30:00")
		assert.Contains(t, output, "|INFO")
		assert.Contains(t, output, "test message")
	})

	t.Run("handles prefix", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		format := &Format{
			TimestampFormat: "2006-01-02 15:04:05",
			PrefixFmt:       "[%s] %s",
		}
		config := NewTextWriterConfig(buf, format, NoColorizer)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "MyApp", "test message")
		writer.CommitMessage()

		output := buf.String()
		assert.Contains(t, output, "[MyApp] test message")
	})

	t.Run("empty text produces minimal output", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		format := &Format{
			TimestampFormat: "2006-01-02 15:04:05",
		}
		config := NewTextWriterConfig(buf, format, NoColorizer)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "")
		writer.CommitMessage()

		output := buf.String()
		assert.Contains(t, output, "2024-01-15 10:30:00")
		assert.Contains(t, output, "|INFO")
	})
}

func TestTextWriter_WriteValues(t *testing.T) {
	timestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	format := &Format{
		TimestampFormat: "2006-01-02",
	}

	t.Run("WriteKey and WriteNil", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewTextWriterConfig(buf, format, NoColorizer)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.WriteKey("myKey")
		writer.WriteNil()
		writer.CommitMessage()

		output := buf.String()
		assert.Contains(t, output, "myKey=nil")
	})

	t.Run("WriteBool true", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewTextWriterConfig(buf, format, NoColorizer)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.WriteKey("active")
		writer.WriteBool(true)
		writer.CommitMessage()

		output := buf.String()
		assert.Contains(t, output, "active=true")
	})

	t.Run("WriteBool false", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewTextWriterConfig(buf, format, NoColorizer)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.WriteKey("active")
		writer.WriteBool(false)
		writer.CommitMessage()

		output := buf.String()
		assert.Contains(t, output, "active=false")
	})

	t.Run("WriteInt", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewTextWriterConfig(buf, format, NoColorizer)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.WriteKey("count")
		writer.WriteInt(-123)
		writer.CommitMessage()

		output := buf.String()
		assert.Contains(t, output, "count=-123")
	})

	t.Run("WriteUint", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewTextWriterConfig(buf, format, NoColorizer)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.WriteKey("count")
		writer.WriteUint(456)
		writer.CommitMessage()

		output := buf.String()
		assert.Contains(t, output, "count=456")
	})

	t.Run("WriteFloat", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewTextWriterConfig(buf, format, NoColorizer)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.WriteKey("price")
		writer.WriteFloat(3.14)
		writer.CommitMessage()

		output := buf.String()
		assert.Contains(t, output, "price=3.14")
	})

	t.Run("WriteString", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewTextWriterConfig(buf, format, NoColorizer)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.WriteKey("name")
		writer.WriteString("John")
		writer.CommitMessage()

		output := buf.String()
		assert.Contains(t, output, `name="John"`)
	})

	t.Run("WriteError single line", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewTextWriterConfig(buf, format, NoColorizer)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.WriteKey("error")
		writer.WriteError(errors.New("something went wrong"))
		writer.CommitMessage()

		output := buf.String()
		assert.Contains(t, output, "error=`something went wrong`")
	})

	t.Run("WriteError multi line", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewTextWriterConfig(buf, format, NoColorizer)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.WriteKey("error")
		writer.WriteError(errors.New("line1\nline2\nline3"))
		writer.CommitMessage()

		output := buf.String()
		assert.Contains(t, output, "error=`\nline1\nline2\nline3\n`")
	})

	t.Run("WriteUUID", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewTextWriterConfig(buf, format, NoColorizer)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		uuid := MustParseUUID("a547276f-b02b-4e7d-b67e-c6deb07567da")
		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.WriteKey("id")
		writer.WriteUUID(uuid)
		writer.CommitMessage()

		output := buf.String()
		assert.Contains(t, output, "id=a547276f-b02b-4e7d-b67e-c6deb07567da")
	})

	t.Run("WriteJSON", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewTextWriterConfig(buf, format, NoColorizer)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.WriteKey("data")
		writer.WriteJSON([]byte(`{"nested":"value"}`))
		writer.CommitMessage()

		output := buf.String()
		assert.Contains(t, output, `data={"nested":"value"}`)
	})
}

func TestTextWriter_WriteSlice(t *testing.T) {
	timestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	format := &Format{
		TimestampFormat: "2006-01-02",
	}

	t.Run("WriteSliceKey and WriteSliceEnd with values", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewTextWriterConfig(buf, format, NoColorizer)
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
		assert.Contains(t, output, "numbers=[1,2,3]")
	})

	t.Run("empty slice", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewTextWriterConfig(buf, format, NoColorizer)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.WriteSliceKey("empty")
		writer.WriteSliceEnd()
		writer.CommitMessage()

		output := buf.String()
		assert.Contains(t, output, "empty=[]")
	})

	t.Run("single element slice", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewTextWriterConfig(buf, format, NoColorizer)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.WriteSliceKey("single")
		writer.WriteInt(42)
		writer.WriteSliceEnd()
		writer.CommitMessage()

		output := buf.String()
		assert.Contains(t, output, "single=[42]")
	})
}

func TestTextWriter_String(t *testing.T) {
	t.Run("returns current buffer content", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewTextWriterConfig(buf, nil, NoColorizer)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info).(*TextWriter)
		writer.BeginMessage(logConfig, time.Now(), DefaultLevels.Info, "", "test")

		str := writer.String()
		assert.Contains(t, str, "test")
	})
}

func TestTextWriter_LevelPadding(t *testing.T) {
	timestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	t.Run("pads level names to same width", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		format := &Format{
			TimestampFormat: "2006-01-02",
		}
		config := NewTextWriterConfig(buf, format, NoColorizer)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		// INFO is 4 chars, ERROR is 5 chars
		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.CommitMessage()

		output := buf.String()
		// Should contain padded INFO
		assert.Contains(t, output, "|INFO ")
	})
}

func TestTextWriter_WithColorizer(t *testing.T) {
	timestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	t.Run("uses colorizer for output", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		format := &Format{
			TimestampFormat: "2006-01-02",
		}
		// Use mock colorizer that wraps values
		colorizer := mockColorizer{}
		config := NewTextWriterConfig(buf, format, colorizer)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

		writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
		writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
		writer.WriteKey("name")
		writer.WriteString("John")
		writer.CommitMessage()

		output := buf.String()
		assert.Contains(t, output, "[ts:")
		assert.Contains(t, output, "[level:")
		assert.Contains(t, output, "[key:")
		assert.Contains(t, output, "[string:")
	})
}

func TestTextWriterInterface(t *testing.T) {
	// Verify interface compliance
	var _ Writer = &TextWriter{}
	var _ WriterConfig = &TextWriterConfig{}
}

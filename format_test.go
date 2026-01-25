package golog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaultFormat(t *testing.T) {
	format := NewDefaultFormat()

	assert.Equal(t, "time", format.TimestampKey)
	assert.Equal(t, "2006-01-02 15:04:05.000", format.TimestampFormat)
	assert.Equal(t, "level", format.LevelKey)
	assert.Equal(t, "%s: %s", format.PrefixFmt)
	assert.Equal(t, "message", format.MessageKey)
}

func TestFormat_CustomValues(t *testing.T) {
	format := &Format{
		TimestampKey:    "ts",
		TimestampFormat: "2006-01-02T15:04:05Z07:00",
		LevelKey:        "severity",
		PrefixFmt:       "[%s] %s",
		MessageKey:      "msg",
	}

	assert.Equal(t, "ts", format.TimestampKey)
	assert.Equal(t, "2006-01-02T15:04:05Z07:00", format.TimestampFormat)
	assert.Equal(t, "severity", format.LevelKey)
	assert.Equal(t, "[%s] %s", format.PrefixFmt)
	assert.Equal(t, "msg", format.MessageKey)
}

func TestFormat_EmptyKeys(t *testing.T) {
	// Format with empty keys is valid - these fields can be omitted
	format := &Format{
		TimestampKey:    "",
		TimestampFormat: "2006-01-02 15:04:05",
		LevelKey:        "",
		PrefixFmt:       "%s: %s",
		MessageKey:      "",
	}

	assert.Empty(t, format.TimestampKey)
	assert.Empty(t, format.LevelKey)
	assert.Empty(t, format.MessageKey)
}

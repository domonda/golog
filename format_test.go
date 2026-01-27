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

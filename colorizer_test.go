package golog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoColorizer(t *testing.T) {
	t.Run("implements Colorizer interface", func(t *testing.T) {
		var _ Colorizer = NoColorizer
	})

	t.Run("ColorizeMsg returns unchanged string", func(t *testing.T) {
		assert.Equal(t, "test message", NoColorizer.ColorizeMsg("test message"))
		assert.Equal(t, "", NoColorizer.ColorizeMsg(""))
	})

	t.Run("ColorizeTimestamp returns unchanged string", func(t *testing.T) {
		assert.Equal(t, "2024-01-15 10:30:00", NoColorizer.ColorizeTimestamp("2024-01-15 10:30:00"))
	})

	t.Run("ColorizeLevel returns level name", func(t *testing.T) {
		assert.Equal(t, "INFO", NoColorizer.ColorizeLevel(&DefaultLevels, DefaultLevels.Info))
		assert.Equal(t, "ERROR", NoColorizer.ColorizeLevel(&DefaultLevels, DefaultLevels.Error))
		assert.Equal(t, "DEBUG", NoColorizer.ColorizeLevel(&DefaultLevels, DefaultLevels.Debug))
	})

	t.Run("ColorizeKey returns unchanged string", func(t *testing.T) {
		assert.Equal(t, "myKey", NoColorizer.ColorizeKey("myKey"))
	})

	t.Run("ColorizeNil returns unchanged string", func(t *testing.T) {
		assert.Equal(t, "nil", NoColorizer.ColorizeNil("nil"))
		assert.Equal(t, "null", NoColorizer.ColorizeNil("null"))
	})

	t.Run("ColorizeTrue returns unchanged string", func(t *testing.T) {
		assert.Equal(t, "true", NoColorizer.ColorizeTrue("true"))
	})

	t.Run("ColorizeFalse returns unchanged string", func(t *testing.T) {
		assert.Equal(t, "false", NoColorizer.ColorizeFalse("false"))
	})

	t.Run("ColorizeInt returns unchanged string", func(t *testing.T) {
		assert.Equal(t, "-123", NoColorizer.ColorizeInt("-123"))
		assert.Equal(t, "0", NoColorizer.ColorizeInt("0"))
	})

	t.Run("ColorizeUint returns unchanged string", func(t *testing.T) {
		assert.Equal(t, "123", NoColorizer.ColorizeUint("123"))
		assert.Equal(t, "0", NoColorizer.ColorizeUint("0"))
	})

	t.Run("ColorizeFloat returns unchanged string", func(t *testing.T) {
		assert.Equal(t, "3.14159", NoColorizer.ColorizeFloat("3.14159"))
		assert.Equal(t, "-1.5", NoColorizer.ColorizeFloat("-1.5"))
	})

	t.Run("ColorizeString returns unchanged string", func(t *testing.T) {
		assert.Equal(t, `"hello world"`, NoColorizer.ColorizeString(`"hello world"`))
	})

	t.Run("ColorizeError returns unchanged string", func(t *testing.T) {
		assert.Equal(t, "error occurred", NoColorizer.ColorizeError("error occurred"))
	})

	t.Run("ColorizeUUID returns unchanged string", func(t *testing.T) {
		uuid := "a547276f-b02b-4e7d-b67e-c6deb07567da"
		assert.Equal(t, uuid, NoColorizer.ColorizeUUID(uuid))
	})
}

func TestColorizerInterface(t *testing.T) {
	// Test that NoColorizer can be used wherever Colorizer is expected
	var colorizer Colorizer = NoColorizer

	result := colorizer.ColorizeMsg("test")
	assert.Equal(t, "test", result)
}

// mockColorizer is a test colorizer that wraps strings in brackets
type mockColorizer struct{}

func (m mockColorizer) ColorizeMsg(s string) string       { return "[msg:" + s + "]" }
func (m mockColorizer) ColorizeTimestamp(s string) string { return "[ts:" + s + "]" }
func (m mockColorizer) ColorizeLevel(l *Levels, level Level) string {
	return "[level:" + l.Name(level) + "]"
}
func (m mockColorizer) ColorizeKey(s string) string    { return "[key:" + s + "]" }
func (m mockColorizer) ColorizeNil(s string) string    { return "[nil:" + s + "]" }
func (m mockColorizer) ColorizeTrue(s string) string   { return "[true:" + s + "]" }
func (m mockColorizer) ColorizeFalse(s string) string  { return "[false:" + s + "]" }
func (m mockColorizer) ColorizeInt(s string) string    { return "[int:" + s + "]" }
func (m mockColorizer) ColorizeUint(s string) string   { return "[uint:" + s + "]" }
func (m mockColorizer) ColorizeFloat(s string) string  { return "[float:" + s + "]" }
func (m mockColorizer) ColorizeString(s string) string { return "[string:" + s + "]" }
func (m mockColorizer) ColorizeError(s string) string  { return "[error:" + s + "]" }
func (m mockColorizer) ColorizeUUID(s string) string   { return "[uuid:" + s + "]" }

func TestCustomColorizer(t *testing.T) {
	var _ Colorizer = mockColorizer{}

	colorizer := mockColorizer{}
	assert.Equal(t, "[msg:hello]", colorizer.ColorizeMsg("hello"))
	assert.Equal(t, "[ts:2024-01-15]", colorizer.ColorizeTimestamp("2024-01-15"))
	assert.Equal(t, "[level:INFO]", colorizer.ColorizeLevel(&DefaultLevels, DefaultLevels.Info))
	assert.Equal(t, "[key:myKey]", colorizer.ColorizeKey("myKey"))
	assert.Equal(t, "[nil:nil]", colorizer.ColorizeNil("nil"))
	assert.Equal(t, "[true:true]", colorizer.ColorizeTrue("true"))
	assert.Equal(t, "[false:false]", colorizer.ColorizeFalse("false"))
	assert.Equal(t, "[int:42]", colorizer.ColorizeInt("42"))
	assert.Equal(t, "[uint:42]", colorizer.ColorizeUint("42"))
	assert.Equal(t, "[float:3.14]", colorizer.ColorizeFloat("3.14"))
	assert.Equal(t, "[string:text]", colorizer.ColorizeString("text"))
	assert.Equal(t, "[error:err]", colorizer.ColorizeError("err"))
	assert.Equal(t, "[uuid:abc]", colorizer.ColorizeUUID("abc"))
}

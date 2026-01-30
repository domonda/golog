package golog

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	t.Run("panics with nil levels", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		assert.PanicsWithValue(t, "golog.Config needs Levels", func() {
			NewConfig(nil, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
		})
	})

	t.Run("panics with no writers", func(t *testing.T) {
		assert.PanicsWithValue(t, "golog.Config needs a Writer", func() {
			NewConfig(&DefaultLevels, AllLevelsActive)
		})
	})

	t.Run("panics with only nil writers", func(t *testing.T) {
		assert.PanicsWithValue(t, "golog.Config needs a Writer", func() {
			NewConfig(&DefaultLevels, AllLevelsActive, nil, nil)
		})
	})

	t.Run("creates valid config", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
		require.NotNil(t, config)
	})

	t.Run("filters out nil writers", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewConfig(&DefaultLevels, AllLevelsActive, nil, NewTextWriterConfig(buf, nil, nil), nil)
		require.NotNil(t, config)
		assert.Len(t, config.WriterConfigs(), 1)
	})

	t.Run("with multiple writers", func(t *testing.T) {
		buf1 := bytes.NewBuffer(nil)
		buf2 := bytes.NewBuffer(nil)
		config := NewConfig(&DefaultLevels, AllLevelsActive,
			NewTextWriterConfig(buf1, nil, nil),
			NewJSONWriterConfig(buf2, nil))
		require.NotNil(t, config)
		assert.Len(t, config.WriterConfigs(), 2)
	})
}

func TestConfig_WriterConfigs(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	writerConfig := NewTextWriterConfig(buf, nil, nil)
	config := NewConfig(&DefaultLevels, AllLevelsActive, writerConfig)

	writers := config.WriterConfigs()
	require.Len(t, writers, 1)
	assert.Same(t, writerConfig, writers[0])
}

func TestConfig_Levels(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))

	levels := config.Levels()
	assert.Equal(t, &DefaultLevels, levels)
}

func TestConfig_IsActive(t *testing.T) {
	buf := bytes.NewBuffer(nil)

	t.Run("all levels active", func(t *testing.T) {
		config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
		ctx := context.Background()

		assert.True(t, config.IsActive(ctx, DefaultLevels.Trace))
		assert.True(t, config.IsActive(ctx, DefaultLevels.Debug))
		assert.True(t, config.IsActive(ctx, DefaultLevels.Info))
		assert.True(t, config.IsActive(ctx, DefaultLevels.Warn))
		assert.True(t, config.IsActive(ctx, DefaultLevels.Error))
		assert.True(t, config.IsActive(ctx, DefaultLevels.Fatal))
	})

	t.Run("all levels inactive", func(t *testing.T) {
		config := NewConfig(&DefaultLevels, AllLevelsInactive, NewTextWriterConfig(buf, nil, nil))
		ctx := context.Background()

		assert.False(t, config.IsActive(ctx, DefaultLevels.Trace))
		assert.False(t, config.IsActive(ctx, DefaultLevels.Debug))
		assert.False(t, config.IsActive(ctx, DefaultLevels.Info))
		assert.False(t, config.IsActive(ctx, DefaultLevels.Warn))
		assert.False(t, config.IsActive(ctx, DefaultLevels.Error))
		assert.False(t, config.IsActive(ctx, DefaultLevels.Fatal))
	})

	t.Run("filter out below warn", func(t *testing.T) {
		filter := LevelFilterOutBelow(DefaultLevels.Warn)
		config := NewConfig(&DefaultLevels, filter, NewTextWriterConfig(buf, nil, nil))
		ctx := context.Background()

		assert.False(t, config.IsActive(ctx, DefaultLevels.Trace))
		assert.False(t, config.IsActive(ctx, DefaultLevels.Debug))
		assert.False(t, config.IsActive(ctx, DefaultLevels.Info))
		assert.True(t, config.IsActive(ctx, DefaultLevels.Warn))
		assert.True(t, config.IsActive(ctx, DefaultLevels.Error))
		assert.True(t, config.IsActive(ctx, DefaultLevels.Fatal))
	})
}

func TestConfig_LevelMethods(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	config := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))

	assert.Equal(t, DefaultLevels.Fatal, config.FatalLevel())
	assert.Equal(t, DefaultLevels.Error, config.ErrorLevel())
	assert.Equal(t, DefaultLevels.Warn, config.WarnLevel())
	assert.Equal(t, DefaultLevels.Info, config.InfoLevel())
	assert.Equal(t, DefaultLevels.Debug, config.DebugLevel())
	assert.Equal(t, DefaultLevels.Trace, config.TraceLevel())
}

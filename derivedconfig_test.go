package golog

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDerivedConfig(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	parentConfig := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))

	t.Run("panics with nil parent", func(t *testing.T) {
		assert.Panics(t, func() {
			NewDerivedConfig(nil)
		})
	})

	t.Run("panics with nil config pointer", func(t *testing.T) {
		var nilConfig Config
		assert.Panics(t, func() {
			NewDerivedConfig(&nilConfig)
		})
	})

	t.Run("creates valid derived config", func(t *testing.T) {
		derived := NewDerivedConfig(&parentConfig)
		require.NotNil(t, derived)
		assert.Equal(t, parentConfig, derived.Parent())
	})

	t.Run("inherits parent levels", func(t *testing.T) {
		derived := NewDerivedConfig(&parentConfig)
		assert.Equal(t, parentConfig.Levels(), derived.Levels())
	})

	t.Run("inherits parent level methods", func(t *testing.T) {
		derived := NewDerivedConfig(&parentConfig)

		assert.Equal(t, parentConfig.FatalLevel(), derived.FatalLevel())
		assert.Equal(t, parentConfig.ErrorLevel(), derived.ErrorLevel())
		assert.Equal(t, parentConfig.WarnLevel(), derived.WarnLevel())
		assert.Equal(t, parentConfig.InfoLevel(), derived.InfoLevel())
		assert.Equal(t, parentConfig.DebugLevel(), derived.DebugLevel())
		assert.Equal(t, parentConfig.TraceLevel(), derived.TraceLevel())
	})

	t.Run("inherits parent writer configs", func(t *testing.T) {
		derived := NewDerivedConfig(&parentConfig)
		assert.Equal(t, parentConfig.WriterConfigs(), derived.WriterConfigs())
	})

	t.Run("inherits parent IsActive when no filter set", func(t *testing.T) {
		derived := NewDerivedConfig(&parentConfig)
		ctx := context.Background()
		assert.True(t, derived.IsActive(ctx, DefaultLevels.Debug))
	})
}

func TestNewDerivedConfigWithFilter(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	parentConfig := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))

	t.Run("panics with nil parent", func(t *testing.T) {
		assert.Panics(t, func() {
			NewDerivedConfigWithFilter(nil, AllLevelsActive)
		})
	})

	t.Run("creates derived config with filter", func(t *testing.T) {
		filter := LevelFilterOutBelow(DefaultLevels.Warn)
		derived := NewDerivedConfigWithFilter(&parentConfig, filter)
		ctx := context.Background()

		// Uses its own filter, not parent's
		assert.False(t, derived.IsActive(ctx, DefaultLevels.Debug))
		assert.True(t, derived.IsActive(ctx, DefaultLevels.Warn))
	})

	t.Run("with no filters behaves like parent", func(t *testing.T) {
		derived := NewDerivedConfigWithFilter(&parentConfig)
		ctx := context.Background()

		// Uses parent's filter (AllLevelsActive)
		assert.True(t, derived.IsActive(ctx, DefaultLevels.Debug))
	})
}

func TestConfigWithAdditionalWriterConfigs(t *testing.T) {
	buf1 := bytes.NewBuffer(nil)
	buf2 := bytes.NewBuffer(nil)
	parentConfig := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf1, nil, nil))

	t.Run("panics with nil parent", func(t *testing.T) {
		assert.Panics(t, func() {
			ConfigWithAdditionalWriterConfigs(nil, NewTextWriterConfig(buf2, nil, nil))
		})
	})

	t.Run("returns parent when no configs added", func(t *testing.T) {
		result := ConfigWithAdditionalWriterConfigs(&parentConfig)
		assert.Equal(t, parentConfig, result)
	})

	t.Run("creates derived config when adding unique writers", func(t *testing.T) {
		result := ConfigWithAdditionalWriterConfigs(&parentConfig, NewTextWriterConfig(buf2, nil, nil))
		require.NotNil(t, result)
		derived, isDerived := result.(*DerivedConfig)
		require.True(t, isDerived, "should create a DerivedConfig when adding new writers")
		assert.Len(t, derived.WriterConfigs(), 2, "should have both parent and new writer")
	})

	t.Run("returns parent when adding duplicate writers", func(t *testing.T) {
		existingWriter := parentConfig.WriterConfigs()[0]
		result := ConfigWithAdditionalWriterConfigs(&parentConfig, existingWriter)
		require.NotNil(t, result)
		assert.Equal(t, parentConfig, result, "should return parent unchanged when all added writers are duplicates")
	})
}

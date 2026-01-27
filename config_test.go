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

func TestDerivedConfig_SetParent(t *testing.T) {
	buf1 := bytes.NewBuffer(nil)
	buf2 := bytes.NewBuffer(nil)
	parentConfig1 := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf1, nil, nil))
	parentConfig2 := NewConfig(&DefaultLevels, LevelFilterOutBelow(DefaultLevels.Warn), NewTextWriterConfig(buf2, nil, nil))

	derived := NewDerivedConfig(&parentConfig1)
	ctx := context.Background()

	// Initially uses parent1's filter
	assert.True(t, derived.IsActive(ctx, DefaultLevels.Debug))

	// Change parent
	derived.SetParent(&parentConfig2)

	// Now uses parent2's filter (if no own filter set, still uses parent's)
	assert.Equal(t, parentConfig2, derived.Parent())
}

func TestDerivedConfig_SetFilter(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	parentConfig := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
	derived := NewDerivedConfig(&parentConfig)
	ctx := context.Background()

	// Initially uses parent's filter
	assert.True(t, derived.IsActive(ctx, DefaultLevels.Debug))

	// Set own filter
	derived.SetFilter(LevelFilterOutBelow(DefaultLevels.Warn))
	assert.False(t, derived.IsActive(ctx, DefaultLevels.Debug))
	assert.True(t, derived.IsActive(ctx, DefaultLevels.Warn))

	// Clear filter (empty args)
	derived.SetFilter()
	assert.True(t, derived.IsActive(ctx, DefaultLevels.Debug))
}

func TestDerivedConfig_SetAdditionalWriterConfigs(t *testing.T) {
	buf1 := bytes.NewBuffer(nil)
	buf2 := bytes.NewBuffer(nil)
	parentConfig := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf1, nil, nil))
	derived := NewDerivedConfig(&parentConfig)

	// Initially has parent's writers
	assert.Len(t, derived.WriterConfigs(), 1)

	// Add additional writer
	derived.SetAdditionalWriterConfigs(NewTextWriterConfig(buf2, nil, nil))
	assert.Len(t, derived.WriterConfigs(), 2)

	// Clear additional writers
	derived.SetAdditionalWriterConfigs()
	assert.Len(t, derived.WriterConfigs(), 1)
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

func TestConfigInterface(t *testing.T) {
	// Verify that config struct implements Config interface
	buf := bytes.NewBuffer(nil)
	var _ Config = NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
}

func TestDerivedConfigInterface(t *testing.T) {
	// Verify that DerivedConfig implements Config interface
	buf := bytes.NewBuffer(nil)
	parentConfig := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
	var _ Config = NewDerivedConfig(&parentConfig)
}

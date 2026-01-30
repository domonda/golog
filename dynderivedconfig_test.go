package golog

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDynDerivedConfig_SetParent(t *testing.T) {
	buf1 := bytes.NewBuffer(nil)
	buf2 := bytes.NewBuffer(nil)
	parentConfig1 := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf1, nil, nil))
	parentConfig2 := NewConfig(&DefaultLevels, LevelFilterOutBelow(DefaultLevels.Warn), NewTextWriterConfig(buf2, nil, nil))

	derived := NewDynDerivedConfig(&parentConfig1)
	ctx := context.Background()

	// Initially uses parent1's filter
	assert.True(t, derived.IsActive(ctx, DefaultLevels.Debug))

	// Change parent
	derived.SetParent(&parentConfig2)

	// Now uses parent2's filter (if no own filter set, still uses parent's)
	assert.Equal(t, parentConfig2, derived.Parent())
}

func TestDynDerivedConfig_SetFilter(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	parentConfig := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil))
	derived := NewDynDerivedConfig(&parentConfig)
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

func TestDynDerivedConfig_SetAdditionalWriterConfigs(t *testing.T) {
	buf1 := bytes.NewBuffer(nil)
	buf2 := bytes.NewBuffer(nil)
	parentConfig := NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf1, nil, nil))
	derived := NewDynDerivedConfig(&parentConfig)

	// Initially has parent's writers
	assert.Len(t, derived.WriterConfigs(), 1)

	// Add additional writer
	derived.SetAdditionalWriterConfigs(NewTextWriterConfig(buf2, nil, nil))
	assert.Len(t, derived.WriterConfigs(), 2)

	// Clear additional writers
	derived.SetAdditionalWriterConfigs()
	assert.Len(t, derived.WriterConfigs(), 1)
}

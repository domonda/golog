package golog

import (
	"bytes"
	"context"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	parentConfig, _ := newTextTestConfig()
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

// TestDynDerivedConfig_SetParentKeepsAdditionalWriterConfigs checks that
// after SetParent the additional writer configs are combined with the
// writer configs of the new parent instead of those of the old parent.
func TestDynDerivedConfig_SetParentKeepsAdditionalWriterConfigs(t *testing.T) {
	writer1 := NewTextWriterConfig(bytes.NewBuffer(nil), nil, nil)
	writer2 := NewTextWriterConfig(bytes.NewBuffer(nil), nil, nil)
	extraWriter := NewTextWriterConfig(bytes.NewBuffer(nil), nil, nil)
	parent1 := NewConfig(&DefaultLevels, AllLevelsActive, writer1)
	parent2 := NewConfig(&DefaultLevels, AllLevelsActive, writer2)

	derived := NewDynDerivedConfig(&parent1)
	derived.SetAdditionalWriterConfigs(extraWriter)
	// slices.Equal compares the configs by identity, assert.Equal would compare them deeply
	assert.True(t, slices.Equal([]WriterConfig{writer1, extraWriter}, derived.WriterConfigs()))

	derived.SetParent(&parent2)
	assert.True(t, slices.Equal([]WriterConfig{writer2, extraWriter}, derived.WriterConfigs()), "writer configs of the new parent")

	derived.SetAdditionalWriterConfigs()
	assert.True(t, slices.Equal([]WriterConfig{writer2}, derived.WriterConfigs()))
}

// TestDynDerivedConfig_Concurrent changes the parent, filter and writer
// configs of a DynDerivedConfig while other goroutines log with it,
// for go test -race. Every message must be written completely
// to the writers of one of the parents.
func TestDynDerivedConfig_Concurrent(t *testing.T) {
	const (
		numGoroutines = 8
		numMessages   = 100
	)
	var out1, out2, extraOut lockedBuffer
	parent1 := NewConfig(&DefaultLevels, AllLevelsActive, NewJSONWriterConfig(&out1, nil))
	parent2 := NewConfig(&DefaultLevels, AllLevelsActive, NewJSONWriterConfig(&out2, nil))
	extraWriter := NewJSONWriterConfig(&extraOut, nil)
	dyn := NewDynDerivedConfig(&parent1)
	logger := NewLogger(dyn)

	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		for i := 0; ; i++ {
			select {
			case <-done:
				return
			default:
			}
			if i%2 == 0 {
				dyn.SetParent(&parent2)
				dyn.SetFilter(AllLevelsActive)
				dyn.SetAdditionalWriterConfigs(extraWriter)
			} else {
				dyn.SetParent(&parent1)
				dyn.SetFilter()
				dyn.SetAdditionalWriterConfigs()
			}
		}
	})
	var logWG sync.WaitGroup
	for g := range numGoroutines {
		logWG.Go(func() {
			for i := range numMessages {
				concurrentTestFields(logger.Info("msg"), g, i).Log()
			}
		})
	}
	logWG.Wait()
	close(done)
	wg.Wait()

	var numLines int
	for _, output := range []string{string(out1.Bytes()), string(out2.Bytes()), string(extraOut.Bytes())} {
		for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
			if line == "" {
				continue
			}
			requireConcurrentTestJSON(t, line)
			numLines++
		}
	}
	// Every message goes to parent1 or parent2, some also to the extra writer
	require.GreaterOrEqual(t, numLines, numGoroutines*numMessages)
}

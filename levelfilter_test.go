package golog

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_LevelFilter_NoLevel(t *testing.T) {
	for level := LevelMin; level <= LevelMax; level++ {
		assert.True(t, AllLevelsActive.IsActive(context.Background(), level), "NoLevel nevel filters")
	}
}

func Test_LevelFilterOutAllOther(t *testing.T) {
	for activeLevel := LevelMin; activeLevel <= LevelMax; activeLevel++ {
		filter := LevelFilterOutAllOther(activeLevel)
		for level := LevelMin; level <= LevelMax; level++ {
			assert.True(t, filter.IsActive(context.Background(), level) == (level == activeLevel), "only active when activeLevel")
		}
	}
}

func Test_LevelFilter_SetActive(t *testing.T) {
	for activeLevel := LevelMin; activeLevel <= LevelMax; activeLevel++ {
		filter := ^LevelFilter(0)
		filter.SetActive(activeLevel, true)
		for level := LevelMin; level <= LevelMax; level++ {
			assert.True(t, filter.IsActive(context.Background(), level) == (level == activeLevel), "only active when activeLevel")
		}
		filter.SetActive(activeLevel, false)
		for level := LevelMin; level <= LevelMax; level++ {
			assert.True(t, !filter.IsActive(context.Background(), level), "never active because set active to false")
		}
	}
}

func Test_LevelFilterOutBelow(t *testing.T) {
	for refLevel := LevelMin; refLevel <= LevelMax; refLevel++ {
		filter := LevelFilterOutBelow(refLevel)
		for level := LevelMin; level <= LevelMax; level++ {
			filteredOut := !filter.IsActive(context.Background(), level)
			assert.True(t, filteredOut == (level < refLevel), "not active when below refLevel")
		}
	}
}

func Test_LevelFilterOutAbove(t *testing.T) {
	for refLevel := LevelMin; refLevel <= LevelMax; refLevel++ {
		filter := LevelFilterOutAbove(refLevel)
		for level := LevelMin; level <= LevelMax; level++ {
			filteredOut := !filter.IsActive(context.Background(), level)
			assert.True(t, filteredOut == (level > refLevel), "not active when above refLevel")
		}
	}
}

func TestLevelFilter_IsActive(t *testing.T) {
	ctx := context.Background()

	t.Run("AllLevelsInactive disables all levels", func(t *testing.T) {
		for l := LevelMin; l <= LevelMax; l++ {
			assert.False(t, AllLevelsInactive.IsActive(ctx, l), "level %d should be inactive", l)
		}
	})

	t.Run("out of range levels are inactive", func(t *testing.T) {
		assert.False(t, AllLevelsActive.IsActive(ctx, Level(-100)))
		assert.False(t, AllLevelsActive.IsActive(ctx, Level(100)))
		assert.False(t, AllLevelsActive.IsActive(ctx, LevelInvalid))
	})
}

func TestLevelFilter_IsInactive(t *testing.T) {
	ctx := context.Background()

	t.Run("inverse of IsActive for AllLevelsActive", func(t *testing.T) {
		assert.False(t, AllLevelsActive.IsInactive(ctx, DefaultLevels.Info))
	})

	t.Run("inverse of IsActive for AllLevelsInactive", func(t *testing.T) {
		assert.True(t, AllLevelsInactive.IsInactive(ctx, DefaultLevels.Info))
	})
}

func TestLevelFilterOut(t *testing.T) {
	ctx := context.Background()

	t.Run("filters out single level", func(t *testing.T) {
		filter := LevelFilterOut(DefaultLevels.Info)

		assert.True(t, filter.IsActive(ctx, DefaultLevels.Trace))
		assert.True(t, filter.IsActive(ctx, DefaultLevels.Debug))
		assert.False(t, filter.IsActive(ctx, DefaultLevels.Info)) // Filtered out
		assert.True(t, filter.IsActive(ctx, DefaultLevels.Warn))
		assert.True(t, filter.IsActive(ctx, DefaultLevels.Error))
		assert.True(t, filter.IsActive(ctx, DefaultLevels.Fatal))
	})

	t.Run("can filter out trace", func(t *testing.T) {
		filter := LevelFilterOut(DefaultLevels.Trace)
		assert.False(t, filter.IsActive(ctx, DefaultLevels.Trace))
		assert.True(t, filter.IsActive(ctx, DefaultLevels.Debug))
	})

	t.Run("can filter out fatal", func(t *testing.T) {
		filter := LevelFilterOut(DefaultLevels.Fatal)
		assert.True(t, filter.IsActive(ctx, DefaultLevels.Error))
		assert.False(t, filter.IsActive(ctx, DefaultLevels.Fatal))
	})
}

func TestLevelFilterOutBelow_DefaultLevels(t *testing.T) {
	ctx := context.Background()

	t.Run("filters out levels below warn", func(t *testing.T) {
		filter := LevelFilterOutBelow(DefaultLevels.Warn)

		assert.False(t, filter.IsActive(ctx, DefaultLevels.Trace))
		assert.False(t, filter.IsActive(ctx, DefaultLevels.Debug))
		assert.False(t, filter.IsActive(ctx, DefaultLevels.Info))
		assert.True(t, filter.IsActive(ctx, DefaultLevels.Warn))
		assert.True(t, filter.IsActive(ctx, DefaultLevels.Error))
		assert.True(t, filter.IsActive(ctx, DefaultLevels.Fatal))
	})

	t.Run("filters out levels below error", func(t *testing.T) {
		filter := LevelFilterOutBelow(DefaultLevels.Error)

		assert.False(t, filter.IsActive(ctx, DefaultLevels.Warn))
		assert.True(t, filter.IsActive(ctx, DefaultLevels.Error))
		assert.True(t, filter.IsActive(ctx, DefaultLevels.Fatal))
	})
}

func TestLevelFilterOutAbove_DefaultLevels(t *testing.T) {
	ctx := context.Background()

	t.Run("filters out levels above warn", func(t *testing.T) {
		filter := LevelFilterOutAbove(DefaultLevels.Warn)

		assert.True(t, filter.IsActive(ctx, DefaultLevels.Trace))
		assert.True(t, filter.IsActive(ctx, DefaultLevels.Debug))
		assert.True(t, filter.IsActive(ctx, DefaultLevels.Info))
		assert.True(t, filter.IsActive(ctx, DefaultLevels.Warn))
		assert.False(t, filter.IsActive(ctx, DefaultLevels.Error))
		assert.False(t, filter.IsActive(ctx, DefaultLevels.Fatal))
	})
}

func TestLevelFilterOutAllOther_DefaultLevels(t *testing.T) {
	ctx := context.Background()

	t.Run("only allows single level", func(t *testing.T) {
		filter := LevelFilterOutAllOther(DefaultLevels.Warn)

		assert.False(t, filter.IsActive(ctx, DefaultLevels.Trace))
		assert.False(t, filter.IsActive(ctx, DefaultLevels.Debug))
		assert.False(t, filter.IsActive(ctx, DefaultLevels.Info))
		assert.True(t, filter.IsActive(ctx, DefaultLevels.Warn))
		assert.False(t, filter.IsActive(ctx, DefaultLevels.Error))
		assert.False(t, filter.IsActive(ctx, DefaultLevels.Fatal))
	})
}

func TestJoinLevelFilters(t *testing.T) {
	ctx := context.Background()

	t.Run("no filters returns AllLevelsActive", func(t *testing.T) {
		filter := JoinLevelFilters()
		assert.Equal(t, AllLevelsActive, filter)
	})

	t.Run("single filter returned as-is", func(t *testing.T) {
		original := LevelFilterOutBelow(DefaultLevels.Warn)
		filter := JoinLevelFilters(original)
		assert.Equal(t, original, filter)
	})

	t.Run("combines multiple filters with OR", func(t *testing.T) {
		filter1 := LevelFilterOut(DefaultLevels.Trace)
		filter2 := LevelFilterOut(DefaultLevels.Debug)
		combined := JoinLevelFilters(filter1, filter2)

		assert.False(t, combined.IsActive(ctx, DefaultLevels.Trace))
		assert.False(t, combined.IsActive(ctx, DefaultLevels.Debug))
		assert.True(t, combined.IsActive(ctx, DefaultLevels.Info))
	})
}

func TestLevelFilter_SetActive_OutOfRange(t *testing.T) {
	t.Run("out of range level is ignored", func(t *testing.T) {
		ctx := context.Background()
		filter := AllLevelsActive
		filter.SetActive(Level(-100), false)
		filter.SetActive(Level(100), false)
		// Should not panic or change other levels
		assert.True(t, filter.IsActive(ctx, DefaultLevels.Info))
	})
}

func TestLevelFilter_ActiveLevelNames(t *testing.T) {
	t.Run("all levels active", func(t *testing.T) {
		filter := AllLevelsActive
		names := filter.ActiveLevelNames(&DefaultLevels)
		assert.Contains(t, names, "TRACE")
		assert.Contains(t, names, "DEBUG")
		assert.Contains(t, names, "INFO")
		assert.Contains(t, names, "WARN")
		assert.Contains(t, names, "ERROR")
		assert.Contains(t, names, "FATAL")
	})

	t.Run("filtered levels", func(t *testing.T) {
		filter := LevelFilterOutBelow(DefaultLevels.Warn)
		names := filter.ActiveLevelNames(&DefaultLevels)
		assert.NotContains(t, names, "TRACE")
		assert.NotContains(t, names, "DEBUG")
		assert.NotContains(t, names, "INFO")
		assert.Contains(t, names, "WARN")
		assert.Contains(t, names, "ERROR")
		assert.Contains(t, names, "FATAL")
	})

	t.Run("all levels inactive", func(t *testing.T) {
		filter := AllLevelsInactive
		names := filter.ActiveLevelNames(&DefaultLevels)
		assert.Empty(t, names)
	})
}

func TestLevelFilter_InactiveLevelNames(t *testing.T) {
	t.Run("all levels active", func(t *testing.T) {
		filter := AllLevelsActive
		names := filter.InactiveLevelNames(&DefaultLevels)
		assert.Empty(t, names)
	})

	t.Run("filtered levels", func(t *testing.T) {
		filter := LevelFilterOutBelow(DefaultLevels.Warn)
		names := filter.InactiveLevelNames(&DefaultLevels)
		assert.Contains(t, names, "TRACE")
		assert.Contains(t, names, "DEBUG")
		assert.Contains(t, names, "INFO")
		assert.NotContains(t, names, "WARN")
		assert.NotContains(t, names, "ERROR")
		assert.NotContains(t, names, "FATAL")
	})

	t.Run("all levels inactive", func(t *testing.T) {
		filter := AllLevelsInactive
		names := filter.InactiveLevelNames(&DefaultLevels)
		assert.Len(t, names, 6)
	})
}

func TestNewLevelFilterOrNil(t *testing.T) {
	ctx := context.Background()

	t.Run("empty filters returns nil", func(t *testing.T) {
		filter := newLevelFilterOrNil(nil)
		assert.Nil(t, filter)

		filter = newLevelFilterOrNil([]LevelFilter{})
		assert.Nil(t, filter)
	})

	t.Run("single filter returns pointer", func(t *testing.T) {
		input := LevelFilterOutBelow(DefaultLevels.Warn)
		filter := newLevelFilterOrNil([]LevelFilter{input})
		assert.NotNil(t, filter)
		assert.Equal(t, input, *filter)
	})

	t.Run("multiple filters are combined", func(t *testing.T) {
		filters := []LevelFilter{
			LevelFilterOut(DefaultLevels.Trace),
			LevelFilterOut(DefaultLevels.Debug),
		}
		filter := newLevelFilterOrNil(filters)
		assert.NotNil(t, filter)
		assert.False(t, filter.IsActive(ctx, DefaultLevels.Trace))
		assert.False(t, filter.IsActive(ctx, DefaultLevels.Debug))
		assert.True(t, filter.IsActive(ctx, DefaultLevels.Info))
	})
}

func TestLevelFilterConstants(t *testing.T) {
	t.Run("AllLevelsActive is zero", func(t *testing.T) {
		assert.Equal(t, LevelFilter(0), AllLevelsActive)
	})

	t.Run("AllLevelsInactive is all bits set", func(t *testing.T) {
		assert.Equal(t, LevelFilter(0xFFFFFFFFFFFFFFFF), AllLevelsInactive)
	})
}

func TestLevelFilterImplementsLevelDecider(t *testing.T) {
	var _ LevelDecider = LevelFilter(0)
	var _ LevelDecider = AllLevelsActive
	var _ LevelDecider = AllLevelsInactive
}

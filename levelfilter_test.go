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

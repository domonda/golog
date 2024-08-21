package golog

import "context"

var _ LevelDecider = LevelFilter(0)

const (
	// AllLevelsActive allows all log levels.
	AllLevelsActive LevelFilter = 0

	// AllLevelsInactive disabled all log levels.
	AllLevelsInactive LevelFilter = 0xFFFFFFFFFFFFFFFF
)

// LevelFilter is a bit mask filter for levels 0..63,
// where a set bit filters out and zero allows a log level.
type LevelFilter uint64

func LevelFilterOut(level Level) LevelFilter {
	levelBitIndex := LevelFilter(level + 32)  // LevelMin is -32
	filter := LevelFilter(1) << levelBitIndex // Set bit with levelBitIndex
	return filter
}

func LevelFilterOutBelow(level Level) LevelFilter {
	levelBitIndex := LevelFilter(level + 32)  // LevelMin is -32
	filter := LevelFilter(1) << levelBitIndex // Set bit with levelBitIndex
	filter -= 1                               // Set all bits below levelBitIndex
	// fmt.Printf("LevelFilter: %b\n", filter)
	return filter
}

func LevelFilterOutAbove(level Level) LevelFilter {
	levelBitIndex := LevelFilter(level + 32)        // LevelMin is -32
	filter := LevelFilter(1) << (levelBitIndex + 1) // Set bit with levelBitIndex+1
	filter -= 1                                     // Set all bits below levelBitIndex+1 which includes levelBitIndex
	filter = ^filter                                // Inverse bits, now levelBitIndex and below are not set
	// fmt.Printf("LevelFilter: %b\n", filter)
	return filter
}

func LevelFilterOutAllOther(level Level) LevelFilter {
	levelBitIndex := LevelFilter(level + 32)  // LevelMin is -32
	filter := LevelFilter(1) << levelBitIndex // Set bit with levelBitIndex
	filter = ^filter                          // Inverse bits, now all bits except levelBitIndex are set
	// fmt.Printf("LevelFilter: %b\n", filter)
	return filter
}

// JoinLevelFilters returns a LevelFilter that filters
// out all levels that are filtered out by any of the passed filters.
func JoinLevelFilters(filters ...LevelFilter) LevelFilter {
	var combined LevelFilter
	for _, filter := range filters {
		combined |= filter
	}
	return combined
}

func newLevelFilterOrNil(filters []LevelFilter) *LevelFilter {
	if len(filters) == 0 {
		return nil
	}
	combined := JoinLevelFilters(filters...)
	return &combined
}

// IsActive returns if the passed level is active or filtered out.
// The context argument is ignored and only there to implement
// the LevelDecider interface.
func (f LevelFilter) IsActive(_ context.Context, level Level) bool {
	if level < LevelMin || level > LevelMax {
		return false
	}
	levelBitIndex := LevelFilter(level + 32) // LevelMin is -32
	levelBitMask := LevelFilter(1) << levelBitIndex
	// level is active when bit at levelBitIndex is zero
	return (f & levelBitMask) == 0
}

// IsInactive is the inverse of IsActive.
func (f LevelFilter) IsInactive(_ context.Context, level Level) bool {
	return !f.IsActive(nil, level)
}

func (f *LevelFilter) SetActive(level Level, active bool) {
	if level < LevelMin || level > LevelMax {
		return
	}
	levelBitIndex := LevelFilter(level + 32) // LevelMin is -32
	levelBitMask := LevelFilter(1) << levelBitIndex
	if active {
		// Don't filter out level by zeroing bit levelBitIndex
		*f &= ^levelBitMask
	} else {
		// Filter out level by setting bit levelBitIndex
		*f |= levelBitMask
	}
}

func (f *LevelFilter) ActiveLevelNames(levels *Levels) []string {
	var names []string
	for l := LevelMin; l <= LevelMax; l++ {
		if levels.HasName(l) && f.IsActive(nil, l) {
			names = append(names, levels.Name(l))
		}
	}
	return names
}

func (f *LevelFilter) InactiveLevelNames(levels *Levels) []string {
	var names []string
	for l := LevelMin; l <= LevelMax; l++ {
		if levels.HasName(l) && !f.IsActive(nil, l) {
			names = append(names, levels.Name(l))
		}
	}
	return names
}

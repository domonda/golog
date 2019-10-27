package golog

// NoFilter allows all log levels.
const NoFilter LevelFilter = 0

// LevelFilter is a bit mask filter for levels 0..63,
// where a set bit filters out and zero allows a log level.
type LevelFilter uint64

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

func LevelFilterCombine(filters ...LevelFilter) LevelFilter {
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
	combined := LevelFilterCombine(filters...)
	return &combined
}

func (f LevelFilter) IsActive(level Level) bool {
	if level < LevelMin || level > LevelMax {
		return false
	}
	levelBitIndex := LevelFilter(level + 32) // LevelMin is -32
	levelBitMask := LevelFilter(1) << levelBitIndex
	// level is active when bit at levelBitIndex is zero
	return (f & levelBitMask) == 0
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
		if levels.HasName(l) && f.IsActive(l) {
			names = append(names, levels.Name(l))
		}
	}
	return names
}

func (f *LevelFilter) InactiveLevelNames(levels *Levels) []string {
	var names []string
	for l := LevelMin; l <= LevelMax; l++ {
		if levels.HasName(l) && !f.IsActive(l) {
			names = append(names, levels.Name(l))
		}
	}
	return names
}

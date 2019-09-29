package golog

// LevelFilterNone allows all log levels.
const LevelFilterNone LevelFilter = 0

// LevelFilter is a bit mask filter for levels 0..63,
// where a set bit filters out and zero allows a log level.
type LevelFilter uint64

func LevelFilterAbove(level Level) LevelFilter {
	return ^((LevelFilter(1) << level) - 1)
}

func LevelFilterBelow(level Level) LevelFilter {
	return (LevelFilter(1) << level) - 1
}

func LevelFilterExclusive(level Level) LevelFilter {
	return ^(LevelFilter(1) << level)
}

func (f LevelFilter) IsActive(level Level) bool {
	return (level <= LevelMax) && (f&(LevelFilter(1)<<level) == 0)
}

func (f *LevelFilter) SetActive(level Level, active bool) {
	if level > LevelMax {
		return
	}
	switch active {
	case true:
		*f &^= (LevelFilter(1) << level)
	case false:
		*f |= (LevelFilter(1) << level)
	}
}

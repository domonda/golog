package golog

const LevelFilterNone LevelFilter = 0

// LevelFilter is a bit mask filter for levels 0..63
type LevelFilter uint64

func LevelFilterAbove(level Level) LevelFilter {
	return ^((LevelFilter(1) << level) - 1)
}

func (f LevelFilter) IsActive(level Level) bool {
	return f&(LevelFilter(1)<<level) == 0
}

func (f *LevelFilter) SetActive(level Level, active bool) {
	switch active {
	case true:
		*f &^= (LevelFilter(1) << level)
	case false:
		*f |= (LevelFilter(1) << level)
	}
}

package golog

const (
	LevelMin = 0
	LevelMax = 63

	LevelInvalid Level = 255
)

type Level uint8

func (l Level) Valid() bool {
	return l <= LevelMax
}

func (l Level) FilterAbove() LevelFilter {
	return LevelFilterAbove(l)
}

func (l Level) FilterExclusive() LevelFilter {
	return LevelFilterExclusive(l)
}

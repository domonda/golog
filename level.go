package golog

const (
	LevelMin = 0
	LevelMax = 63

	LevelInvalid Level = 255
)

var (
	LevelFatal Level = 0
	LevelError Level = 1
	LevelWarn  Level = 2
	LevelInfo  Level = 3
	LevelDebug Level = 4
	LevelTrace Level = 5
)

type Level uint8

func (l Level) Valid() bool {
	return l <= LevelMax
}

package golog

const (
	LevelMin     Level = -32
	LevelMax     Level = 31
	LevelInvalid Level = -128
)

type Level int8

func (l Level) Valid() bool {
	return l >= LevelMin && l <= LevelMax
}

func (l Level) FilterOutAbove() LevelFilter {
	return LevelFilterOutAbove(l)
}

func (l Level) FilterOutBelow() LevelFilter {
	return LevelFilterOutBelow(l)
}

func (l Level) FilterOutAllOther() LevelFilter {
	return LevelFilterOutAllOther(l)
}

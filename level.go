package golog

const (
	// LevelMin is the minimum valid log level value.
	LevelMin Level = -32
	// LevelMax is the maximum valid log level value.
	LevelMax Level = 31
	// LevelInvalid represents an invalid log level,
	// used as a sentinel value to disable certain features.
	LevelInvalid Level = -128
)

// Level represents a log severity level.
// Valid levels range from LevelMin (-32) to LevelMax (31).
// Lower values represent less severe levels (e.g., trace, debug),
// higher values represent more severe levels (e.g., warn, error, fatal).
type Level int8

// Valid returns true if the level is within the valid range [LevelMin, LevelMax].
func (l Level) Valid() bool {
	return l >= LevelMin && l <= LevelMax
}

// FilterOut returns a LevelFilter that filters out only this level.
func (l Level) FilterOut() LevelFilter {
	return LevelFilterOut(l)
}

// FilterOutAbove returns a LevelFilter that filters out all levels above this one.
func (l Level) FilterOutAbove() LevelFilter {
	return LevelFilterOutAbove(l)
}

// FilterOutBelow returns a LevelFilter that filters out all levels below this one.
func (l Level) FilterOutBelow() LevelFilter {
	return LevelFilterOutBelow(l)
}

// FilterOutAllOther returns a LevelFilter that filters out all levels except this one.
func (l Level) FilterOutAllOther() LevelFilter {
	return LevelFilterOutAllOther(l)
}

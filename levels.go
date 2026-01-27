package golog

import (
	"maps"
	"math"
	"slices"
	"strconv"
)

// DefaultLevels provides the standard log level configuration
// with Trace=-20, Debug=-10, Info=0, Warn=10, Error=20, Fatal=30.
var DefaultLevels = Levels{
	Trace: -20,
	Debug: -10,
	Info:  0,
	Warn:  10,
	Error: 20,
	Fatal: 30,
	Names: map[Level]string{
		-20: "TRACE",
		-10: "DEBUG",
		0:   "INFO",
		10:  "WARN",
		20:  "ERROR",
		30:  "FATAL",
	},
}

// Levels defines the log level values and their display names.
// Use DefaultLevels for the standard configuration.
type Levels struct {
	Trace Level
	Debug Level
	Info  Level
	Warn  Level
	Error Level
	Fatal Level
	Names map[Level]string
}

// Name returns the name of a level if available
// or the integer value of the level as string.
func (l *Levels) Name(level Level) string {
	if name, ok := l.Names[level]; ok {
		return name
	}
	return strconv.Itoa(int(level))
}

// HasName returns true if the level has a name defined in Names.
func (l *Levels) HasName(level Level) bool {
	_, has := l.Names[level]
	return has
}

// FatalName returns the display name for the Fatal level.
func (l *Levels) FatalName() string {
	return l.Name(l.Fatal)
}

// ErrorName returns the display name for the Error level.
func (l *Levels) ErrorName() string {
	return l.Name(l.Error)
}

// WarnName returns the display name for the Warn level.
func (l *Levels) WarnName() string {
	return l.Name(l.Warn)
}

// InfoName returns the display name for the Info level.
func (l *Levels) InfoName() string {
	return l.Name(l.Info)
}

// DebugName returns the display name for the Debug level.
func (l *Levels) DebugName() string {
	return l.Name(l.Debug)
}

// TraceName returns the display name for the Trace level.
func (l *Levels) TraceName() string {
	return l.Name(l.Trace)
}

// NamesSorted returns the names of the levels sorted by level value.
func (l *Levels) NamesSorted() []string {
	names := make([]string, len(l.Names))
	for i, level := range slices.Sorted(maps.Keys(l.Names)) {
		names[i] = l.Names[level]
	}
	return names
}

// LevelOfName returns the [Level] with a give name.
// If name is formatted as an integer and within [LevelMin..LevelMax]
// then a level with that integer value will be returned.
// This is the inverse operation to what Levels.Name(unnamedLevel) returns.
// If there is no level with name or valid integer value
// then [LevelInvalid] will be returned.
func (l *Levels) LevelOfName(name string) Level {
	for level, levelName := range l.Names {
		if name == levelName {
			return level
		}
	}
	if i, err := strconv.Atoi(name); err == nil && i >= int(LevelMin) && i <= int(LevelMax) {
		return Level(i) //#nosec G115 -- integer conversion OK: LevelMin is -32
	}
	return LevelInvalid
}

// LevelOfNameOrDefault returns the [Level] with a given name,
// or defaultLevel if the name is not in Levels.Names.
func (l *Levels) LevelOfNameOrDefault(name string, defaultLevel Level) Level {
	for level, levelName := range l.Names {
		if name == levelName {
			return level
		}
	}
	return defaultLevel
}

// NameLenRange returns the minimum and maximum length of level names.
func (l *Levels) NameLenRange() (minLen, maxLen int) {
	if len(l.Names) == 0 {
		return 0, 0
	}
	minLen = math.MaxInt
	for _, name := range l.Names {
		l := len(name)
		minLen = min(minLen, l)
		maxLen = max(maxLen, l)
	}
	return minLen, maxLen
}

// CopyWithLeftPaddedNames returns a copy of Levels with all names
// left-padded with spaces to match the longest name length.
func (l *Levels) CopyWithLeftPaddedNames() *Levels {
	padded := *l
	padded.Names = make(map[Level]string, len(l.Names))
	_, maxLen := l.NameLenRange()
	for level, name := range l.Names {
		padded.Names[level] = name
		for len(padded.Names[level]) < maxLen {
			padded.Names[level] = " " + padded.Names[level]
		}
	}
	return &padded
}

// CopyWithRightPaddedNames returns a copy of Levels with all names
// right-padded with spaces to match the longest name length.
func (l *Levels) CopyWithRightPaddedNames() *Levels {
	padded := *l
	padded.Names = make(map[Level]string, len(l.Names))
	_, maxLen := l.NameLenRange()
	for level, name := range l.Names {
		padded.Names[level] = name
		for len(padded.Names[level]) < maxLen {
			padded.Names[level] += " "
		}
	}
	return &padded
}

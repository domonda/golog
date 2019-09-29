package golog

import "strconv"

var DefaultLevels = &Levels{
	Fatal: 0,
	Error: 1,
	Warn:  2,
	Info:  3,
	Debug: 4,
	Trace: 5,
	Names: []string{
		"FATAL", // 0
		"ERROR", // 1
		"WARN",  // 2
		"INFO",  // 3
		"DEBUG", // 4
		"TRACE", // 5
	},
}

type Levels struct {
	Fatal Level
	Error Level
	Warn  Level
	Info  Level
	Debug Level
	Trace Level
	Names []string
}

func (l *Levels) Name(level Level) string {
	if int(level) >= len(l.Names) {
		return strconv.Itoa(int(level))
	}
	return l.Names[level]
}

func (l *Levels) FatalName() string {
	return l.Name(l.Fatal)
}

func (l *Levels) ErrorName() string {
	return l.Name(l.Error)
}

func (l *Levels) WarnName() string {
	return l.Name(l.Warn)
}

func (l *Levels) InfoName() string {
	return l.Name(l.Info)
}

func (l *Levels) DebugName() string {
	return l.Name(l.Debug)
}

func (l *Levels) TraceName() string {
	return l.Name(l.Trace)
}

func (l *Levels) LevelOfName(name string) Level {
	for i := range l.Names {
		if l.Names[i] == name {
			return Level(i)
		}
	}
	if i, err := strconv.Atoi(name); err == nil && i >= LevelMin && i <= LevelMax {
		return Level(i)
	}
	return LevelInvalid
}

func (l *Levels) NameLenRange() (min, max int) {
	for _, name := range l.Names {
		nameLen := len(name)
		if nameLen < min {
			min = nameLen
		}
		if nameLen > max {
			max = nameLen
		}
	}
	return min, max
}

func (l *Levels) CopyWithLeftPaddedNames() *Levels {
	padded := *l
	padded.Names = make([]string, len(l.Names))
	_, maxLen := l.NameLenRange()
	for i, name := range l.Names {
		padded.Names[i] = name
		for len(padded.Names[i]) < maxLen {
			padded.Names[i] = " " + padded.Names[i]
		}
	}
	return &padded
}

func (l *Levels) CopyWithRightPaddedNames() *Levels {
	padded := *l
	padded.Names = make([]string, len(l.Names))
	_, maxLen := l.NameLenRange()
	for i, name := range l.Names {
		padded.Names[i] = name
		for len(padded.Names[i]) < maxLen {
			padded.Names[i] += " "
		}
	}
	return &padded
}

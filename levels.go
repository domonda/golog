package golog

import "strconv"

var DefaultLevels = &Levels{
	Fatal: 0,
	Error: 1,
	Warn:  2,
	Info:  3,
	Debug: 4,
	Trace: 5,
	Names: map[Level]string{
		0: "FATAL",
		1: "ERROR",
		2: "WARN",
		3: "INFO",
		4: "DEBUG",
		5: "TRACE",
	},
}

type Levels struct {
	Fatal Level
	Error Level
	Warn  Level
	Info  Level
	Debug Level
	Trace Level
	Names map[Level]string
}

func (l *Levels) Name(level Level) string {
	if name, ok := l.Names[level]; ok {
		return name
	}
	return strconv.Itoa(int(level))
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
	for level, levelName := range l.Names {
		if name == levelName {
			return level
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

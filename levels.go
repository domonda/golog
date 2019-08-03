package golog

import "strconv"

var DefaultLevels = Levels{
	"FATAL", // 0
	"ERROR", // 1
	"WARN",  // 2
	"INFO",  // 3
	"DEBUG", // 4
	"TRACE", // 5
}

type Levels []string

func (l Levels) Name(level Level) string {
	if int(level) >= len(l) {
		return strconv.Itoa(int(level))
	}
	return l[level]
}

func (l Levels) LevelOfName(name string) Level {
	for i := range l {
		if l[i] == name {
			return Level(i)
		}
	}
	if i, err := strconv.Atoi(name); err == nil && i >= LevelMin && i <= LevelMax {
		return Level(i)
	}
	return LevelInvalid
}

func (l Levels) MaxNameLen() int {
	max := 0
	for _, name := range l {
		if len(name) > max {
			max = len(name)
		}
	}
	return max
}

func (l Levels) CopyWithPaddedNames() Levels {
	p := make(Levels, len(l))
	maxLen := l.MaxNameLen()
	for i := range l {
		p[i] = l[i]
		for len(p[i]) < maxLen {
			p[i] += " "
		}
	}
	return p
}

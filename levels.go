package golog

import "strconv"

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

func (l Levels) NameLenRange() (min, max int) {
	for _, name := range l {
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

func (l Levels) CopyWithLeftPaddedNames() Levels {
	p := make(Levels, len(l))
	_, maxLen := l.NameLenRange()
	for i := range l {
		p[i] = l[i]
		for len(p[i]) < maxLen {
			p[i] = " " + p[i]
		}
	}
	return p
}

func (l Levels) CopyWithRightPaddedNames() Levels {
	p := make(Levels, len(l))
	_, maxLen := l.NameLenRange()
	for i := range l {
		p[i] = l[i]
		for len(p[i]) < maxLen {
			p[i] += " "
		}
	}
	return p
}

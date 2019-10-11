package golog

import (
	"strings"
	"sync"
	"time"
)

type MultiFormatter []Formatter

var multiFormatterPool sync.Pool

func newMultiFormatter(l int) MultiFormatter {
	if f, ok := multiFormatterPool.Get().(MultiFormatter); ok && l <= cap(f) {
		return f[:l]
	}

	return make(MultiFormatter, l)
}

func (mf MultiFormatter) Clone() Formatter {
	clone := newMultiFormatter(len(mf))
	for i, f := range mf {
		clone[i] = f.Clone()
	}
	return clone
}

func (mf MultiFormatter) WriteText(t time.Time, levels *Levels, level Level, text string) {
	for _, f := range mf {
		f.WriteText(t, levels, level, text)
	}
}

func (mf MultiFormatter) FlushAndFree() {
	for i, f := range mf {
		f.FlushAndFree()
		mf[i] = nil
	}
	multiFormatterPool.Put(mf)
}

// String is here only for debugging
func (mf MultiFormatter) String() string {
	var b strings.Builder
	for i, f := range mf {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(f.String())
	}
	return b.String()
}

func (mf MultiFormatter) WriteKey(key string) {
	for _, f := range mf {
		f.WriteKey(key)
	}
}

func (mf MultiFormatter) WriteSliceKey(key string) {
	for _, f := range mf {
		f.WriteSliceKey(key)
	}
}

func (mf MultiFormatter) WriteSliceEnd() {
	for _, f := range mf {
		f.WriteSliceEnd()
	}
}

func (mf MultiFormatter) WriteNil() {
	for _, f := range mf {
		f.WriteNil()
	}
}

func (mf MultiFormatter) WriteBool(val bool) {
	for _, f := range mf {
		f.WriteBool(val)
	}
}

func (mf MultiFormatter) WriteInt(val int64) {
	for _, f := range mf {
		f.WriteInt(val)
	}
}

func (mf MultiFormatter) WriteUint(val uint64) {
	for _, f := range mf {
		f.WriteUint(val)
	}
}

func (mf MultiFormatter) WriteFloat(val float64) {
	for _, f := range mf {
		f.WriteFloat(val)
	}
}

func (mf MultiFormatter) WriteString(val string) {
	for _, f := range mf {
		f.WriteString(val)
	}
}

func (mf MultiFormatter) WriteError(val error) {
	for _, f := range mf {
		f.WriteError(val)
	}
}

func (mf MultiFormatter) WriteUUID(val [16]byte) {
	for _, f := range mf {
		f.WriteUUID(val)
	}
}

func (mf MultiFormatter) WriteJSON(val []byte) {
	for _, f := range mf {
		f.WriteJSON(val)
	}
}

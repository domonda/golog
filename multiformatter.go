package golog

import (
	"sync"
	"time"
)

type MultiFormatter []Formatter

var multiFormatterPool sync.Pool

func NewMultiFormatter(count int) MultiFormatter {
	if f, ok := multiFormatterPool.Get().(MultiFormatter); ok {
		if count > len(f) {
			return make(MultiFormatter, count)
		}
		return f[:count]
	}

	return make(MultiFormatter, count)
}

func (mf MultiFormatter) Begin(t time.Time, level Level, msg string, data []byte) {
	for _, f := range mf {
		f.Begin(t, level, msg, data)
	}
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

func (mf MultiFormatter) WriteUUID(val [16]byte) {
	for _, f := range mf {
		f.WriteUUID(val)
	}
}

func (mf MultiFormatter) Flush() {
	for _, f := range mf {
		f.Flush()
	}
	multiFormatterPool.Put(mf)
}

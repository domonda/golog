package golog

import (
	"strings"
	"sync"
	"time"
)

type MultiWriter []Writer

var multiWriterPool sync.Pool

func newMultiWriter(l int) MultiWriter {
	if f, ok := multiWriterPool.Get().(MultiWriter); ok && l <= cap(f) {
		return f[:l]
	}

	return make(MultiWriter, l)
}

func (m MultiWriter) Clone(level Level) Writer {
	clone := newMultiWriter(len(m))
	for i, f := range m {
		clone[i] = f.Clone(level)
	}
	return clone
}

func (m MultiWriter) BeginMessage(t time.Time, levels *Levels, level Level, prefix, text string) {
	for _, f := range m {
		f.BeginMessage(t, levels, level, prefix, text)
	}
}

func (m MultiWriter) FinishMessage() {
	for i, f := range m {
		f.FinishMessage()
		m[i] = nil
	}
	multiWriterPool.Put(m)
}

func (m MultiWriter) FlushUnderlying() {
	for _, f := range m {
		f.FlushUnderlying()
	}
}

func (m MultiWriter) String() string {
	var b strings.Builder
	for i, f := range m {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(f.String())
	}
	return b.String()
}

func (m MultiWriter) WriteKey(key string) {
	for _, f := range m {
		f.WriteKey(key)
	}
}

func (m MultiWriter) WriteSliceKey(key string) {
	for _, f := range m {
		f.WriteSliceKey(key)
	}
}

func (m MultiWriter) WriteSliceEnd() {
	for _, f := range m {
		f.WriteSliceEnd()
	}
}

func (m MultiWriter) WriteNil() {
	for _, f := range m {
		f.WriteNil()
	}
}

func (m MultiWriter) WriteBool(val bool) {
	for _, f := range m {
		f.WriteBool(val)
	}
}

func (m MultiWriter) WriteInt(val int64) {
	for _, f := range m {
		f.WriteInt(val)
	}
}

func (m MultiWriter) WriteUint(val uint64) {
	for _, f := range m {
		f.WriteUint(val)
	}
}

func (m MultiWriter) WriteFloat(val float64) {
	for _, f := range m {
		f.WriteFloat(val)
	}
}

func (m MultiWriter) WriteString(val string) {
	for _, f := range m {
		f.WriteString(val)
	}
}

func (m MultiWriter) WriteError(val error) {
	for _, f := range m {
		f.WriteError(val)
	}
}

func (m MultiWriter) WriteUUID(val [16]byte) {
	for _, f := range m {
		f.WriteUUID(val)
	}
}

func (m MultiWriter) WriteJSON(val []byte) {
	for _, f := range m {
		f.WriteJSON(val)
	}
}

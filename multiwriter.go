package golog

import (
	"strings"
	"sync"
	"time"
)

type MultiWriter []Writer

var multiWriterPool sync.Pool

func getMultiWriter(numWriters int) MultiWriter {
	if recycled, ok := multiWriterPool.Get().(MultiWriter); ok && numWriters <= cap(recycled) {
		return recycled[:numWriters]
	}
	return make(MultiWriter, numWriters)
}

func (m MultiWriter) BeginMessage(logger *Logger, t time.Time, level Level, text string) Writer {
	next := getMultiWriter(len(m))
	for i, w := range m {
		next[i] = w.BeginMessage(logger, t, level, text)
	}
	return next
}

func (m MultiWriter) CommitMessage() {
	for i, f := range m {
		f.CommitMessage()
		m[i] = nil
	}
	multiWriterPool.Put(m)
}

func (m MultiWriter) FlushUnderlying() {
	for _, f := range m {
		if f != nil {
			f.FlushUnderlying()
		}
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

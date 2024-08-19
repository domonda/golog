package golog

import (
	"context"
	"strings"
	"sync"
	"time"
)

// MultiWriter is a Writer that writes to multiple other Writers
type MultiWriter []Writer

// combineWriters flattens all passed Writers that are
// themselves MultiWriters into a single MultiWriter.
func combineWriters(writer Writer, additionalWriters ...Writer) Writer {
	if len(additionalWriters) == 0 {
		return writer
	}
	mw := make(MultiWriter, 0, 1+len(additionalWriters))
	if writer != nil {
		if mw2, ok := writer.(MultiWriter); ok {
			mw = append(mw, mw2...)
		} else {
			mw = append(mw, writer)
		}
	}
	for _, w := range additionalWriters {
		if w == nil {
			continue
		}
		if mw2, ok := w.(MultiWriter); ok {
			mw = append(mw, mw2...)
		} else {
			mw = append(mw, w)
		}
	}
	return mw
}

var multiWriterPool sync.Pool

func getMultiWriter(numWriters int) MultiWriter {
	if recycled, ok := multiWriterPool.Get().(MultiWriter); ok && numWriters <= cap(recycled) {
		return recycled[:numWriters]
	}
	return make(MultiWriter, numWriters)
}

func (m MultiWriter) BeginMessage(ctx context.Context, logger *Logger, t time.Time, level Level, text string) Writer {
	next := getMultiWriter(len(m))
	for i, w := range m {
		next[i] = w.BeginMessage(ctx, logger, t, level, text)
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

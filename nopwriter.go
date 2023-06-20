package golog

import (
	"context"
	"time"
)

// NopWriter is a Writer that does nothing (no operation)
const NopWriter nopWriter = 0

type nopWriter int

func (w nopWriter) BeginMessage(_ context.Context, logger *Logger, t time.Time, level Level, text string) Writer {
	return w
}

func (nopWriter) CommitMessage() {}

func (nopWriter) FlushUnderlying() {}

func (nopWriter) String() string {
	return "NopWriter"
}

func (nopWriter) WriteKey(key string) {}

func (nopWriter) WriteSliceKey(key string) {}

func (nopWriter) WriteSliceEnd() {}

func (nopWriter) WriteNil() {}

func (nopWriter) WriteBool(val bool) {}

func (nopWriter) WriteInt(val int64) {}

func (nopWriter) WriteUint(val uint64) {}

func (nopWriter) WriteFloat(val float64) {}

func (nopWriter) WriteString(val string) {}

func (nopWriter) WriteError(val error) {}

func (nopWriter) WriteUUID(val [16]byte) {}

func (nopWriter) WriteJSON(val []byte) {}

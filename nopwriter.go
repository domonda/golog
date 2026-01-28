package golog

import (
	"context"
	"time"
)

var (
	_ Writer       = new(NopWriter)
	_ WriterConfig = new(NopWriterConfig)
)

type NopWriterConfig string

func (c NopWriterConfig) WriterForNewMessage(context.Context, Level) Writer {
	if c == "" {
		return NopWriter("NopWriter")
	}
	return NopWriter(c)
}

func (c NopWriterConfig) FlushUnderlying() {}

///////////////////////////////////////////////////////////////////////////////

type NopWriter string

func (NopWriter) BeginMessage(config Config, timestamp time.Time, level Level, prefix, text string) {}

func (NopWriter) CommitMessage() {}

func (w NopWriter) String() string {
	return string(w)
}

func (NopWriter) WriteKey(key string) {}

func (NopWriter) WriteSliceKey(key string) {}

func (NopWriter) WriteSliceEnd() {}

func (NopWriter) WriteNil() {}

func (NopWriter) WriteBool(val bool) {}

func (NopWriter) WriteInt(val int64) {}

func (NopWriter) WriteUint(val uint64) {}

func (NopWriter) WriteFloat(val float64) {}

func (NopWriter) WriteString(val string) {}

func (NopWriter) WriteError(val error) {}

func (NopWriter) WriteTime(val time.Time) {}

func (NopWriter) WriteUUID(val [16]byte) {}

func (NopWriter) WriteJSON(val []byte) {}

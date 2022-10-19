package golog

import (
	"time"
)

// Formatter gets implemented to format log messages
// in a certain message format.
// FlushAndFree must be called before a formatter
// can be re-used for a new message.
type Formatter interface {
	// Clone the formatter for a new message with the passed log level
	Clone(level Level) Formatter

	// String is here only for debugging
	String() string

	// WriteText of a new log message
	WriteText(t time.Time, levels *Levels, level Level, prefix, text string)

	WriteKey(key string)
	WriteSliceKey(key string)
	WriteSliceEnd()

	WriteNil()
	WriteBool(val bool)
	WriteInt(val int64)
	WriteUint(val uint64)
	WriteFloat(val float64)
	WriteString(val string)
	WriteError(val error)
	WriteUUID(val [16]byte)
	WriteJSON(val []byte)
	// WritePtr(val uintptr)

	// FlushAndFree flushes the current log message
	// to the underlying writer and frees any resources
	// to make the formatter ready for a new message.
	FlushAndFree()

	// FlushUnderlying flushes underlying log writing
	// streams to make sure all messages have been
	// saved or transmitted.
	FlushUnderlying()
}

func flushUnderlying(writer any) {
	switch x := writer.(type) {
	case interface{ Sync() error }:
		x.Sync() //#nosec G104
	}
}

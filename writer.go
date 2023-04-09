package golog

import (
	"time"
)

// Writer implementations write log messages
// in a certain message format.
// FinishMessage must be called before a formatter
// can be re-used for a new message.
type Writer interface {
	// Clone the formatter for a new message with the passed log level
	Clone(level Level) Writer

	// String is here only for debugging
	String() string

	// BeginMessage with a level and text
	BeginMessage(t time.Time, levels *Levels, level Level, prefix, text string)

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

	// FinishMessage flushes the current log message
	// to the underlying writer and frees any resources
	// to make the formatter ready for a new message.
	FinishMessage()

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

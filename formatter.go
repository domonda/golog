package golog

import (
	"time"
)

type Formatter interface {
	Clone(level Level) Formatter
	WriteText(t time.Time, levels *Levels, level Level, prefix, text string)
	FlushAndFree()
	FlushUnderlying()

	// String is here only for debugging
	String() string

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
}

func flushUnderlying(writer any) {
	switch x := writer.(type) {
	case interface{ Sync() error }:
		x.Sync() //#nosec G104
	}
}

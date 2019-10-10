package golog

import (
	"time"
)

type Formatter interface {
	NewChild() Formatter
	WriteMsg(t time.Time, levels *Levels, level Level, msg string)
	FlushAndFree()

	// String is here only for debugging
	String() string

	WriteKey(key string)
	WriteSliceKey(key string)
	WriteSliceEnd()

	WriteNil()
	WriteBool(val bool)
	WriteInt(val int64)
	WriteUint(val uint64)
	// WritePtr(val uintptr)
	WriteFloat(val float64)
	WriteString(val string)
	WriteError(val error)
	WriteUUID(val [16]byte)
	WriteJSON(val []byte)
}

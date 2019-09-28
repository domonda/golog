package golog

import (
	"time"
)

type Formatter interface {
	NewChild() Formatter
	WriteMsg(t time.Time, level Level, msg string)
	FlushAndFree()

	// String is here only for debugging
	String() string

	WriteKey(key string)
	WriteSliceKey(key string)
	WriteSliceEnd()

	WriteBool(val bool)
	WriteInt(val int64)
	WriteUint(val uint64)
	WriteFloat(val float64)
	WriteString(val string)
	WriteUUID(val [16]byte)
}

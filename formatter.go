package golog

import (
	"io"
	"time"
)

type NewFormatterFunc func(writer io.Writer, format *Format) Formatter

type Formatter interface {
	WriteIntro(t time.Time, level Level, msg string, data []byte)
	WriteOutro()
	Flush()

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

type Format struct {
	TimestampKey    string
	TimestampFormat string

	LevelKey string // can be empty
	Levels   Levels // can be empty

	MessageKey string
}

package golog

import (
	"time"
)

// Writer implementations write log messages
// in a certain message format to some underlying
// data stream.`
//
// CommitMessage must be called before the Writer
// can be re-used for a new message.
type Writer interface {
	// BeginMessage begins writing a new message
	// that must be finished with CommitMessage.
	//
	// This method is only called for log levels that are active at the logger,
	// so implementations don't have to check the passed logger to decide
	// if they should log the passed level.
	//
	// The config is passed to give access to other data that might be needed
	// for message formatting level names.
	BeginMessage(config Config, t time.Time, level Level, prefix, text string)

	WriteKey(string)
	WriteSliceKey(string)
	WriteSliceEnd()

	WriteNil()
	WriteBool(bool)
	WriteInt(int64)
	WriteUint(uint64)
	WriteFloat(float64)
	WriteString(string)
	WriteError(error)
	WriteUUID([16]byte)
	WriteJSON([]byte)
	// WritePtr(uintptr)

	// CommitMessage flushes the current log message
	// to the underlying writer and frees any resources
	// to make the Writer ready for a new message.
	CommitMessage()

	// String is here only for debugging
	String() string
}

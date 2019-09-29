package golog

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/domonda/go-encjson"
)

var jsonFormatterPool sync.Pool

type JSONFormatter struct {
	parent *JSONFormatter
	writer io.Writer
	format *Format
	buf    []byte
}

func NewJSONFormatter(writer io.Writer, format *Format) *JSONFormatter {
	return &JSONFormatter{
		writer: writer,
		format: format,
		buf:    make([]byte, 0, 1024),
	}
}

func (f *JSONFormatter) NewChild() Formatter {
	child, ok := jsonFormatterPool.Get().(*JSONFormatter)
	if ok {
		child.writer = f.writer
		child.format = f.format
	} else {
		child = NewJSONFormatter(f.writer, f.format)
	}
	child.parent = f
	return child
}

func (f *JSONFormatter) WriteMsg(t time.Time, level Level, msg string) {
	f.buf = append(f.buf, '{')

	if f.format.TimestampKey != "" {
		f.buf = encjson.AppendKey(f.buf, f.format.TimestampKey)
		f.buf = encjson.AppendTime(f.buf, t, f.format.TimestampFormat)
	}

	if f.format.LevelKey != "" {
		f.buf = encjson.AppendKey(f.buf, f.format.LevelKey)
		f.buf = encjson.AppendString(f.buf, f.format.Levels.Name(level))
	}

	if f.format.MessageKey != "" && msg != "" {
		f.buf = encjson.AppendKey(f.buf, f.format.MessageKey)
		f.buf = encjson.AppendString(f.buf, msg)
	}

	f.buf = f.appendParent(f.buf)
}

func (f *JSONFormatter) appendParent(buf []byte) []byte {
	if f.parent != nil {
		buf = f.parent.appendParent(buf)
		if len(f.parent.buf) > 0 {
			buf = append(buf, ',')
			buf = append(buf, f.parent.buf...)
		}
	}
	return buf
}

func (f *JSONFormatter) FlushAndFree() {
	// Flush
	f.buf = append(f.buf, '}', '\n')
	_, err := f.writer.Write(f.buf)
	if err != nil && ErrorHandler != nil {
		ErrorHandler(fmt.Errorf("golog.JSONFormatter error: %w", err))
	}

	// Free
	f.parent = nil
	f.writer = nil
	f.format = nil
	f.buf = f.buf[:0]
	jsonFormatterPool.Put(f)
}

// String is here only for debugging
func (f *JSONFormatter) String() string {
	return string(f.buf)
}

func (f *JSONFormatter) WriteKey(key string) {
	f.buf = encjson.AppendKey(f.buf, key)
}

func (f *JSONFormatter) WriteSliceKey(key string) {
	f.buf = encjson.AppendKey(f.buf, key)
	f.buf = encjson.AppendArrayStart(f.buf)
}

func (f *JSONFormatter) WriteSliceEnd() {
	f.buf = encjson.AppendArrayEnd(f.buf)
}

func (f *JSONFormatter) WriteBool(val bool) {
	f.buf = encjson.AppendBool(f.buf, val)
}

func (f *JSONFormatter) WriteInt(val int64) {
	f.buf = encjson.AppendInt(f.buf, val)
}

func (f *JSONFormatter) WriteUint(val uint64) {
	f.buf = encjson.AppendUint(f.buf, val)
}

func (f *JSONFormatter) WriteFloat(val float64) {
	f.buf = encjson.AppendFloat(f.buf, val)
}

func (f *JSONFormatter) WriteString(val string) {
	f.buf = encjson.AppendString(f.buf, val)
}

func (f *JSONFormatter) WriteUUID(val [16]byte) {
	f.buf = encjson.AppendUUID(f.buf, val)
}

func (f *JSONFormatter) WriteJSON(val []byte) {
	f.buf = append(f.buf, val...)
}

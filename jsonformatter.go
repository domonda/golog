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

func (f *JSONFormatter) Clone(level Level) Formatter {
	if clone, ok := jsonFormatterPool.Get().(*JSONFormatter); ok {
		clone.writer = f.writer
		clone.format = f.format
		return clone
	}
	return NewJSONFormatter(f.writer, f.format)
}

func (f *JSONFormatter) WriteText(t time.Time, levels *Levels, level Level, prefix, text string) {
	f.buf = append(f.buf, '{')

	if f.format.TimestampKey != "" {
		f.buf = encjson.AppendKey(f.buf, f.format.TimestampKey)
		f.buf = encjson.AppendTime(f.buf, t, f.format.TimestampFormat)
	}

	if f.format.LevelKey != "" {
		f.buf = encjson.AppendKey(f.buf, f.format.LevelKey)
		f.buf = encjson.AppendString(f.buf, levels.Name(level))
	}

	if f.format.MessageKey != "" && text != "" {
		f.buf = encjson.AppendKey(f.buf, f.format.MessageKey)
		f.buf = encjson.AppendString(f.buf, prefix+text)
	}
}

func (f *JSONFormatter) FlushAndFree() {
	// Flush f.buf
	if len(f.buf) > 0 {
		_, err := f.writer.Write(append(f.buf, '}', ',', '\n'))
		if err != nil && ErrorHandler != nil {
			ErrorHandler(fmt.Errorf("golog.JSONFormatter error: %w", err))
		}
	}

	// Free
	f.writer = nil
	f.format = nil
	f.buf = f.buf[:0]
	jsonFormatterPool.Put(f)
}

func (f *JSONFormatter) FlushUnderlying() {
	flushUnderlying(f.writer)
}

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

func (f *JSONFormatter) WriteNil() {
	f.buf = encjson.AppendNull(f.buf)
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

func (f *JSONFormatter) WriteError(val error) {
	f.buf = encjson.AppendString(f.buf, val.Error())
}

func (f *JSONFormatter) WriteUUID(val [16]byte) {
	f.buf = encjson.AppendUUID(f.buf, val)
}

func (f *JSONFormatter) WriteJSON(val []byte) {
	if len(val) == 0 {
		val = []byte("null")
	}
	f.buf = append(f.buf, val...)
}

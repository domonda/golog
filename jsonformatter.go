package golog

import (
	"io"
	"sync"
	"time"

	"github.com/domonda/go-encjson"
)

var jsonFormatterPool sync.Pool

type jsonFormatter struct {
	writer io.Writer
	format *Format
	buf    []byte
}

func NewJSONFormatter(writer io.Writer, format *Format) Formatter {
	if f, ok := jsonFormatterPool.Get().(*jsonFormatter); ok {
		f.writer = writer
		f.format = format
		f.buf = f.buf[:0]
		return f
	}

	return &jsonFormatter{
		writer: writer,
		format: format,
		buf:    make([]byte, 0, 1024),
	}
}

func (f *jsonFormatter) Begin(t time.Time, level Level, msg string, data []byte) {
	f.buf = append(f.buf, '{')

	f.buf = encjson.AppendKey(f.buf, f.format.TimestampKey)
	f.buf = encjson.AppendTime(f.buf, t, f.format.TimestampFormat)

	if f.format.LevelKey != "" {
		f.buf = encjson.AppendKey(f.buf, f.format.LevelKey)
		f.buf = encjson.AppendString(f.buf, f.format.Levels.Name(level))
	}

	f.buf = encjson.AppendKey(f.buf, f.format.MessageKey)
	f.buf = encjson.AppendString(f.buf, msg)

	// Write data from super logger
	f.buf = append(f.buf, data...)
}

func (f *jsonFormatter) WriteKey(key string) {
	f.buf = encjson.AppendKey(f.buf, key)
}

func (f *jsonFormatter) WriteSliceKey(key string) {
	f.buf = encjson.AppendKey(f.buf, key)
	f.buf = encjson.AppendArrayStart(f.buf)
}

func (f *jsonFormatter) WriteSliceEnd() {
	f.buf = encjson.AppendArrayEnd(f.buf)
}

func (f *jsonFormatter) WriteBool(val bool) {
	f.buf = encjson.AppendBool(f.buf, val)
}

func (f *jsonFormatter) WriteInt(val int64) {
	f.buf = encjson.AppendInt(f.buf, val)
}

func (f *jsonFormatter) WriteUint(val uint64) {
	f.buf = encjson.AppendUint(f.buf, val)
}

func (f *jsonFormatter) WriteFloat(val float64) {
	f.buf = encjson.AppendFloat(f.buf, val)
}

func (f *jsonFormatter) WriteString(val string) {
	f.buf = encjson.AppendString(f.buf, val)
}

func (f *jsonFormatter) WriteUUID(val [16]byte) {
	f.buf = encjson.AppendUUID(f.buf, val)
}

func (f *jsonFormatter) Flush() {
	f.buf = append(f.buf, '}', '\n')
	f.writer.Write(f.buf)

	f.writer = nil
	f.format = nil
	textFormatterPool.Put(f)
}

package golog

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/domonda/go-encjson"
)

var jsonWriterPool sync.Pool

type JSONWriter struct {
	writer io.Writer
	format *Format
	buf    []byte
}

func NewJSONWriter(writer io.Writer, format *Format) *JSONWriter {
	return &JSONWriter{
		writer: writer,
		format: format,
		buf:    make([]byte, 0, 1024),
	}
}

func (w *JSONWriter) Clone(level Level) Writer {
	if clone, ok := jsonWriterPool.Get().(*JSONWriter); ok {
		clone.writer = w.writer
		clone.format = w.format
		return clone
	}
	return NewJSONWriter(w.writer, w.format)
}

func (w *JSONWriter) BeginMessage(t time.Time, levels *Levels, level Level, prefix, text string) {
	w.buf = append(w.buf, '{')

	if w.format.TimestampKey != "" {
		w.buf = encjson.AppendKey(w.buf, w.format.TimestampKey)
		w.buf = encjson.AppendTime(w.buf, t, w.format.TimestampFormat)
	}

	if w.format.LevelKey != "" {
		w.buf = encjson.AppendKey(w.buf, w.format.LevelKey)
		w.buf = encjson.AppendString(w.buf, levels.Name(level))
	}

	if w.format.MessageKey != "" && text != "" {
		w.buf = encjson.AppendKey(w.buf, w.format.MessageKey)
		w.buf = encjson.AppendString(w.buf, prefix+text)
	}
}

func (w *JSONWriter) FinishMessage() {
	// Flush f.buf
	if len(w.buf) > 0 {
		_, err := w.writer.Write(append(w.buf, '}', ',', '\n'))
		if err != nil && ErrorHandler != nil {
			ErrorHandler(fmt.Errorf("golog.JSONWriter error: %w", err))
		}
	}

	// Free
	w.writer = nil
	w.format = nil
	w.buf = w.buf[:0]
	jsonWriterPool.Put(w)
}

func (w *JSONWriter) FlushUnderlying() {
	flushUnderlying(w.writer)
}

func (w *JSONWriter) String() string {
	return string(w.buf)
}

func (w *JSONWriter) WriteKey(key string) {
	w.buf = encjson.AppendKey(w.buf, key)
}

func (w *JSONWriter) WriteSliceKey(key string) {
	w.buf = encjson.AppendKey(w.buf, key)
	w.buf = encjson.AppendArrayStart(w.buf)
}

func (w *JSONWriter) WriteSliceEnd() {
	w.buf = encjson.AppendArrayEnd(w.buf)
}

func (w *JSONWriter) WriteNil() {
	w.buf = encjson.AppendNull(w.buf)
}

func (w *JSONWriter) WriteBool(val bool) {
	w.buf = encjson.AppendBool(w.buf, val)
}

func (w *JSONWriter) WriteInt(val int64) {
	w.buf = encjson.AppendInt(w.buf, val)
}

func (w *JSONWriter) WriteUint(val uint64) {
	w.buf = encjson.AppendUint(w.buf, val)
}

func (w *JSONWriter) WriteFloat(val float64) {
	w.buf = encjson.AppendFloat(w.buf, val)
}

func (w *JSONWriter) WriteString(val string) {
	w.buf = encjson.AppendString(w.buf, val)
}

func (w *JSONWriter) WriteError(val error) {
	w.buf = encjson.AppendString(w.buf, val.Error())
}

func (w *JSONWriter) WriteUUID(val [16]byte) {
	w.buf = encjson.AppendUUID(w.buf, val)
}

func (w *JSONWriter) WriteJSON(val []byte) {
	if len(val) == 0 {
		val = []byte("null")
	}
	w.buf = append(w.buf, val...)
}

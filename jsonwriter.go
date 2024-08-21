package golog

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/domonda/go-encjson"
)

var (
	_ Writer       = new(JSONWriter)
	_ WriterConfig = new(JSONWriterConfig)
)

type JSONWriterConfig struct {
	writer     io.Writer
	format     *Format
	filter     LevelFilter
	writerPool sync.Pool
}

func NewJSONWriterConfig(writer io.Writer, format *Format, filters ...LevelFilter) *JSONWriterConfig {
	if writer == nil {
		panic("nil writer")
	}
	if format == nil {
		format = NewDefaultFormat()
	}
	return &JSONWriterConfig{
		writer: writer,
		format: format,
		filter: JoinLevelFilters(filters...),
	}
}

func (c *JSONWriterConfig) WriterForNewMessage(ctx context.Context, level Level) Writer {
	if c.filter.IsInactive(ctx, level) {
		return nil
	}
	if w, _ := c.writerPool.Get().(Writer); w != nil {
		return w
	}
	return &JSONWriter{
		config: c,
		buf:    make([]byte, 0, 1024),
	}
}

func (c *JSONWriterConfig) FlushUnderlying() {
	flushUnderlying(c.writer)
}

///////////////////////////////////////////////////////////////////////////////

type JSONWriter struct {
	config *JSONWriterConfig
	buf    []byte
}

func (w *JSONWriter) BeginMessage(config Config, t time.Time, level Level, prefix, text string) {
	w.buf = append(w.buf, '{')

	if w.config.format.TimestampKey != "" {
		w.buf = encjson.AppendKey(w.buf, w.config.format.TimestampKey)
		w.buf = encjson.AppendTime(w.buf, t, w.config.format.TimestampFormat)
	}

	if w.config.format.LevelKey != "" {
		w.buf = encjson.AppendKey(w.buf, w.config.format.LevelKey)
		w.buf = encjson.AppendString(w.buf, config.Levels().Name(level))
	}

	if w.config.format.MessageKey != "" && text != "" {
		if prefix != "" {
			text = prefix + w.config.format.PrefixSep + text
		}
		w.buf = encjson.AppendKey(w.buf, w.config.format.MessageKey)
		w.buf = encjson.AppendString(w.buf, text)
	}
}

func (w *JSONWriter) CommitMessage() {
	// Flush f.buf
	if len(w.buf) > 0 {
		_, err := w.config.writer.Write(append(w.buf, '}', ',', '\n'))
		if err != nil && ErrorHandler != nil {
			ErrorHandler(fmt.Errorf("golog.JSONWriter error: %w", err))
		}
	}

	// Reset and return to pool
	w.buf = w.buf[:0]
	w.config.writerPool.Put(w)
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

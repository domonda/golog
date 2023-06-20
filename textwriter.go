package golog

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"
)

type sliceMode int

const (
	sliceModeNone       sliceMode = 0
	sliceModeFirstElem  sliceMode = 1
	sliceModeSecondElem sliceMode = 2
)

var textWriterPool sync.Pool

type TextWriter struct {
	writer    io.Writer
	format    *Format
	colorizer Colorizer
	sliceMode sliceMode
	buf       []byte
}

func NewTextWriter(writer io.Writer, format *Format, colorizer Colorizer) *TextWriter {
	return &TextWriter{
		writer:    writer,
		format:    format,
		colorizer: colorizer,
		buf:       make([]byte, 0, 1024),
	}
}

func getTextWriter(writer io.Writer, format *Format, colorizer Colorizer) *TextWriter {
	if recycled, ok := textWriterPool.Get().(*TextWriter); ok {
		recycled.writer = writer
		recycled.format = format
		recycled.colorizer = colorizer
		return recycled
	}
	return NewTextWriter(writer, format, colorizer)
}

func (w *TextWriter) BeginMessage(_ context.Context, logger *Logger, t time.Time, level Level, text string) Writer {
	next := getTextWriter(w.writer, w.format, w.colorizer)
	next.beginWriteMessage(logger, t, level, text)
	return next
}

func (w *TextWriter) beginWriteMessage(logger *Logger, t time.Time, level Level, text string) {
	// Write timestamp
	timestamp := t.Format(w.format.TimestampFormat)
	w.buf = append(w.buf, w.colorizer.ColorizeTimestamp(timestamp)...)
	w.buf = append(w.buf, ' ')

	// Write level
	levels := logger.Config().Levels()
	if min, max := levels.NameLenRange(); min != max {
		levels = levels.CopyWithRightPaddedNames() // TODO optimize performance
	}
	str := w.colorizer.ColorizeLevel(levels, level)
	w.buf = append(w.buf, '|')
	w.buf = append(w.buf, str...)
	w.buf = append(w.buf, '|')

	// Write message
	if text != "" {
		if logger.prefix != "" {
			text = logger.prefix + w.format.PrefixSep + text
		}
		w.buf = append(w.buf, ' ')
		w.buf = append(w.buf, w.colorizer.ColorizeMsg(text)...)
	}
}

func (w *TextWriter) CommitMessage() {
	// Flush w.buf
	if len(w.buf) > 0 {
		_, err := w.writer.Write(append(w.buf, '\n'))
		if err != nil && ErrorHandler != nil {
			ErrorHandler(fmt.Errorf("golog.TextWriter error: %w", err))
		}
	}

	// Free
	w.writer = nil
	w.format = nil
	w.sliceMode = sliceModeNone
	w.buf = w.buf[:0]
	textWriterPool.Put(w)
}

func (w *TextWriter) FlushUnderlying() {
	flushUnderlying(w.writer)
}

func (w *TextWriter) String() string {
	return string(w.buf)
}

func (w *TextWriter) WriteKey(key string) {
	str := w.colorizer.ColorizeKey(key)
	w.buf = append(w.buf, ' ')
	w.buf = append(w.buf, str...)
	w.buf = append(w.buf, '=')
}

func (w *TextWriter) WriteSliceKey(key string) {
	str := w.colorizer.ColorizeKey(key)
	w.buf = append(w.buf, ' ')
	w.buf = append(w.buf, str...)
	w.buf = append(w.buf, '=', '[')
	w.sliceMode = sliceModeFirstElem
}

func (w *TextWriter) WriteSliceEnd() {
	w.buf = append(w.buf, ']')
	w.sliceMode = sliceModeNone
}

func (w *TextWriter) writeSliceSep() {
	switch w.sliceMode {
	case sliceModeFirstElem:
		w.sliceMode = sliceModeSecondElem
	case sliceModeSecondElem:
		w.buf = append(w.buf, ',')
	}
}

func (w *TextWriter) WriteNil() {
	w.writeSliceSep()
	str := w.colorizer.ColorizeNil("nil")
	w.buf = append(w.buf, str...)
}

func (w *TextWriter) WriteBool(val bool) {
	w.writeSliceSep()
	var str string
	if val {
		str = w.colorizer.ColorizeTrue("true")
	} else {
		str = w.colorizer.ColorizeFalse("false")
	}
	w.buf = append(w.buf, str...)
}

func (w *TextWriter) WriteInt(val int64) {
	w.writeSliceSep()
	str := w.colorizer.ColorizeInt(strconv.FormatInt(val, 10))
	w.buf = append(w.buf, str...)
}

func (w *TextWriter) WriteUint(val uint64) {
	w.writeSliceSep()
	str := w.colorizer.ColorizeUint(strconv.FormatUint(val, 10))
	w.buf = append(w.buf, str...)
}

func (w *TextWriter) WriteFloat(val float64) {
	w.writeSliceSep()
	str := w.colorizer.ColorizeFloat(strconv.FormatFloat(val, 'f', -1, 64))
	w.buf = append(w.buf, str...)
}

func (w *TextWriter) WriteString(val string) {
	w.writeSliceSep()
	str := w.colorizer.ColorizeString(strconv.Quote(val))
	w.buf = append(w.buf, str...)
}

func (w *TextWriter) WriteError(val error) {
	w.writeSliceSep()

	lines := strings.Split(val.Error(), "\n")
	if len(lines) == 1 {
		w.buf = append(w.buf, '`')
		w.buf = append(w.buf, w.colorizer.ColorizeError(lines[0])...)
		w.buf = append(w.buf, '`')
	} else {
		w.buf = append(w.buf, '`', '\n')
		for _, line := range lines {
			w.buf = append(w.buf, w.colorizer.ColorizeError(line)...)
			w.buf = append(w.buf, '\n')
		}
		w.buf = append(w.buf, '`')
	}
}

// func (f *TextFormatter) WriteBytes(val []byte) {
// 	w.writeSliceSep()
// 	hexVal := make([]byte, len(val)*2+2)
// 	hexVal[0] = '"'
// 	hex.Encode(hexVal[1:], val)
// 	hexVal[len(hexVal)-1] = '"'
// 	w.buf.Write(hexVal)
// }

func (w *TextWriter) WriteUUID(val [16]byte) {
	w.writeSliceSep()

	str := w.colorizer.ColorizeUUID(FormatUUID(val))
	w.buf = append(w.buf, str...)
}

func (w *TextWriter) WriteJSON(val []byte) {
	w.buf = append(w.buf, val...)
}

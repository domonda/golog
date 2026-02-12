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

var (
	_ Writer       = new(TextWriter)
	_ WriterConfig = new(TextWriterConfig)
)

type TextWriterConfig struct {
	writer            io.Writer
	format            *Format
	colorizer         Colorizer
	filter            LevelFilter
	noColorizer       bool     // true when colorizer is NoColorizer (enables zero-allocation fast path)
	paddedLevelsCache sync.Map // map[*Levels]*Levels - caches padded level names
}

func NewTextWriterConfig(writer io.Writer, format *Format, colorizer Colorizer, filters ...LevelFilter) *TextWriterConfig {
	if writer == nil {
		panic("nil writer")
	}
	if format == nil {
		format = NewDefaultFormat()
	}
	if colorizer == nil {
		colorizer = NoColorizer
	}
	_, isNoColorizer := colorizer.(noColorizer)
	return &TextWriterConfig{
		writer:      writer,
		format:      format,
		filter:      JoinLevelFilters(filters...),
		colorizer:   colorizer,
		noColorizer: isNoColorizer,
	}
}

func (c *TextWriterConfig) WriterForNewMessage(ctx context.Context, level Level) Writer {
	if c.filter.IsInactive(ctx, level) {
		return nil
	}
	w := textWriterPool.GetOrNew()
	w.config = c
	if w.buf == nil {
		w.buf = make([]byte, 0, 1024)
	}
	return w
}

func (c *TextWriterConfig) FlushUnderlying() {
	flushUnderlying(c.writer)
}

///////////////////////////////////////////////////////////////////////////////

type TextWriter struct {
	config    *TextWriterConfig
	sliceMode sliceMode
	buf       []byte
}

func (w *TextWriter) BeginMessage(config Config, timestamp time.Time, level Level, prefix, text string) {
	// Write timestamp
	if w.config.noColorizer {
		// Fast path: append directly without allocation
		w.buf = timestamp.AppendFormat(w.buf, w.config.format.TimestampFormat)
	} else {
		timestampStr := timestamp.Format(w.config.format.TimestampFormat)
		w.buf = append(w.buf, w.config.colorizer.ColorizeTimestamp(timestampStr)...)
	}
	w.buf = append(w.buf, ' ')

	// Write level
	levels := config.Levels()
	if min, max := levels.NameLenRange(); min != max {
		// Use cached padded levels if available, otherwise compute and cache
		if cached, ok := w.config.paddedLevelsCache.Load(levels); ok {
			levels = cached.(*Levels)
		} else {
			padded := levels.CopyWithRightPaddedNames()
			w.config.paddedLevelsCache.Store(levels, padded)
			levels = padded
		}
	}
	str := w.config.colorizer.ColorizeLevel(levels, level)
	w.buf = append(w.buf, '|')
	w.buf = append(w.buf, str...)
	w.buf = append(w.buf, '|')

	if text == "" {
		return
	}
	// Write message
	w.buf = append(w.buf, ' ')
	if prefix != "" {
		w.buf = fmt.Appendf(w.buf, w.config.format.PrefixFmt, prefix, text)
	} else {
		w.buf = append(w.buf, w.config.colorizer.ColorizeMsg(text)...)
	}
}

func (w *TextWriter) CommitMessage() {
	// Flush w.buf
	if len(w.buf) > 0 {
		_, err := w.config.writer.Write(append(w.buf, '\n'))
		if err != nil && ErrorHandler != nil {
			ErrorHandler(fmt.Errorf("golog.TextWriter error: %w", err))
		}
	}

	// Reset and return to pool
	w.config = nil
	w.sliceMode = sliceModeNone
	w.buf = w.buf[:0]
	textWriterPool.PutBack(w)
}

func (w *TextWriter) String() string {
	return string(w.buf)
}

func (w *TextWriter) WriteKey(key string) {
	str := w.config.colorizer.ColorizeKey(key)
	w.buf = append(w.buf, ' ')
	w.buf = append(w.buf, str...)
	w.buf = append(w.buf, '=')
}

func (w *TextWriter) WriteSliceKey(key string) {
	str := w.config.colorizer.ColorizeKey(key)
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
	if w.config.noColorizer {
		w.buf = append(w.buf, "nil"...)
	} else {
		str := w.config.colorizer.ColorizeNil("nil")
		w.buf = append(w.buf, str...)
	}
}

func (w *TextWriter) WriteBool(val bool) {
	w.writeSliceSep()
	if w.config.noColorizer {
		if val {
			w.buf = append(w.buf, "true"...)
		} else {
			w.buf = append(w.buf, "false"...)
		}
	} else {
		var str string
		if val {
			str = w.config.colorizer.ColorizeTrue("true")
		} else {
			str = w.config.colorizer.ColorizeFalse("false")
		}
		w.buf = append(w.buf, str...)
	}
}

func (w *TextWriter) WriteInt(val int64) {
	w.writeSliceSep()
	if w.config.noColorizer {
		w.buf = strconv.AppendInt(w.buf, val, 10)
	} else {
		str := w.config.colorizer.ColorizeInt(strconv.FormatInt(val, 10))
		w.buf = append(w.buf, str...)
	}
}

func (w *TextWriter) WriteUint(val uint64) {
	w.writeSliceSep()
	if w.config.noColorizer {
		w.buf = strconv.AppendUint(w.buf, val, 10)
	} else {
		str := w.config.colorizer.ColorizeUint(strconv.FormatUint(val, 10))
		w.buf = append(w.buf, str...)
	}
}

func (w *TextWriter) WriteFloat(val float64) {
	w.writeSliceSep()
	if w.config.noColorizer {
		w.buf = strconv.AppendFloat(w.buf, val, 'f', -1, 64)
	} else {
		str := w.config.colorizer.ColorizeFloat(strconv.FormatFloat(val, 'f', -1, 64))
		w.buf = append(w.buf, str...)
	}
}

func (w *TextWriter) WriteString(val string) {
	w.writeSliceSep()
	if w.config.noColorizer {
		// Fast path: append directly without allocation
		w.buf = strconv.AppendQuote(w.buf, val)
	} else {
		str := w.config.colorizer.ColorizeString(strconv.Quote(val))
		w.buf = append(w.buf, str...)
	}
}

func (w *TextWriter) WriteError(val error) {
	if val == nil {
		w.WriteNil()
		return
	}
	w.writeSliceSep()

	lines := strings.Split(val.Error(), "\n")
	if len(lines) == 1 {
		w.buf = append(w.buf, '`')
		w.buf = append(w.buf, w.config.colorizer.ColorizeError(lines[0])...)
		w.buf = append(w.buf, '`')
	} else {
		w.buf = append(w.buf, '`', '\n')
		for _, line := range lines {
			w.buf = append(w.buf, w.config.colorizer.ColorizeError(line)...)
			w.buf = append(w.buf, '\n')
		}
		w.buf = append(w.buf, '`')
	}
}

func (w *TextWriter) WriteTime(val time.Time) {
	w.writeSliceSep()
	format := w.config.format.TimeFormat
	if format == "" {
		format = DefaultTimeFormat
	}
	if w.config.noColorizer {
		// Fast path: append directly without allocation
		w.buf = append(w.buf, '"')
		w.buf = val.AppendFormat(w.buf, format)
		w.buf = append(w.buf, '"')
	} else {
		str := w.config.colorizer.ColorizeString(strconv.Quote(val.Format(format)))
		w.buf = append(w.buf, str...)
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
	if w.config.noColorizer {
		w.buf = AppendUUID(w.buf, val)
	} else {
		str := w.config.colorizer.ColorizeUUID(FormatUUID(val))
		w.buf = append(w.buf, str...)
	}
}

func (w *TextWriter) WriteJSON(val []byte) {
	w.buf = append(w.buf, val...)
}

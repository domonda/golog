package golog

import (
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

var textFormatterPool sync.Pool

type TextFormatter struct {
	writer    io.Writer
	levels    *Levels
	format    *Format
	sliceMode sliceMode
	buf       []byte
	colorizer Colorizer
}

func NewTextFormatter(writer io.Writer, format *Format, colorizer Colorizer) *TextFormatter {
	return &TextFormatter{
		writer:    writer,
		format:    format,
		colorizer: colorizer,
		buf:       make([]byte, 0, 1024),
	}
}

func (f *TextFormatter) Clone(level Level) Formatter {
	if clone, ok := textFormatterPool.Get().(*TextFormatter); ok {
		clone.writer = f.writer
		clone.format = f.format
		clone.colorizer = f.colorizer
		return clone
	}
	return NewTextFormatter(f.writer, f.format, f.colorizer)
}

func (f *TextFormatter) WriteText(t time.Time, levels *Levels, level Level, prefix, text string) {
	// Write timestamp
	timestamp := t.Format(f.format.TimestampFormat)
	f.buf = append(f.buf, f.colorizer.ColorizeTimestamp(timestamp)...)
	f.buf = append(f.buf, ' ')

	// Write level
	if min, max := levels.NameLenRange(); min != max {
		levels = levels.CopyWithRightPaddedNames() // TODO optimize performance
	}
	str := f.colorizer.ColorizeLevel(levels, level)
	f.buf = append(f.buf, '|')
	f.buf = append(f.buf, str...)
	f.buf = append(f.buf, '|')

	// Write message
	if text != "" {
		f.buf = append(f.buf, ' ')
		f.buf = append(f.buf, f.colorizer.ColorizeMsg(prefix+text)...)
	}
}

func (f *TextFormatter) FlushAndFree() {
	// Flush f.buf
	if len(f.buf) > 0 {
		_, err := f.writer.Write(append(f.buf, '\n'))
		if err != nil && ErrorHandler != nil {
			ErrorHandler(fmt.Errorf("golog.TextFormatter error: %w", err))
		}
	}

	// Free
	f.writer = nil
	f.levels = nil
	f.format = nil
	f.sliceMode = sliceModeNone
	f.buf = f.buf[:0]
	textFormatterPool.Put(f)
}

func (f *TextFormatter) FlushUnderlying() {
	flushUnderlying(f.writer)
}

// String is here only for debugging
func (f *TextFormatter) String() string {
	return string(f.buf)
}

func (f *TextFormatter) WriteKey(key string) {
	str := f.colorizer.ColorizeKey(key)
	f.buf = append(f.buf, ' ')
	f.buf = append(f.buf, str...)
	f.buf = append(f.buf, '=')
}

func (f *TextFormatter) WriteSliceKey(key string) {
	str := f.colorizer.ColorizeKey(key)
	f.buf = append(f.buf, ' ')
	f.buf = append(f.buf, str...)
	f.buf = append(f.buf, '=', '[')
	f.sliceMode = sliceModeFirstElem
}

func (f *TextFormatter) WriteSliceEnd() {
	f.buf = append(f.buf, ']')
	f.sliceMode = sliceModeNone
}

func (f *TextFormatter) writeSliceSep() {
	switch f.sliceMode {
	case sliceModeFirstElem:
		f.sliceMode = sliceModeSecondElem
	case sliceModeSecondElem:
		f.buf = append(f.buf, ',')
	}
}

func (f *TextFormatter) WriteNil() {
	f.writeSliceSep()
	str := f.colorizer.ColorizeNil("nil")
	f.buf = append(f.buf, str...)
}

func (f *TextFormatter) WriteBool(val bool) {
	f.writeSliceSep()
	var str string
	if val {
		str = f.colorizer.ColorizeTrue("true")
	} else {
		str = f.colorizer.ColorizeFalse("false")
	}
	f.buf = append(f.buf, str...)
}

func (f *TextFormatter) WriteInt(val int64) {
	f.writeSliceSep()
	str := f.colorizer.ColorizeInt(strconv.FormatInt(val, 10))
	f.buf = append(f.buf, str...)
}

func (f *TextFormatter) WriteUint(val uint64) {
	f.writeSliceSep()
	str := f.colorizer.ColorizeUint(strconv.FormatUint(val, 10))
	f.buf = append(f.buf, str...)
}

func (f *TextFormatter) WriteFloat(val float64) {
	f.writeSliceSep()
	str := f.colorizer.ColorizeFloat(strconv.FormatFloat(val, 'f', -1, 64))
	f.buf = append(f.buf, str...)
}

func (f *TextFormatter) WriteString(val string) {
	f.writeSliceSep()
	str := f.colorizer.ColorizeString(strconv.Quote(val))
	f.buf = append(f.buf, str...)
}

func (f *TextFormatter) WriteError(val error) {
	f.writeSliceSep()

	lines := strings.Split(val.Error(), "\n")
	if len(lines) == 1 {
		f.buf = append(f.buf, '`')
		f.buf = append(f.buf, f.colorizer.ColorizeError(lines[0])...)
		f.buf = append(f.buf, '`')
	} else {
		f.buf = append(f.buf, '`', '\n')
		for _, line := range lines {
			f.buf = append(f.buf, f.colorizer.ColorizeError(line)...)
			f.buf = append(f.buf, '\n')
		}
		f.buf = append(f.buf, '`')
	}
}

// func (f *TextFormatter) WriteBytes(val []byte) {
// 	f.writeSliceSep()
// 	hexVal := make([]byte, len(val)*2+2)
// 	hexVal[0] = '"'
// 	hex.Encode(hexVal[1:], val)
// 	hexVal[len(hexVal)-1] = '"'
// 	f.buf.Write(hexVal)
// }

func (f *TextFormatter) WriteUUID(val [16]byte) {
	f.writeSliceSep()

	str := f.colorizer.ColorizeUUID(FormatUUID(val))
	f.buf = append(f.buf, str...)
}

func (f *TextFormatter) WriteJSON(val []byte) {
	f.buf = append(f.buf, val...)
}

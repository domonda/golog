package golog

import (
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"
)

var textFormatterPool sync.Pool

type TextFormatter struct {
	parent    *TextFormatter
	writer    io.Writer
	format    *Format
	sliceMode sliceMode
	buf       []byte
	colorizer Colorizer
}

func NewTextFormatter(writer io.Writer, format *Format, colorizer Colorizer) *TextFormatter {
	if min, max := format.Levels.NameLenRange(); min != max {
		paddedFormat := *format
		paddedFormat.Levels = format.Levels.CopyWithRightPaddedNames()
		format = &paddedFormat
	}

	return &TextFormatter{
		writer:    writer,
		format:    format,
		colorizer: colorizer,
		buf:       make([]byte, 0, 1024),
	}
}

func (f *TextFormatter) NewChild() Formatter {
	if child, ok := textFormatterPool.Get().(*TextFormatter); ok {
		child.writer = f.writer
		child.format = f.format
		child.colorizer = f.colorizer
		return f
	}

	return NewTextFormatter(f.writer, f.format, f.colorizer)
}

func (f *TextFormatter) WriteMsg(t time.Time, level Level, msg string) {
	// Write timestamp
	timestamp := t.Format(f.format.TimestampFormat)
	f.buf = append(f.buf, f.colorizer.ColorizeTimestamp(timestamp)...)
	f.buf = append(f.buf, ' ')

	// Write level
	str := f.colorizer.ColorizeLevel(f.format.Levels.Name(level))
	f.buf = append(f.buf, '|')
	f.buf = append(f.buf, str...)
	f.buf = append(f.buf, '|')

	// Write message
	if msg != "" {
		f.buf = append(f.buf, ' ')
		f.buf = append(f.buf, f.colorizer.ColorizeMsg(msg)...)
	}

	f.buf = f.appendParent(f.buf)
}

func (f *TextFormatter) appendParent(buf []byte) []byte {
	if f.parent != nil {
		buf = f.parent.appendParent(buf)
		if len(f.parent.buf) > 0 {
			buf = append(buf, ' ')
			buf = append(buf, f.parent.buf...)
		}
	}
	return buf
}

func (f *TextFormatter) FlushAndFree() {
	// Flush
	f.buf = append(f.buf, '\n')
	_, err := f.writer.Write(f.buf)
	if err != nil && ErrorHandler != nil {
		ErrorHandler(fmt.Errorf("golog.TextFormatter error: %w", err))
	}

	// Free
	f.parent = nil
	f.writer = nil
	f.format = nil
	f.sliceMode = sliceModeNone
	f.buf = f.buf[:0]
	textFormatterPool.Put(f)
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

	var b [36]byte
	hex.Encode(b[0:8], val[0:4])
	b[8] = '-'
	hex.Encode(b[9:13], val[4:6])
	b[13] = '-'
	hex.Encode(b[14:18], val[6:8])
	b[18] = '-'
	hex.Encode(b[19:23], val[8:10])
	b[23] = '-'
	hex.Encode(b[24:36], val[10:16])

	str := f.colorizer.ColorizeUUID(string(b[:]))
	f.buf = append(f.buf, str...)
}

func (f *TextFormatter) WriteJSON(val []byte) {
	f.buf = append(f.buf, val...)
}

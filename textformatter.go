package golog

import (
	"encoding/hex"
	"io"
	"strconv"
	"sync"
	"time"
)

var textFormatterPool sync.Pool

type textFormatter struct {
	writer    io.Writer
	format    *Format
	sliceMode sliceMode
	buf       []byte
	colorizer Colorizer
}

func newTextFormatter(writer io.Writer, format *Format, colorizer Colorizer) *textFormatter {
	if f, ok := textFormatterPool.Get().(*textFormatter); ok {
		f.writer = writer
		f.format = format
		f.sliceMode = sliceModeNone
		f.colorizer = colorizer
		f.buf = f.buf[:0]
		return f
	}

	return &textFormatter{
		writer:    writer,
		format:    format,
		sliceMode: sliceModeNone,
		colorizer: colorizer,
		buf:       make([]byte, 0, 1024),
	}
}

func NewTextFormatterFuncWithColorizer(colorizer Colorizer) NewFormatterFunc {
	return func(writer io.Writer, format *Format) Formatter {
		return newTextFormatter(writer, format, colorizer)
	}
}

func NewTextFormatter(writer io.Writer, format *Format) Formatter {
	return newTextFormatter(writer, format, DefaultColorizer)
}

func (f *textFormatter) Begin(t time.Time, level Level, msg string, data []byte) {
	// Write timestamp
	timestamp := t.Format(f.format.TimestampFormat)
	f.buf = append(f.buf, f.colorizer.ColorizeTimestamp(timestamp)...)
	f.buf = append(f.buf, ' ')

	// Write level
	str := f.colorizer.ColorizeLevel(f.format.Levels.Name(level))
	f.buf = append(f.buf, str...)
	// f.buf = append(f.buf, f.format.LevelPadding...)
	f.buf = append(f.buf, ':', ' ')

	// Write message
	f.buf = strconv.AppendQuote(f.buf, f.colorizer.ColorizeMsg(msg))

	// Write data from super logger
	f.buf = append(f.buf, data...)
}

func (f *textFormatter) growValBuf(n int) {
}

func (f *textFormatter) WriteKey(key string) {
	str := f.colorizer.ColorizeKey(key)
	f.growValBuf(len(key) + 2)
	f.buf = append(f.buf, ' ')
	f.buf = append(f.buf, str...)
	f.buf = append(f.buf, '=')
}

func (f *textFormatter) WriteSliceKey(key string) {
	str := f.colorizer.ColorizeKey(key)
	f.growValBuf(len(key) + 3)
	f.buf = append(f.buf, ' ')
	f.buf = append(f.buf, str...)
	f.buf = append(f.buf, '=', '[')
	f.sliceMode = sliceModeFirstElem
}

func (f *textFormatter) WriteSliceEnd() {
	f.buf = append(f.buf, ']')
	f.sliceMode = sliceModeNone
}

func (f *textFormatter) writeSliceSep() {
	switch f.sliceMode {
	case sliceModeFirstElem:
		f.sliceMode = sliceModeSecondElem
	case sliceModeSecondElem:
		f.buf = append(f.buf, ',')
	}
}

func (f *textFormatter) WriteBool(val bool) {
	f.writeSliceSep()
	var str string
	if val {
		str = f.colorizer.ColorizeTrue("true")
	} else {
		str = f.colorizer.ColorizeFalse("false")
	}
	f.buf = append(f.buf, str...)
}

func (f *textFormatter) WriteInt(val int64) {
	f.writeSliceSep()
	str := f.colorizer.ColorizeInt(strconv.FormatInt(val, 10))
	f.buf = append(f.buf, str...)
}

func (f *textFormatter) WriteUint(val uint64) {
	f.writeSliceSep()
	str := f.colorizer.ColorizeUint(strconv.FormatUint(val, 10))
	f.buf = append(f.buf, str...)
}

func (f *textFormatter) WriteFloat(val float64) {
	f.writeSliceSep()
	str := f.colorizer.ColorizeFloat(strconv.FormatFloat(val, 'f', -1, 64))
	f.buf = append(f.buf, str...)
}

func (f *textFormatter) WriteString(val string) {
	f.writeSliceSep()
	str := f.colorizer.ColorizeString(strconv.Quote(val))
	f.buf = append(f.buf, str...)
}

// func (f *textFormatter) Bytes(val []byte) {
// 	f.writeSliceSep()
// 	hexVal := make([]byte, len(val)*2+2)
// 	hexVal[0] = '"'
// 	hex.Encode(hexVal[1:], val)
// 	hexVal[len(hexVal)-1] = '"'
// 	f.buf.Write(hexVal)
// }

func (f *textFormatter) WriteUUID(val [16]byte) {
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

func (f *textFormatter) Flush() {
	f.buf = append(f.buf, '\n')
	// s := string(f.buf)
	f.writer.Write(f.buf)

	f.writer = nil
	f.format = nil
	textFormatterPool.Put(f)
}

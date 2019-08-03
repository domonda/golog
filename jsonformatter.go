package golog

import (
	"encoding/hex"
	"io"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/domonda/golog/json"
)

var jsonFormatterPool sync.Pool

type jsonFormatter struct {
	writer    io.Writer
	format    *Format
	sliceMode sliceMode
	buf       []byte
}

func NewJSONFormatter(writer io.Writer, format *Format) Formatter {
	if f, ok := jsonFormatterPool.Get().(*jsonFormatter); ok {
		f.writer = writer
		f.format = format
		f.sliceMode = sliceModeNone
		f.buf = f.buf[:0]
		return f
	}

	return &jsonFormatter{
		writer:    writer,
		format:    format,
		sliceMode: sliceModeNone,
		buf:       make([]byte, 0, 1024),
	}
}

func (f *jsonFormatter) Begin(t time.Time, level Level, msg string, data []byte) {
	f.buf = append(f.buf, '{')

	// Write timestamp
	timestamp := t.Format(f.format.TimestampFormat)
	f.buf = json.EncodeString(f.buf, f.format.TimestampKey)
	f.buf = append(f.buf, ':')
	f.buf = json.EncodeString(f.buf, timestamp)

	// Write level
	if f.format.LevelKey != "" {
		f.buf = append(f.buf, ',')
		f.buf = json.EncodeString(f.buf, f.format.LevelKey)
		f.buf = append(f.buf, ':')
		f.buf = json.EncodeString(f.buf, f.format.Levels.Name(level))
	}

	// Write message
	f.buf = append(f.buf, ',')
	f.buf = json.EncodeString(f.buf, f.format.MessageKey)
	f.buf = append(f.buf, ':')
	f.buf = json.EncodeString(f.buf, msg)

	// Write data from super logger
	f.buf = append(f.buf, data...)
}

func (f *jsonFormatter) growValBuf(n int) {
}

func (f *jsonFormatter) WriteKey(key string) {
	// f.buf = growBytesCap(f.buf, len(key)+4)
	f.buf = append(f.buf, ',')
	f.buf = json.EncodeString(f.buf, key)
	f.buf = append(f.buf, ':')
}

func (f *jsonFormatter) WriteSliceKey(key string) {
	// f.buf = growBytesCap(f.buf, len(key)+5)
	f.buf = append(f.buf, ',')
	f.buf = json.EncodeString(f.buf, key)
	f.buf = append(f.buf, ':', '[')
	f.sliceMode = sliceModeFirstElem
}

func (f *jsonFormatter) WriteSliceEnd() {
	f.buf = append(f.buf, ']')
	f.sliceMode = sliceModeNone
}

func (f *jsonFormatter) writeSliceSep() {
	switch f.sliceMode {
	case sliceModeFirstElem:
		f.sliceMode = sliceModeSecondElem
	case sliceModeSecondElem:
		f.buf = append(f.buf, ',')
	}
}

func (f *jsonFormatter) WriteBool(val bool) {
	f.writeSliceSep()
	if val {
		f.buf = append(f.buf, "true"...)
	} else {
		f.buf = append(f.buf, "false"...)
	}
}

func (f *jsonFormatter) WriteInt(val int64) {
	f.writeSliceSep()
	f.buf = strconv.AppendInt(f.buf, val, 10)
}

func (f *jsonFormatter) WriteUint(val uint64) {
	f.writeSliceSep()
	f.buf = strconv.AppendUint(f.buf, val, 10)
}

func (f *jsonFormatter) WriteFloat(val float64) {
	f.writeSliceSep()
	// Special floats like NaN and Inf have to be written as quoted strings
	switch {
	case math.IsNaN(val):
		f.buf = append(f.buf, `"NaN"`...)
	case val > math.MaxFloat64:
		f.buf = append(f.buf, `"+Inf"`...)
	case val < -math.MaxFloat64:
		f.buf = append(f.buf, `"-Inf"`...)
	default:
		f.buf = strconv.AppendFloat(f.buf, val, 'f', -1, 64)
	}
}

func (f *jsonFormatter) WriteString(val string) {
	f.writeSliceSep()
	f.buf = json.EncodeString(f.buf, val)
}

func (f *jsonFormatter) WriteUUID(val [16]byte) {
	f.writeSliceSep()

	// TODO use grow len first then write directly to slice
	var b [38]byte
	b[0] = '"'
	hex.Encode(b[1:9], val[0:4])
	b[9] = '-'
	hex.Encode(b[10:14], val[4:6])
	b[14] = '-'
	hex.Encode(b[15:19], val[6:8])
	b[19] = '-'
	hex.Encode(b[20:24], val[8:10])
	b[2] = '-'
	hex.Encode(b[25:37], val[10:16])
	b[37] = '"'

	f.buf = append(f.buf, b[:]...)
}

// func (f *jsonFormatter) Bytes(val []byte) {
// 	f.writeSliceSep()
// 	hexVal := make([]byte, len(val)*2+2)
// 	hexVal[0] = '"'
// 	hex.Encode(hexVal[1:], val)
// 	hexVal[len(hexVal)-1] = '"'
// 	f.buf.Write(hexVal)
// }

func (f *jsonFormatter) Flush() {
	f.buf = append(f.buf, '}', '\n')
	f.writer.Write(f.buf)

	f.writer = nil
	f.format = nil
	textFormatterPool.Put(f)
}

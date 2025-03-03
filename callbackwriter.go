package golog

import (
	"context"
	"strings"
	"sync"
	"time"
)

var (
	_ Writer       = new(CallbackWriter)
	_ WriterConfig = new(CallbackWriterConfig)
)

// MessageCallback is called when a message is committed.
//
// The passed attribs slice is not retained after the callback,
// so it must be copied if needed.
type MessageCallback func(timestamp time.Time, level Level, prefix, text string, attribs Attribs)

type CallbackWriterConfig struct {
	filter     LevelFilter
	callback   MessageCallback
	writerPool sync.Pool
}

func NewCallbackWriterConfig(callback MessageCallback, filters ...LevelFilter) *CallbackWriterConfig {
	if callback == nil {
		panic("nil callback")
	}
	return &CallbackWriterConfig{
		filter:   JoinLevelFilters(filters...),
		callback: callback,
	}
}

func (c *CallbackWriterConfig) WriterForNewMessage(ctx context.Context, level Level) Writer {
	if c.filter.IsInactive(ctx, level) {
		return nil
	}
	if w, _ := c.writerPool.Get().(Writer); w != nil {
		return w
	}
	return &CallbackWriter{config: c}
}

func (c *CallbackWriterConfig) FlushUnderlying() {}

///////////////////////////////////////////////////////////////////////////////

type CallbackWriter struct {
	config      *CallbackWriterConfig
	levels      *Levels
	timestamp   time.Time
	level       Level
	prefix      string
	text        string
	attribs     Attribs
	key         string
	isSlice     bool
	sliceAttrib SliceAttrib
}

func (w *CallbackWriter) BeginMessage(config Config, timestamp time.Time, level Level, prefix, text string) {
	w.levels = config.Levels()
	w.timestamp = timestamp
	w.level = level
	w.prefix = prefix
	w.text = text
}

func (w *CallbackWriter) CommitMessage() {
	defer func() {
		// Recover from any panic in callback
		recover()

		// Reset and return to pool
		w.levels = nil
		w.timestamp = time.Time{}
		w.level = 0
		w.prefix = ""
		w.text = ""
		w.attribs = w.attribs[:0]
		w.key = ""
		w.isSlice = false
		w.sliceAttrib = nil

		w.config.writerPool.Put(w)
	}()

	w.config.callback(w.timestamp, w.level, w.prefix, w.text, w.attribs)
}

func (w *CallbackWriter) String() string {
	var b strings.Builder
	b.WriteString(w.timestamp.Format("2006-01-02T15:04:05.999"))
	b.WriteString(" ")
	b.WriteString(w.levels.Name(w.level))
	b.WriteString(" ")
	if w.prefix != "" {
		b.WriteString(w.prefix)
		b.WriteString(": ")
	}
	b.WriteString(w.text)
	for _, attrib := range w.attribs {
		b.WriteString(" ")
		b.WriteString(attrib.GetKey())
		b.WriteString("=")
		b.WriteString(attrib.GetValString())
	}
	return b.String()
}

func (w *CallbackWriter) WriteKey(key string) {
	w.key = key
}

func (w *CallbackWriter) WriteSliceKey(key string) {
	w.key = key
	w.isSlice = true
}

func (w *CallbackWriter) WriteSliceEnd() {
	w.attribs = append(w.attribs, w.sliceAttrib)
	w.sliceAttrib = nil
	w.isSlice = false
}

func (w *CallbackWriter) WriteNil() {
	w.attribs = append(w.attribs, Nil{Key: w.key})
}

func (w *CallbackWriter) WriteBool(val bool) {
	if w.isSlice {
		a, _ := w.sliceAttrib.(Bools)
		w.sliceAttrib = Bools{Key: w.key, Vals: append(a.Vals, val)}
	} else {
		w.attribs = append(w.attribs, Bool{Key: w.key, Val: val})
	}
}

func (w *CallbackWriter) WriteInt(val int64) {
	if w.isSlice {
		a, _ := w.sliceAttrib.(Ints)
		w.sliceAttrib = Ints{Key: w.key, Vals: append(a.Vals, val)}
	} else {
		w.attribs = append(w.attribs, Int{Key: w.key, Val: val})
	}
}

func (w *CallbackWriter) WriteUint(val uint64) {
	if w.isSlice {
		a, _ := w.sliceAttrib.(Uints)
		w.sliceAttrib = Uints{Key: w.key, Vals: append(a.Vals, val)}
	} else {
		w.attribs = append(w.attribs, Uint{Key: w.key, Val: val})
	}
}

func (w *CallbackWriter) WriteFloat(val float64) {
	if w.isSlice {
		a, _ := w.sliceAttrib.(Floats)
		w.sliceAttrib = Floats{Key: w.key, Vals: append(a.Vals, val)}
	} else {
		w.attribs = append(w.attribs, Float{Key: w.key, Val: val})
	}
}

func (w *CallbackWriter) WriteString(val string) {
	if w.isSlice {
		a, _ := w.sliceAttrib.(Strings)
		w.sliceAttrib = Strings{Key: w.key, Vals: append(a.Vals, val)}
	} else {
		w.attribs = append(w.attribs, String{Key: w.key, Val: val})
	}
}

func (w *CallbackWriter) WriteError(val error) {
	if w.isSlice {
		a, _ := w.sliceAttrib.(Errors)
		w.sliceAttrib = Errors{Key: w.key, Vals: append(a.Vals, val)}
	} else {
		w.attribs = append(w.attribs, Error{Key: w.key, Val: val})
	}
}

func (w *CallbackWriter) WriteUUID(val [16]byte) {
	if w.isSlice {
		a, _ := w.sliceAttrib.(UUIDs)
		w.sliceAttrib = UUIDs{Key: w.key, Vals: append(a.Vals, val)}
	} else {
		w.attribs = append(w.attribs, UUID{Key: w.key, Val: val})
	}
}

func (w *CallbackWriter) WriteJSON(val []byte) {
	if len(val) == 0 {
		val = []byte("null")
	}
	w.attribs = append(w.attribs, JSON{Key: w.key, Val: val})
}

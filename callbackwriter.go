package golog

import (
	"context"
	"strings"
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
	filter   LevelFilter
	callback MessageCallback
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
	w := callbackWriterPool.GetOrNew()
	w.config = c
	return w
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

func (w *CallbackWriter) reset() {
	// First clear data then put back into the pool
	w.attribs.Free()

	// Then zero the writer
	var zero CallbackWriter
	*w = zero
}

func (w *CallbackWriter) CommitMessage() {
	defer func() {
		recover() // Recover from any panic in w.config.callback
		w.reset()
		callbackWriterPool.PutBack(w)
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
		b.WriteString(attrib.Key())
		b.WriteString("=")
		b.WriteString(attrib.ValueString())
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
	w.attribs = append(w.attribs, NewNil(w.key))
}

func (w *CallbackWriter) WriteBool(val bool) {
	if w.isSlice {
		a, _ := w.sliceAttrib.(*Bools)
		if a == nil {
			a = NewBools(w.key, nil)
			w.sliceAttrib = a
		}
		a.vals = append(a.vals, val)
	} else {
		w.attribs = append(w.attribs, NewBool(w.key, val))
	}
}

func (w *CallbackWriter) WriteInt(val int64) {
	if w.isSlice {
		a, _ := w.sliceAttrib.(*Ints)
		if a == nil {
			a = NewInts(w.key, nil)
			w.sliceAttrib = a
		}
		a.vals = append(a.vals, val)
	} else {
		w.attribs = append(w.attribs, NewInt(w.key, val))
	}
}

func (w *CallbackWriter) WriteUint(val uint64) {
	if w.isSlice {
		a, _ := w.sliceAttrib.(*Uints)
		if a == nil {
			a = NewUints(w.key, nil)
			w.sliceAttrib = a
		}
		a.vals = append(a.vals, val)
	} else {
		w.attribs = append(w.attribs, NewUint(w.key, val))
	}
}

func (w *CallbackWriter) WriteFloat(val float64) {
	if w.isSlice {
		a, _ := w.sliceAttrib.(*Floats)
		if a == nil {
			a = NewFloats(w.key, nil)
			w.sliceAttrib = a
		}
		a.vals = append(a.vals, val)
	} else {
		w.attribs = append(w.attribs, NewFloat(w.key, val))
	}
}

func (w *CallbackWriter) WriteString(val string) {
	if w.isSlice {
		a, _ := w.sliceAttrib.(*Strings)
		if a == nil {
			a = NewStrings(w.key, nil)
			w.sliceAttrib = a
		}
		a.vals = append(a.vals, val)
	} else {
		w.attribs = append(w.attribs, NewString(w.key, val))
	}
}

func (w *CallbackWriter) WriteError(val error) {
	if val == nil {
		w.WriteNil()
		return
	}
	if w.isSlice {
		a, _ := w.sliceAttrib.(*Errors)
		if a == nil {
			a = NewErrors(w.key, nil)
			w.sliceAttrib = a
		}
		a.vals = append(a.vals, val)
	} else {
		w.attribs = append(w.attribs, NewError(w.key, val))
	}
}

func (w *CallbackWriter) WriteTime(val time.Time) {
	if w.isSlice {
		a, _ := w.sliceAttrib.(*Times)
		if a == nil {
			a = NewTimes(w.key, nil)
			w.sliceAttrib = a
		}
		a.vals = append(a.vals, val)
	} else {
		w.attribs = append(w.attribs, NewTime(w.key, val))
	}
}

func (w *CallbackWriter) WriteUUID(val [16]byte) {
	if w.isSlice {
		a, _ := w.sliceAttrib.(*UUIDs)
		if a == nil {
			a = NewUUIDs(w.key, nil)
			w.sliceAttrib = a
		}
		a.vals = append(a.vals, val)
	} else {
		w.attribs = append(w.attribs, NewUUID(w.key, val))
	}
}

func (w *CallbackWriter) WriteJSON(val []byte) {
	if len(val) == 0 {
		val = []byte("null")
	}
	w.attribs = append(w.attribs, NewJSON(w.key, val))
}

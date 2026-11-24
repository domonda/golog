package golog

import (
	"context"
	"encoding/json"
	"reflect"
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
//
// The callback is called concurrently when the logger is used
// by multiple goroutines.
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
	// mixedSlice collects the elements of a slice whose elements
	// have different types, like a []any, or are nil or JSON
	mixedSlice []any
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
	switch {
	case w.sliceAttrib != nil:
		w.attribs = append(w.attribs, w.sliceAttrib)
	case w.mixedSlice != nil:
		w.attribs = append(w.attribs, NewAny(w.key, w.mixedSlice))
	default:
		// An empty slice has no element that created a slice attrib.
		// Never append a nil attrib, the callback and Free would panic on it.
		w.attribs = append(w.attribs, NewAny(w.key, []any{}))
	}
	w.sliceAttrib = nil
	w.mixedSlice = nil
	w.isSlice = false
}

// appendMixedSliceElem appends val to the elements of a slice
// whose elements have different types and returns true.
// If the earlier elements were collected in a typed slice attrib
// of another type than val, they are moved to the mixed elements first.
// It returns false for the first element of a slice.
// Callers call it only if val does not fit into w.sliceAttrib.
func (w *CallbackWriter) appendMixedSliceElem(val any) bool {
	if w.sliceAttrib == nil && w.mixedSlice == nil {
		return false
	}
	w.toMixedSlice()
	w.mixedSlice = append(w.mixedSlice, val)
	return true
}

// toMixedSlice moves the elements collected in w.sliceAttrib
// to w.mixedSlice, which is not nil afterwards.
func (w *CallbackWriter) toMixedSlice() {
	if w.mixedSlice != nil {
		return
	}
	if w.sliceAttrib == nil {
		w.mixedSlice = make([]any, 0, 1)
		return
	}
	prev := reflect.ValueOf(w.sliceAttrib.Value())
	w.mixedSlice = make([]any, prev.Len(), prev.Len()+1)
	for i := range w.mixedSlice {
		w.mixedSlice[i] = prev.Index(i).Interface()
	}
	w.sliceAttrib.Free()
	w.sliceAttrib = nil
}

func (w *CallbackWriter) WriteNil() {
	if w.isSlice {
		// Keep a nil element at its position in the slice
		w.toMixedSlice()
		w.mixedSlice = append(w.mixedSlice, nil)
		return
	}
	w.attribs = append(w.attribs, NewNil(w.key))
}

func (w *CallbackWriter) WriteBool(val bool) {
	if w.isSlice {
		if a, ok := w.sliceAttrib.(*Bools); ok {
			a.vals = append(a.vals, val)
		} else if !w.appendMixedSliceElem(val) {
			w.sliceAttrib = NewBools(w.key, []bool{val})
		}
	} else {
		w.attribs = append(w.attribs, NewBool(w.key, val))
	}
}

func (w *CallbackWriter) WriteInt(val int64) {
	if w.isSlice {
		if a, ok := w.sliceAttrib.(*Ints); ok {
			a.vals = append(a.vals, val)
		} else if !w.appendMixedSliceElem(val) {
			w.sliceAttrib = NewInts(w.key, []int64{val})
		}
	} else {
		w.attribs = append(w.attribs, NewInt(w.key, val))
	}
}

func (w *CallbackWriter) WriteUint(val uint64) {
	if w.isSlice {
		if a, ok := w.sliceAttrib.(*Uints); ok {
			a.vals = append(a.vals, val)
		} else if !w.appendMixedSliceElem(val) {
			w.sliceAttrib = NewUints(w.key, []uint64{val})
		}
	} else {
		w.attribs = append(w.attribs, NewUint(w.key, val))
	}
}

func (w *CallbackWriter) WriteFloat(val float64) {
	if w.isSlice {
		if a, ok := w.sliceAttrib.(*Floats); ok {
			a.vals = append(a.vals, val)
		} else if !w.appendMixedSliceElem(val) {
			w.sliceAttrib = NewFloats(w.key, []float64{val})
		}
	} else {
		w.attribs = append(w.attribs, NewFloat(w.key, val))
	}
}

func (w *CallbackWriter) WriteString(val string) {
	if w.isSlice {
		if a, ok := w.sliceAttrib.(*Strings); ok {
			a.vals = append(a.vals, val)
		} else if !w.appendMixedSliceElem(val) {
			w.sliceAttrib = NewStrings(w.key, []string{val})
		}
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
		if a, ok := w.sliceAttrib.(*Errors); ok {
			a.vals = append(a.vals, val)
		} else if !w.appendMixedSliceElem(val) {
			w.sliceAttrib = NewErrors(w.key, []error{val})
		}
	} else {
		w.attribs = append(w.attribs, NewError(w.key, val))
	}
}

func (w *CallbackWriter) WriteTime(val time.Time) {
	if w.isSlice {
		if a, ok := w.sliceAttrib.(*Times); ok {
			a.vals = append(a.vals, val)
		} else if !w.appendMixedSliceElem(val) {
			w.sliceAttrib = NewTimes(w.key, []time.Time{val})
		}
	} else {
		w.attribs = append(w.attribs, NewTime(w.key, val))
	}
}

func (w *CallbackWriter) WriteUUID(val [16]byte) {
	if w.isSlice {
		if a, ok := w.sliceAttrib.(*UUIDs); ok {
			a.vals = append(a.vals, val)
		} else if !w.appendMixedSliceElem(val) {
			w.sliceAttrib = NewUUIDs(w.key, [][16]byte{val})
		}
	} else {
		w.attribs = append(w.attribs, NewUUID(w.key, val))
	}
}

func (w *CallbackWriter) WriteJSON(val []byte) {
	if len(val) == 0 {
		val = []byte("null")
	}
	if w.isSlice {
		// Keep a JSON element at its position in the slice
		w.toMixedSlice()
		w.mixedSlice = append(w.mixedSlice, json.RawMessage(val))
		return
	}
	w.attribs = append(w.attribs, NewJSON(w.key, val))
}

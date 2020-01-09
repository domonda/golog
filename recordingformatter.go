package golog

import (
	"fmt"
	"time"
)

type recordingFormatter struct {
	values     []NamedValue
	key        string
	slice      bool
	sliceValue NamedValue
}

func (r *recordingFormatter) Clone(level Level) Formatter {
	panic("should never happen")
}

func (r *recordingFormatter) WriteText(t time.Time, levels *Levels, level Level, prefix, text string) {
	panic("should never happen")
}

func (r *recordingFormatter) FlushAndFree() {
	panic("calling golog.Message.Log() after Logger.Record() is invalid, call Message.NewLogger() instead")
}

// String is here only for debugging
func (r *recordingFormatter) String() string {
	return fmt.Sprintf("recordingFormatter with %d recored values", r.values)
}

func (r *recordingFormatter) WriteKey(key string) {
	r.key = key
}

func (r *recordingFormatter) WriteSliceKey(key string) {
	r.key = key
	r.slice = true
}

func (r *recordingFormatter) WriteSliceEnd() {
	r.values = append(r.values, r.sliceValue)
	r.sliceValue = nil
	r.slice = false
}

func (r *recordingFormatter) WriteNil() {
	r.values = append(r.values, &NilNamedValue{Key: r.key})
}

func (r *recordingFormatter) WriteBool(val bool) {
	if !r.slice {
		r.values = append(r.values, &BoolNamedValue{Key: r.key, Val: val})
		return
	}
	if slice, ok := r.sliceValue.(*BoolsNamedValue); ok {
		slice.Vals = append(slice.Vals, val)
	} else {
		r.sliceValue = &BoolsNamedValue{Key: r.key, Vals: []bool{val}}
	}
}

func (r *recordingFormatter) WriteInt(val int64) {
	if !r.slice {
		r.values = append(r.values, &IntNamedValue{Key: r.key, Val: val})
		return
	}
	if slice, ok := r.sliceValue.(*IntsNamedValue); ok {
		slice.Vals = append(slice.Vals, val)
	} else {
		r.sliceValue = &IntsNamedValue{Key: r.key, Vals: []int64{val}}
	}
}

func (r *recordingFormatter) WriteUint(val uint64) {
	if !r.slice {
		r.values = append(r.values, &UintNamedValue{Key: r.key, Val: val})
		return
	}
	if slice, ok := r.sliceValue.(*UintsNamedValue); ok {
		slice.Vals = append(slice.Vals, val)
	} else {
		r.sliceValue = &UintsNamedValue{Key: r.key, Vals: []uint64{val}}
	}
}

func (r *recordingFormatter) WriteFloat(val float64) {
	if !r.slice {
		r.values = append(r.values, &FloatNamedValue{Key: r.key, Val: val})
		return
	}
	if slice, ok := r.sliceValue.(*FloatsNamedValue); ok {
		slice.Vals = append(slice.Vals, val)
	} else {
		r.sliceValue = &FloatsNamedValue{Key: r.key, Vals: []float64{val}}
	}
}

func (r *recordingFormatter) WriteString(val string) {
	if !r.slice {
		r.values = append(r.values, &StringNamedValue{Key: r.key, Val: val})
		return
	}
	if slice, ok := r.sliceValue.(*StringsNamedValue); ok {
		slice.Vals = append(slice.Vals, val)
	} else {
		r.sliceValue = &StringsNamedValue{Key: r.key, Vals: []string{val}}
	}
}

func (r *recordingFormatter) WriteError(val error) {
	if !r.slice {
		r.values = append(r.values, &ErrorNamedValue{Key: r.key, Val: val})
		return
	}
	if slice, ok := r.sliceValue.(*ErrorsNamedValue); ok {
		slice.Vals = append(slice.Vals, val)
	} else {
		r.sliceValue = &ErrorsNamedValue{Key: r.key, Vals: []error{val}}
	}
}

func (r *recordingFormatter) WriteUUID(val [16]byte) {
	if !r.slice {
		r.values = append(r.values, &UUIDNamedValue{Key: r.key, Val: val})
		return
	}
	if slice, ok := r.sliceValue.(*UUIDsNamedValue); ok {
		slice.Vals = append(slice.Vals, val)
	} else {
		r.sliceValue = &UUIDsNamedValue{Key: r.key, Vals: [][16]byte{val}}
	}
}

func (r *recordingFormatter) WriteJSON(val []byte) {
	r.values = append(r.values, &JSONNamedValue{Key: r.key, Val: val})
}

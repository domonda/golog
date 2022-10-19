package golog

import (
	"fmt"
	"time"
)

// valueRecorder implements the Formatter interface
// and records all logged values that can be retrieved
// with the Values() method.
type valueRecorder struct {
	currentKey   string
	writingSlice bool
	currentSlice Value

	recordedValues Values
}

// NewValueRecorder returns a Formatter that records all logged values
// instead of formatting and printing them somewhere.
// The recorded values can be retrieved with the Values() method.
func NewValueRecorder() *valueRecorder { return new(valueRecorder) }

// Values returns the recorded values
func (r *valueRecorder) Values() Values {
	return r.recordedValues
}

func (r *valueRecorder) Clone(level Level) Formatter {
	panic("Clone not supported by golog.valueRecorder")
}

func (r *valueRecorder) BeginMessage(t time.Time, levels *Levels, level Level, prefix, text string) {
	panic("BeginMessage not supported by golog.valueRecorder")
}

func (r *valueRecorder) FinishMessage() {
	panic("calling golog.Message.Log() after golog.Logger.With() is not valid, call golog.Message.SubLogger() instead")
}

func (r *valueRecorder) FlushUnderlying() {}

func (r *valueRecorder) String() string {
	return fmt.Sprintf("golog.valueRecorder with %d recored values", r.recordedValues)
}

func (r *valueRecorder) WriteKey(key string) {
	r.currentKey = key
}

func (r *valueRecorder) WriteSliceKey(key string) {
	r.currentKey = key
	r.writingSlice = true
}

func (r *valueRecorder) WriteSliceEnd() {
	r.recordedValues = append(r.recordedValues, r.currentSlice)
	r.currentSlice = nil
	r.writingSlice = false
}

func (r *valueRecorder) WriteNil() {
	r.recordedValues = append(r.recordedValues, NewNilValue(r.currentKey))
}

func (r *valueRecorder) WriteBool(val bool) {
	if r.writingSlice {
		if slice, ok := r.currentSlice.(*BoolsValue); ok {
			slice.Vals = append(slice.Vals, val)
		} else {
			r.currentSlice = NewBoolsValue(r.currentKey, val)
		}
	} else {
		r.recordedValues = append(r.recordedValues, NewBoolValue(r.currentKey, val))
	}
}

func (r *valueRecorder) WriteInt(val int64) {
	if r.writingSlice {
		if slice, ok := r.currentSlice.(*IntsValue); ok {
			slice.Vals = append(slice.Vals, val)
		} else {
			r.currentSlice = NewIntsValue(r.currentKey, val)
		}
	} else {
		r.recordedValues = append(r.recordedValues, NewIntValue(r.currentKey, val))
	}
}

func (r *valueRecorder) WriteUint(val uint64) {
	if r.writingSlice {
		if slice, ok := r.currentSlice.(*UintsValue); ok {
			slice.Vals = append(slice.Vals, val)
		} else {
			r.currentSlice = NewUintsValue(r.currentKey, val)
		}
	} else {
		r.recordedValues = append(r.recordedValues, NewUintValue(r.currentKey, val))
	}
}

func (r *valueRecorder) WriteFloat(val float64) {
	if r.writingSlice {
		if slice, ok := r.currentSlice.(*FloatsValue); ok {
			slice.Vals = append(slice.Vals, val)
		} else {
			r.currentSlice = NewFloatsValue(r.currentKey, val)
		}
	} else {
		r.recordedValues = append(r.recordedValues, NewFloatValue(r.currentKey, val))
	}
}

func (r *valueRecorder) WriteString(val string) {
	if r.writingSlice {
		if slice, ok := r.currentSlice.(*StringsValue); ok {
			slice.Vals = append(slice.Vals, val)
		} else {
			r.currentSlice = NewStringsValue(r.currentKey, val)
		}
	} else {
		r.recordedValues = append(r.recordedValues, NewStringValue(r.currentKey, val))
	}
}

func (r *valueRecorder) WriteError(val error) {
	if r.writingSlice {
		if slice, ok := r.currentSlice.(*ErrorsValue); ok {
			slice.Vals = append(slice.Vals, val)
		} else {
			r.currentSlice = NewErrorsValue(r.currentKey, val)
		}
	} else {
		r.recordedValues = append(r.recordedValues, NewErrorValue(r.currentKey, val))
	}
}

func (r *valueRecorder) WriteUUID(val [16]byte) {
	if r.writingSlice {
		if slice, ok := r.currentSlice.(*UUIDsValue); ok {
			slice.Vals = append(slice.Vals, val)
		} else {
			r.currentSlice = NewUUIDsValue(r.currentKey, val)
		}
	} else {
		r.recordedValues = append(r.recordedValues, NewUUIDValue(r.currentKey, val))
	}
}

func (r *valueRecorder) WriteJSON(val []byte) {
	r.recordedValues = append(r.recordedValues, NewJSONValue(r.currentKey, val))
}

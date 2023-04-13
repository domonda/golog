package golog

import (
	"fmt"
	"time"
)

var (
	_ Writer       = new(attribsRecorder)
	_ fmt.Stringer = new(attribsRecorder)
)

// attribsRecorder implements the Writer interface
// and records all logged attributes that can be retrieved
// with the Attribs() method.
type attribsRecorder struct {
	currentKey   string
	writingSlice bool
	currentSlice Attrib

	recorded Attribs
}

// NewAttribsRecorder returns a Writer that records all logged attributes
// instead of formatting and printing them somewhere.
// The recorded values can be retrieved with the Attrs() method.
func NewAttribsRecorder() *attribsRecorder { return new(attribsRecorder) }

// Attribs returns the recorded values
func (r *attribsRecorder) Attribs() Attribs {
	return r.recorded
}

func (r *attribsRecorder) BeginMessage(logger *Logger, t time.Time, level Level, text string) Writer {
	panic("BeginMessage not supported by golog.attribsRecorder")
}

func (r *attribsRecorder) CommitMessage() {
	panic("calling golog.Message.Log() after golog.Logger.With() is not valid, call golog.Message.SubLogger() instead")
}

func (r *attribsRecorder) FlushUnderlying() {}

func (r *attribsRecorder) String() string {
	return fmt.Sprintf("golog.attribsRecorder with %d recored attribs", r.recorded)
}

func (r *attribsRecorder) WriteKey(key string) {
	r.currentKey = key
}

func (r *attribsRecorder) WriteSliceKey(key string) {
	r.currentKey = key
	r.writingSlice = true
}

func (r *attribsRecorder) WriteSliceEnd() {
	r.recorded = append(r.recorded, r.currentSlice)
	r.currentSlice = nil
	r.writingSlice = false
}

func (r *attribsRecorder) WriteNil() {
	r.recorded = append(r.recorded, Nil{r.currentKey})
}

func (r *attribsRecorder) WriteBool(val bool) {
	if r.writingSlice {
		// Need pointer type to be able to modify slice.Vals
		if slice, ok := r.currentSlice.(*Bools); ok {
			slice.Vals = append(slice.Vals, val)
		} else {
			r.currentSlice = &Bools{r.currentKey, []bool{val}}
		}
	} else {
		r.recorded = append(r.recorded, Bool{r.currentKey, val})
	}
}

func (r *attribsRecorder) WriteInt(val int64) {
	if r.writingSlice {
		// Need pointer type to be able to modify slice.Vals
		if slice, ok := r.currentSlice.(*Ints); ok {
			slice.Vals = append(slice.Vals, val)
		} else {
			r.currentSlice = &Ints{r.currentKey, []int64{val}}
		}
	} else {
		r.recorded = append(r.recorded, Int{r.currentKey, val})
	}
}

func (r *attribsRecorder) WriteUint(val uint64) {
	if r.writingSlice {
		// Need pointer type to be able to modify slice.Vals
		if slice, ok := r.currentSlice.(*Uints); ok {
			slice.Vals = append(slice.Vals, val)
		} else {
			r.currentSlice = &Uints{r.currentKey, []uint64{val}}
		}
	} else {
		r.recorded = append(r.recorded, Uint{r.currentKey, val})
	}
}

func (r *attribsRecorder) WriteFloat(val float64) {
	if r.writingSlice {
		// Need pointer type to be able to modify slice.Vals
		if slice, ok := r.currentSlice.(*Floats); ok {
			slice.Vals = append(slice.Vals, val)
		} else {
			r.currentSlice = &Floats{r.currentKey, []float64{val}}
		}
	} else {
		r.recorded = append(r.recorded, Float{r.currentKey, val})
	}
}

func (r *attribsRecorder) WriteString(val string) {
	if r.writingSlice {
		// Need pointer type to be able to modify slice.Vals
		if slice, ok := r.currentSlice.(*Strings); ok {
			slice.Vals = append(slice.Vals, val)
		} else {
			r.currentSlice = &Strings{r.currentKey, []string{val}}
		}
	} else {
		r.recorded = append(r.recorded, String{r.currentKey, val})
	}
}

func (r *attribsRecorder) WriteError(val error) {
	if r.writingSlice {
		// Need pointer type to be able to modify slice.Vals
		if slice, ok := r.currentSlice.(*Errors); ok {
			slice.Vals = append(slice.Vals, val)
		} else {
			r.currentSlice = &Errors{r.currentKey, []error{val}}
		}
	} else {
		r.recorded = append(r.recorded, Error{r.currentKey, val})
	}
}

func (r *attribsRecorder) WriteUUID(val [16]byte) {
	if r.writingSlice {
		// Need pointer type to be able to modify slice.Vals
		if slice, ok := r.currentSlice.(*UUIDs); ok {
			slice.Vals = append(slice.Vals, val)
		} else {
			r.currentSlice = &UUIDs{r.currentKey, [][16]byte{val}}
		}
	} else {
		r.recorded = append(r.recorded, UUID{r.currentKey, val})
	}
}

func (r *attribsRecorder) WriteJSON(val []byte) {
	r.recorded = append(r.recorded, JSON{r.currentKey, val})
}

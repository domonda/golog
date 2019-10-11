package golog

import (
	"fmt"
	"time"
)

type recordingFormatter struct {
	hooks     []Hook
	key       string
	slice     bool
	sliceHook Hook
}

func (r *recordingFormatter) Clone() Formatter {
	panic("should never happen")
}

func (r *recordingFormatter) WriteMsg(t time.Time, levels *Levels, level Level, msg string) {
	panic("should never happen")
}

func (r *recordingFormatter) FlushAndFree() {
	panic("calling golog.Message.Log() after Logger.Record() is invalid, call Message.NewLogger() instead")
}

// String is here only for debugging
func (r *recordingFormatter) String() string {
	return fmt.Sprintf("recordingFormatter with %d recored hooks", r.hooks)
}

func (r *recordingFormatter) WriteKey(key string) {
	r.key = key
}

func (r *recordingFormatter) WriteSliceKey(key string) {
	r.key = key
	r.slice = true
}

func (r *recordingFormatter) WriteSliceEnd() {
	r.hooks = append(r.hooks, r.sliceHook)
	r.sliceHook = nil
	r.slice = false
}

func (r *recordingFormatter) WriteNil() {
	r.hooks = append(r.hooks, &NilHook{Key: r.key})
}

func (r *recordingFormatter) WriteBool(val bool) {
	if !r.slice {
		r.hooks = append(r.hooks, &BoolHook{Key: r.key, Val: val})
		return
	}
	if hook, ok := r.sliceHook.(*BoolsHook); ok {
		hook.Vals = append(hook.Vals, val)
	} else {
		r.sliceHook = &BoolsHook{Key: r.key, Vals: []bool{val}}
	}
}

func (r *recordingFormatter) WriteInt(val int64) {
	if !r.slice {
		r.hooks = append(r.hooks, &IntHook{Key: r.key, Val: val})
		return
	}
	if hook, ok := r.sliceHook.(*IntsHook); ok {
		hook.Vals = append(hook.Vals, val)
	} else {
		r.sliceHook = &IntsHook{Key: r.key, Vals: []int64{val}}
	}
}

func (r *recordingFormatter) WriteUint(val uint64) {
	if !r.slice {
		r.hooks = append(r.hooks, &UintHook{Key: r.key, Val: val})
		return
	}
	if hook, ok := r.sliceHook.(*UintsHook); ok {
		hook.Vals = append(hook.Vals, val)
	} else {
		r.sliceHook = &UintsHook{Key: r.key, Vals: []uint64{val}}
	}
}

func (r *recordingFormatter) WriteFloat(val float64) {
	if !r.slice {
		r.hooks = append(r.hooks, &FloatHook{Key: r.key, Val: val})
		return
	}
	if hook, ok := r.sliceHook.(*FloatsHook); ok {
		hook.Vals = append(hook.Vals, val)
	} else {
		r.sliceHook = &FloatsHook{Key: r.key, Vals: []float64{val}}
	}
}

func (r *recordingFormatter) WriteString(val string) {
	if !r.slice {
		r.hooks = append(r.hooks, &StringHook{Key: r.key, Val: val})
		return
	}
	if hook, ok := r.sliceHook.(*StringsHook); ok {
		hook.Vals = append(hook.Vals, val)
	} else {
		r.sliceHook = &StringsHook{Key: r.key, Vals: []string{val}}
	}
}

func (r *recordingFormatter) WriteError(val error) {
	if !r.slice {
		r.hooks = append(r.hooks, &ErrorHook{Key: r.key, Val: val})
		return
	}
	if hook, ok := r.sliceHook.(*ErrorsHook); ok {
		hook.Vals = append(hook.Vals, val)
	} else {
		r.sliceHook = &ErrorsHook{Key: r.key, Vals: []error{val}}
	}
}

func (r *recordingFormatter) WriteUUID(val [16]byte) {
	if !r.slice {
		r.hooks = append(r.hooks, &UUIDHook{Key: r.key, Val: val})
		return
	}
	if hook, ok := r.sliceHook.(*UUIDsHook); ok {
		hook.Vals = append(hook.Vals, val)
	} else {
		r.sliceHook = &UUIDsHook{Key: r.key, Vals: [][16]byte{val}}
	}
}

func (r *recordingFormatter) WriteJSON(val []byte) {
	r.hooks = append(r.hooks, &JSONHook{Key: r.key, Val: val})
}

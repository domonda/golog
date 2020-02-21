package golog

import "time"

type FilteredFormatter struct {
	filter  LevelFilter
	wrapped Formatter
}

// NewFilteredFormatter wraps another Formatter that will only be
// used for messages with a level that is active with the passed filter.
func NewFilteredFormatter(filter LevelFilter, wrapped Formatter) *FilteredFormatter {
	return &FilteredFormatter{
		filter:  filter,
		wrapped: wrapped,
	}
}

func (f *FilteredFormatter) Clone(level Level) Formatter {
	if !f.filter.IsActive(level) {
		return NopFormatter
	}
	return NewFilteredFormatter(f.filter, f.wrapped.Clone(level))
}

func (f *FilteredFormatter) WriteText(t time.Time, levels *Levels, level Level, prefix, text string) {
	f.wrapped.WriteText(t, levels, level, prefix, text)
}

func (f *FilteredFormatter) FlushAndFree() {
	f.wrapped.FlushAndFree()
}

func (f *FilteredFormatter) FlushUnderlying() {
	f.wrapped.FlushUnderlying()
}

// String is here only for debugging
func (f *FilteredFormatter) String() string {
	return f.wrapped.String()
}

func (f *FilteredFormatter) WriteKey(key string) {
	f.wrapped.WriteKey(key)
}

func (f *FilteredFormatter) WriteSliceKey(key string) {
	f.wrapped.WriteSliceKey(key)
}

func (f *FilteredFormatter) WriteSliceEnd() {
	f.wrapped.WriteSliceEnd()
}

func (f *FilteredFormatter) WriteNil() {
	f.wrapped.WriteNil()
}

func (f *FilteredFormatter) WriteBool(val bool) {
	f.wrapped.WriteBool(val)
}

func (f *FilteredFormatter) WriteInt(val int64) {
	f.wrapped.WriteInt(val)
}

func (f *FilteredFormatter) WriteUint(val uint64) {
	f.wrapped.WriteUint(val)
}

func (f *FilteredFormatter) WriteFloat(val float64) {
	f.wrapped.WriteFloat(val)
}

func (f *FilteredFormatter) WriteString(val string) {
	f.wrapped.WriteString(val)
}

func (f *FilteredFormatter) WriteError(val error) {
	f.wrapped.WriteError(val)
}

func (f *FilteredFormatter) WriteUUID(val [16]byte) {
	f.wrapped.WriteUUID(val)
}

func (f *FilteredFormatter) WriteJSON(val []byte) {
	f.wrapped.WriteJSON(val)
}

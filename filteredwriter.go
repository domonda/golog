package golog

import "time"

type FilteredWriter struct {
	filter  LevelFilter
	wrapped Writer
}

// NewFilteredWriter wraps another Writer that will only be
// used for messages with a level that is active with the passed filter.
func NewFilteredWriter(filter LevelFilter, wrapped Writer) *FilteredWriter {
	return &FilteredWriter{
		filter:  filter,
		wrapped: wrapped,
	}
}

func (f *FilteredWriter) Clone(level Level) Writer {
	if !f.filter.IsActive(level) {
		return NopWriter
	}
	return NewFilteredWriter(f.filter, f.wrapped.Clone(level))
}

func (f *FilteredWriter) BeginMessage(t time.Time, levels *Levels, level Level, prefix, text string) {
	f.wrapped.BeginMessage(t, levels, level, prefix, text)
}

func (f *FilteredWriter) FinishMessage() {
	f.wrapped.FinishMessage()
}

func (f *FilteredWriter) FlushUnderlying() {
	f.wrapped.FlushUnderlying()
}

func (f *FilteredWriter) String() string {
	return f.wrapped.String()
}

func (f *FilteredWriter) WriteKey(key string) {
	f.wrapped.WriteKey(key)
}

func (f *FilteredWriter) WriteSliceKey(key string) {
	f.wrapped.WriteSliceKey(key)
}

func (f *FilteredWriter) WriteSliceEnd() {
	f.wrapped.WriteSliceEnd()
}

func (f *FilteredWriter) WriteNil() {
	f.wrapped.WriteNil()
}

func (f *FilteredWriter) WriteBool(val bool) {
	f.wrapped.WriteBool(val)
}

func (f *FilteredWriter) WriteInt(val int64) {
	f.wrapped.WriteInt(val)
}

func (f *FilteredWriter) WriteUint(val uint64) {
	f.wrapped.WriteUint(val)
}

func (f *FilteredWriter) WriteFloat(val float64) {
	f.wrapped.WriteFloat(val)
}

func (f *FilteredWriter) WriteString(val string) {
	f.wrapped.WriteString(val)
}

func (f *FilteredWriter) WriteError(val error) {
	f.wrapped.WriteError(val)
}

func (f *FilteredWriter) WriteUUID(val [16]byte) {
	f.wrapped.WriteUUID(val)
}

func (f *FilteredWriter) WriteJSON(val []byte) {
	f.wrapped.WriteJSON(val)
}

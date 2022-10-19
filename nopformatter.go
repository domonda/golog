package golog

import "time"

// NopFormatter is a Formatter that does nothing (no operation)
const NopFormatter nopFormatter = 0

type nopFormatter int

func (f nopFormatter) Clone(level Level) Formatter {
	return f
}

func (nopFormatter) WriteText(t time.Time, levels *Levels, level Level, prefix, text string) {}

func (nopFormatter) FlushAndFree() {}

func (nopFormatter) FlushUnderlying() {}

func (nopFormatter) String() string {
	return "nopFormatter"
}

func (nopFormatter) WriteKey(key string) {}

func (nopFormatter) WriteSliceKey(key string) {}

func (nopFormatter) WriteSliceEnd() {}

func (nopFormatter) WriteNil() {}

func (nopFormatter) WriteBool(val bool) {}

func (nopFormatter) WriteInt(val int64) {}

func (nopFormatter) WriteUint(val uint64) {}

func (nopFormatter) WriteFloat(val float64) {}

func (nopFormatter) WriteString(val string) {}

func (nopFormatter) WriteError(val error) {}

func (nopFormatter) WriteUUID(val [16]byte) {}

func (nopFormatter) WriteJSON(val []byte) {}

package logsentry

import (
	"time"

	"github.com/getsentry/sentry-go"

	"github.com/domonda/golog"
)

// UnknownLevel will be used if a golog.Level
// can't be mapped to a sentry.LevelError.
var UnknownLevel = sentry.LevelError

type Formatter struct {
	filter  golog.LevelFilter
	hub     *sentry.Hub
	level   sentry.Level
	message string
	extra   map[string]interface{}
	key     string
	slice   []interface{}
}

func NewFormatter(filter golog.LevelFilter, hub *sentry.Hub) *Formatter {
	return &Formatter{
		filter: filter,
		hub:    hub,
		extra:  make(map[string]interface{}),
	}
}

func (f *Formatter) Clone(level golog.Level) golog.Formatter {
	if !f.filter.IsActive(level) {
		return golog.NopFormatter
	}
	return NewFormatter(f.filter, f.hub) // Clone hub too?
}

func (f *Formatter) WriteText(t time.Time, levels *golog.Levels, level golog.Level, prefix, text string) {
	switch level {
	case levels.Fatal:
		f.level = sentry.LevelFatal
	case levels.Error:
		f.level = sentry.LevelError
	case levels.Warn:
		f.level = sentry.LevelWarning
	case levels.Info:
		f.level = sentry.LevelInfo
	case levels.Debug:
		f.level = sentry.LevelDebug
	case levels.Trace:
		f.level = sentry.LevelDebug
	default:
		f.level = UnknownLevel
	}

	f.message = prefix + text
}

func (f *Formatter) FlushAndFree() {
	// Flush
	event := sentry.NewEvent()
	event.Level = f.level
	event.Message = f.message
	event.Extra = f.extra

	f.hub.CaptureEvent(event)

	// Free
	f.extra = nil
	f.slice = nil
	f.hub = nil
}

// String is here only for debugging
func (f *Formatter) String() string {
	return f.message
}

func (f *Formatter) WriteKey(key string) {
	f.key = key
}

func (f *Formatter) WriteSliceKey(key string) {
	f.key = key
	f.slice = make([]interface{}, 0)
}

func (f *Formatter) WriteSliceEnd() {
	f.extra[f.key] = f.slice
	f.slice = nil
}

func (f *Formatter) writeVal(val interface{}) {
	if f.slice != nil {
		f.slice = append(f.slice, val)
	} else {
		f.extra[f.key] = val
	}
}

func (f *Formatter) WriteNil() {
	f.writeVal(nil)
}

func (f *Formatter) WriteBool(val bool) {
	f.writeVal(val)
}

func (f *Formatter) WriteInt(val int64) {
	f.writeVal(val)
}

func (f *Formatter) WriteUint(val uint64) {
	f.writeVal(val)
}

func (f *Formatter) WriteFloat(val float64) {
	f.writeVal(val)
}

func (f *Formatter) WriteString(val string) {
	f.writeVal(val)
}

func (f *Formatter) WriteError(val error) {
	f.writeVal(val)
}

func (f *Formatter) WriteUUID(val [16]byte) {
	f.writeVal(val)
}

func (f *Formatter) WriteJSON(val []byte) {
	f.writeVal(val)
}

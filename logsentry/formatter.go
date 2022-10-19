package logsentry

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"

	"github.com/domonda/golog"
)

var (
	// UnknownLevel will be used if a golog.Level
	// can't be mapped to a sentry.LevelError.
	UnknownLevel = sentry.LevelError

	FlushTimeout time.Duration = 3 * time.Second
)

// Compile time check Formatter implements interface golog.Formatter
var _ golog.Formatter = new(Formatter)

type Formatter struct {
	hub       *sentry.Hub
	filter    golog.LevelFilter
	timestamp time.Time
	level     sentry.Level
	message   strings.Builder
	valsAsMsg bool
	extra     map[string]any
	values    map[string]any
	key       string
	slice     []any
}

// NewFormatter returns a new Formatter for a sentry.Hub.
// Any values passed as extra will be added to every log messsage.
func NewFormatter(hub *sentry.Hub, filter golog.LevelFilter, valsAsMsg bool, extra map[string]any) *Formatter {
	return &Formatter{
		filter:    filter,
		hub:       hub,
		valsAsMsg: valsAsMsg,
		extra:     extra,
	}
}

func (f *Formatter) Clone(level golog.Level) golog.Formatter {
	if !f.filter.IsActive(level) {
		return golog.NopFormatter
	}
	return NewFormatter(f.hub, f.filter, f.valsAsMsg, f.extra) // Clone hub too?
}

func (f *Formatter) WriteText(t time.Time, levels *golog.Levels, level golog.Level, prefix, text string) {
	f.timestamp = t

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

	f.message.WriteString(prefix)
	f.message.WriteString(text)
}

func (f *Formatter) FlushAndFree() {
	// Flush f.message
	if f.message.Len() > 0 {
		event := sentry.NewEvent()
		event.Timestamp = f.timestamp
		event.Level = f.level
		event.Message = f.message.String()
		event.Fingerprint = []string{event.Message}
		for key, val := range f.extra {
			event.Extra[key] = val
		}
		for key, val := range f.values {
			event.Extra[key] = val
		}
		if f.hub.Client().Options().AttachStacktrace {
			stackTrace := sentry.NewStacktrace()
			stackTrace.Frames = filterFrames(stackTrace.Frames)
			event.Threads = []sentry.Thread{{
				Stacktrace: stackTrace,
				Current:    true,
			}}
		}
		f.hub.CaptureEvent(event)
	}

	// Free pointers
	f.message.Reset()
	if f.values != nil {
		valueMapPool.Put(f.values)
		f.values = nil
	}
	f.slice = nil
	f.hub = nil
}

func (f *Formatter) FlushUnderlying() {
	f.hub.Flush(FlushTimeout)
}

func filterFrames(frames []sentry.Frame) []sentry.Frame {
	filtered := make([]sentry.Frame, 0, len(frames))
	for _, frame := range frames {
		if !strings.HasPrefix(frame.Module, "github.com/domonda/golog") {
			filtered = append(filtered, frame)
		}
	}
	return filtered
}

func (f *Formatter) String() string {
	return f.message.String()
}

func (f *Formatter) WriteKey(key string) {
	f.key = key

	if f.valsAsMsg {
		fmt.Fprintf(&f.message, " %s=", key)
	}
}

func (f *Formatter) WriteSliceKey(key string) {
	f.key = key
	f.slice = make([]any, 0)

	if f.valsAsMsg {
		fmt.Fprintf(&f.message, " %s=[", key)
	}
}

func (f *Formatter) WriteSliceEnd() {
	f.writeFinalVal(f.slice)
	f.slice = nil

	if f.valsAsMsg {
		f.message.WriteByte(']')
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
	f.writeVal(val.Error())
}

func (f *Formatter) WriteUUID(val [16]byte) {
	f.writeVal(golog.FormatUUID(val))
}

func (f *Formatter) WriteJSON(val []byte) {
	f.writeVal(json.RawMessage(val))
}

func (f *Formatter) writeVal(val any) {
	if f.slice != nil {
		f.slice = append(f.slice, val)
	} else {
		f.writeFinalVal(val)
	}

	if f.valsAsMsg {
		if len(f.slice) > 1 {
			f.message.WriteByte(',')
		}
		switch x := val.(type) {
		case json.RawMessage:
			f.message.Write(x)
		case string:
			fmt.Fprintf(&f.message, "%q", val)
		default:
			fmt.Fprintf(&f.message, "%v", val)
		}
	}
}

var valueMapPool sync.Pool

func (f *Formatter) writeFinalVal(val any) {
	if f.values != nil {
		f.values[f.key] = val
		return
	}
	if m, _ := valueMapPool.Get().(map[string]any); m != nil {
		for k := range m {
			delete(m, k)
		}
		m[f.key] = val
		f.values = m
	} else {
		f.values = map[string]any{f.key: val}
	}
}

package logsentry

import (
	"encoding/json"
	"fmt"
	"strings"
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
	filter    golog.LevelFilter
	hub       *sentry.Hub
	level     sentry.Level
	message   strings.Builder
	valsAsMsg bool
	extra     map[string]interface{}
	key       string
	slice     []interface{}
}

func NewFormatter(filter golog.LevelFilter, hub *sentry.Hub, valsAsMsg bool) *Formatter {
	return &Formatter{
		filter:    filter,
		hub:       hub,
		extra:     make(map[string]interface{}),
		valsAsMsg: valsAsMsg,
	}
}

func (f *Formatter) Clone(level golog.Level) golog.Formatter {
	if !f.filter.IsActive(level) {
		return golog.NopFormatter
	}
	return NewFormatter(f.filter, f.hub, f.valsAsMsg) // Clone hub too?
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

	f.message.WriteString(prefix)
	f.message.WriteString(text)
}

func (f *Formatter) FlushAndFree() {
	// Flush
	event := sentry.NewEvent()
	event.Level = f.level
	event.Message = f.message.String()
	event.Extra = f.extra
	if f.hub.Client().Options().AttachStacktrace {
		stackTrace := sentry.NewStacktrace()
		stackTrace.Frames = filterFrames(stackTrace.Frames)
		event.Threads = []sentry.Thread{{
			Stacktrace: stackTrace,
			Current:    true,
		}}
	}
	f.hub.CaptureEvent(event)

	// Free
	f.extra = nil
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

// String is here only for debugging
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
	f.slice = make([]interface{}, 0)

	if f.valsAsMsg {
		fmt.Fprintf(&f.message, " %s=[", key)
	}
}

func (f *Formatter) WriteSliceEnd() {
	f.extra[f.key] = f.slice
	f.slice = nil

	if f.valsAsMsg {
		f.message.WriteByte(']')
	}
}

func (f *Formatter) writeVal(val interface{}) {
	if f.slice != nil {
		f.slice = append(f.slice, val)
	} else {
		f.extra[f.key] = val
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

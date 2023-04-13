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

// Writer implements interface golog.Writer
var _ golog.Writer = new(Writer)

type Writer struct {
	hub       *sentry.Hub
	format    *golog.Format
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

// NewWriter returns a new Writer for a sentry.Hub.
// Any values passed as extra will be added to every log messsage.
func NewWriter(hub *sentry.Hub, format *golog.Format, filter golog.LevelFilter, valsAsMsg bool, extra map[string]any) *Writer {
	return &Writer{
		hub:       hub,
		format:    format,
		filter:    filter,
		valsAsMsg: valsAsMsg,
		extra:     extra,
	}
}

func (w *Writer) BeginMessage(logger *golog.Logger, t time.Time, level golog.Level, text string) golog.Writer {
	if !w.filter.IsActive(level) {
		return golog.NopWriter
	}
	next := NewWriter(w.hub, w.format, w.filter, w.valsAsMsg, w.extra) // Clone hub too?
	next.beginWriteMessage(logger, t, level, text)
	return next
}

func (w *Writer) beginWriteMessage(logger *golog.Logger, t time.Time, level golog.Level, text string) {
	w.timestamp = t

	levels := logger.Config().Levels()
	switch level {
	case levels.Fatal:
		w.level = sentry.LevelFatal
	case levels.Error:
		w.level = sentry.LevelError
	case levels.Warn:
		w.level = sentry.LevelWarning
	case levels.Info:
		w.level = sentry.LevelInfo
	case levels.Debug:
		w.level = sentry.LevelDebug
	case levels.Trace:
		w.level = sentry.LevelDebug
	default:
		w.level = UnknownLevel
	}

	if prefix := logger.Prefix(); prefix != "" {
		w.message.WriteString(prefix)
		w.message.WriteString(w.format.PrefixSep)
	}
	w.message.WriteString(text)
}

func (w *Writer) CommitMessage() {
	// Flush w.message
	if w.message.Len() > 0 {
		event := sentry.NewEvent()
		event.Timestamp = w.timestamp
		event.Level = w.level
		event.Message = w.message.String()
		event.Fingerprint = []string{event.Message}
		for key, val := range w.extra {
			event.Extra[key] = val
		}
		for key, val := range w.values {
			event.Extra[key] = val
		}
		if w.hub.Client().Options().AttachStacktrace {
			stackTrace := sentry.NewStacktrace()
			stackTrace.Frames = filterFrames(stackTrace.Frames)
			event.Threads = []sentry.Thread{{
				Stacktrace: stackTrace,
				Current:    true,
			}}
		}
		w.hub.CaptureEvent(event)
	}

	// Free pointers
	w.message.Reset()
	if w.values != nil {
		valueMapPool.Put(w.values)
		w.values = nil
	}
	w.slice = nil
	w.hub = nil
}

func (w *Writer) FlushUnderlying() {
	w.hub.Flush(FlushTimeout)
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

func (w *Writer) String() string {
	return w.message.String()
}

func (w *Writer) WriteKey(key string) {
	w.key = key

	if w.valsAsMsg {
		fmt.Fprintf(&w.message, " %s=", key)
	}
}

func (w *Writer) WriteSliceKey(key string) {
	w.key = key
	w.slice = make([]any, 0)

	if w.valsAsMsg {
		fmt.Fprintf(&w.message, " %s=[", key)
	}
}

func (w *Writer) WriteSliceEnd() {
	w.writeFinalVal(w.slice)
	w.slice = nil

	if w.valsAsMsg {
		w.message.WriteByte(']')
	}
}

func (w *Writer) WriteNil() {
	w.writeVal(nil)
}

func (w *Writer) WriteBool(val bool) {
	w.writeVal(val)
}

func (w *Writer) WriteInt(val int64) {
	w.writeVal(val)
}

func (w *Writer) WriteUint(val uint64) {
	w.writeVal(val)
}

func (w *Writer) WriteFloat(val float64) {
	w.writeVal(val)
}

func (w *Writer) WriteString(val string) {
	w.writeVal(val)
}

func (w *Writer) WriteError(val error) {
	w.writeVal(val.Error())
}

func (w *Writer) WriteUUID(val [16]byte) {
	w.writeVal(golog.FormatUUID(val))
}

func (w *Writer) WriteJSON(val []byte) {
	w.writeVal(json.RawMessage(val))
}

func (w *Writer) writeVal(val any) {
	if w.slice != nil {
		w.slice = append(w.slice, val)
	} else {
		w.writeFinalVal(val)
	}

	if w.valsAsMsg {
		if len(w.slice) > 1 {
			w.message.WriteByte(',')
		}
		switch x := val.(type) {
		case json.RawMessage:
			w.message.Write(x)
		case string:
			fmt.Fprintf(&w.message, "%q", val)
		default:
			fmt.Fprintf(&w.message, "%v", val)
		}
	}
}

var valueMapPool sync.Pool

func (w *Writer) writeFinalVal(val any) {
	if w.values != nil {
		w.values[w.key] = val
		return
	}
	if m, _ := valueMapPool.Get().(map[string]any); m != nil {
		for k := range m {
			delete(m, k)
		}
		m[w.key] = val
		w.values = m
	} else {
		w.values = map[string]any{w.key: val}
	}
}

package logsentry

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"

	"github.com/domonda/golog"
)

var (
	_ golog.Writer       = new(Writer)
	_ golog.WriterConfig = new(WriterConfig)
)

type WriterConfig struct {
	hub        *sentry.Hub
	format     *golog.Format
	filter     golog.LevelFilter
	valsAsMsg  bool
	extra      map[string]any
	writerPool sync.Pool
}

// NewWriterConfig returns a new WriterConfig for a sentry.Hub.
// Any values passed as extra will be added to every log messsage.
func NewWriterConfig(hub *sentry.Hub, format *golog.Format, filter golog.LevelFilter, valsAsMsg bool, extra map[string]any) *WriterConfig {
	return &WriterConfig{
		hub:       hub,
		format:    format,
		filter:    filter,
		valsAsMsg: valsAsMsg,
		extra:     extra,
	}
}

func (c *WriterConfig) WriterForNewMessage(ctx context.Context, level golog.Level) golog.Writer {
	if c.filter.IsInactive(ctx, level) || IsContextWithoutLogging(ctx) {
		return nil
	}
	if w, _ := c.writerPool.Get().(golog.Writer); w != nil {
		return w
	}
	return &Writer{config: c}
}

func (c *WriterConfig) FlushUnderlying() {
	c.hub.Flush(FlushTimeout)
}

///////////////////////////////////////////////////////////////////////////////

type Writer struct {
	config    *WriterConfig
	timestamp time.Time
	level     sentry.Level
	message   strings.Builder
	values    map[string]any
	key       string
	slice     []any
}

func (w *Writer) BeginMessage(config golog.Config, t time.Time, level golog.Level, prefix, text string) {
	w.timestamp = t

	levels := config.Levels()
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

	if prefix != "" {
		w.message.WriteString(prefix)
		w.message.WriteString(w.config.format.PrefixSep)
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
		for key, val := range w.config.extra {
			event.Extra[key] = val
		}
		for key, val := range w.values {
			event.Extra[key] = val
		}
		if w.config.hub.Client().Options().AttachStacktrace {
			stackTrace := sentry.NewStacktrace()
			stackTrace.Frames = filterFrames(stackTrace.Frames)
			event.Threads = []sentry.Thread{{
				Stacktrace: stackTrace,
				Current:    true,
			}}
		}
		w.config.hub.CaptureEvent(event)
	}

	// Reset and return to pool
	w.message.Reset()
	if w.values != nil {
		valueMapPool.Put(w.values)
		w.values = nil
	}
	w.slice = nil
	w.config.writerPool.Put(w)
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

	if w.config.valsAsMsg {
		fmt.Fprintf(&w.message, " %s=", key)
	}
}

func (w *Writer) WriteSliceKey(key string) {
	w.key = key
	w.slice = make([]any, 0)

	if w.config.valsAsMsg {
		fmt.Fprintf(&w.message, " %s=[", key)
	}
}

func (w *Writer) WriteSliceEnd() {
	w.writeFinalVal(w.slice)
	w.slice = nil

	if w.config.valsAsMsg {
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

	if w.config.valsAsMsg {
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

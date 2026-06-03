package logsentry

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
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

// WriterConfig implements golog.WriterConfig and serves as a factory for
// creating Writer instances that send log messages to Sentry.
//
// It maintains configuration for Sentry integration including the Sentry hub,
// message formatting, level filtering, and object pooling for efficient
// memory usage. WriterConfig instances are typically created once and reused
// across multiple log operations.
//
// The config determines:
//   - Which Sentry project receives events (via hub)
//   - How messages are formatted (via format)
//   - Which log levels are sent to Sentry (via filter)
//   - Whether values appear in message text (via valsAsMsg)
//   - Additional metadata included with every event (via extra)
//   - Where errors that occur while logging to Sentry are reported
//     (via [WithErrorHandler]; defaults to [golog.ErrorHandler])
//
// Example usage:
//
//	config := logsentry.NewWriterConfig(
//	    sentry.CurrentHub(),
//	    golog.NewDefaultFormat(),
//	    golog.ErrorLevel().FilterOutBelow(),
//	    false,
//	    map[string]any{"service": "my-app"},
//	)
type WriterConfig struct {
	hub        *sentry.Hub
	format     *golog.Format
	filter     golog.LevelFilter
	valsAsMsg  bool
	extra      map[string]any
	onError    func(error)
	writerPool sync.Pool
}

// NewWriterConfig returns a new WriterConfig for a sentry.Hub.
// Any values passed as extra will be added to every log messsage.
//
// Errors that occur while writing to Sentry (recovered panics in the writer)
// are reported to the handler set via [WithErrorHandler], or to
// [golog.ErrorHandler] when no option is given. To also route Sentry's own
// asynchronous transport errors (e.g. HTTP 413, network failures) into the
// same handler, pass [NewSentryDebugWriter] as sentry.ClientOptions.DebugWriter
// (with Debug: true) when constructing the client.
func NewWriterConfig(hub *sentry.Hub, format *golog.Format, filter golog.LevelFilter, valsAsMsg bool, extra map[string]any, opts ...Option) *WriterConfig {
	if hub == nil {
		panic("logsentry.NewWriterConfig: hub must not be nil")
	}
	if format == nil {
		panic("logsentry.NewWriterConfig: format must not be nil")
	}
	c := &WriterConfig{
		hub:       hub,
		format:    format,
		filter:    filter,
		valsAsMsg: valsAsMsg,
		extra:     extra,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
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
	defer func() {
		if r := recover(); r != nil {
			c.handleError(fmt.Errorf("logsentry.WriterConfig.FlushUnderlying recovered panic: %v\n%s", r, debug.Stack()))
		}
	}()

	c.hub.Flush(FlushTimeout)
}

///////////////////////////////////////////////////////////////////////////////

// Writer implements golog.Writer and handles the actual writing of log
// messages to Sentry. It accumulates log data during the logging process
// and sends it as a Sentry event when CommitMessage() is called.
//
// Writer instances are designed to be reused through object pooling to
// minimize memory allocations and improve performance. Each Writer
// accumulates:
//   - Message text and timestamp
//   - Sentry level (mapped from golog level)
//   - Key-value pairs as a Sentry event "log" context
//   - Optional stack trace information
//
// The Writer automatically maps golog levels to Sentry levels:
//
//	TRACE/DEBUG -> DEBUG, INFO -> INFO, WARN -> WARNING, ERROR -> ERROR, FATAL -> FATAL
//
// Sentry reserves the key "type" inside every context object to denote the
// context type, so a value logged under the key "type" is remapped to "type_"
// to keep it visible as data (see [copyContextValues]).
//
// Example usage through golog:
//
//	logger.Error("Database error").Str("query", sql).Err(err).Log()
//	// Creates Sentry event with level=ERROR, message="Database error",
//	// and a "log" context: {"query": sql, "error": err.Error()}
type Writer struct {
	config    *WriterConfig
	timestamp time.Time
	level     sentry.Level
	message   strings.Builder
	values    map[string]any
	key       string
	slice     []any
}

func (w *Writer) BeginMessage(config golog.Config, timestamp time.Time, level golog.Level, prefix, text string) {
	w.timestamp = timestamp

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
		fmt.Fprintf(&w.message, w.config.format.PrefixFmt, prefix, text)
	} else {
		w.message.WriteString(text)
	}
}

// CommitMessage implements golog.Writer and finalizes the log message
// by sending it to Sentry as an event. This method is called at the end
// of each log operation after all data has been written.
//
// The method creates a Sentry event with:
//   - The accumulated message text
//   - The mapped Sentry level
//   - The original timestamp
//   - All key-value pairs as a "log" context (from both config.extra and
//     values), with the Sentry-reserved "type" key remapped to "type_"
//   - Optional stack trace (if enabled in Sentry options)
//   - A fingerprint based on the message for grouping
//
// After sending the event, the Writer is reset and returned to the object
// pool for reuse, ensuring efficient memory management.
func (w *Writer) CommitMessage() {
	defer func() {
		if r := recover(); r != nil {
			w.config.handleError(fmt.Errorf("logsentry.Writer.CommitMessage recovered panic: %v\n%s", r, debug.Stack()))
		}

		// Reset and return to pool
		w.message.Reset()
		if w.values != nil {
			valueMapPool.Put(w.values)
			w.values = nil
		}
		w.slice = nil
		w.config.writerPool.Put(w)
	}()

	// Flush w.message
	if w.message.Len() > 0 {
		event := sentry.NewEvent()
		event.Timestamp = w.timestamp
		event.Level = w.level
		event.Message = w.message.String()
		event.Fingerprint = []string{event.Message}
		// sentry-go v0.46.0 removed Event.Extra; attach the key-value pairs
		// as a named context instead (sentry.Context is map[string]any).
		logCtx := make(sentry.Context, len(w.config.extra)+len(w.values))
		copyContextValues(logCtx, w.config.extra)
		copyContextValues(logCtx, w.values)
		if len(logCtx) > 0 {
			event.Contexts["log"] = logCtx
		}
		if client := w.config.hub.Client(); client != nil && client.Options().AttachStacktrace {
			stackTrace := sentry.NewStacktrace()
			stackTrace.Frames = filterFrames(stackTrace.Frames)
			event.Threads = []sentry.Thread{{
				Stacktrace: stackTrace,
				Current:    true,
			}}
		}
		w.config.hub.CaptureEvent(event)
	}
}

const (
	// reservedContextKey is the key Sentry reserves inside every context
	// object to identify the context's type. Within our "log" context it
	// defaults to "log" when absent; a value logged under this key would be
	// consumed as the context type instead of being shown as data.
	// See https://develop.sentry.dev/sdk/data-model/event-payloads/contexts/.
	reservedContextKey = "type"

	// remappedContextKey is where a logged "type" value is stored instead, so
	// it survives as ordinary context data. The Event.Extra map removed in
	// sentry-go v0.46.0 had no reserved keys, so this collision is unique to
	// the context-based encoding.
	remappedContextKey = "type_"
)

// copyContextValues copies src into dst, remapping the Sentry-reserved
// [reservedContextKey] ("type") to [remappedContextKey] ("type_") so a golog
// value logged under "type" is preserved as data rather than swallowed by
// Sentry as the context type.
func copyContextValues(dst, src map[string]any) {
	for k, v := range src {
		if k == reservedContextKey {
			k = remappedContextKey
		}
		dst[k] = v
	}
}

// filterFrames removes golog internal frames from stack traces to provide
// cleaner Sentry debugging information by focusing on application code.
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
	if val == nil {
		w.WriteNil()
		return
	}
	w.writeVal(val.Error())
}

func (w *Writer) WriteTime(val time.Time) {
	// Format the time using the configured Format (matching JSONWriter and
	// TextWriter) so structured time values honor Format.TimeFormat and
	// Format.Location instead of sentry's default time.Time JSON marshaling.
	format := w.config.format.TimeFormat
	if format == "" {
		format = golog.DefaultTimeFormat
	}
	if w.config.format.Location != nil {
		val = val.In(w.config.format.Location)
	}
	w.writeVal(val.Format(format))
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

// valueMapPool is a global pool for reusing map[string]any instances
// to reduce memory allocations when storing key-value pairs.
var valueMapPool sync.Pool

// writeFinalVal is an internal method that stores a value as a key-value pair
// in the Writer's values map. It uses object pooling to efficiently manage
// the map instances.
//
// The method first tries to reuse an existing map from the pool, clearing
// it before use. If no pooled map is available, it creates a new one.
//
// Parameters:
//   - val: the value to store with the current key
func (w *Writer) writeFinalVal(val any) {
	// If we already have a values map, just add the key-value pair
	if w.values != nil {
		w.values[w.key] = val
		return
	}

	// Try to get a reusable map from the pool
	if m, _ := valueMapPool.Get().(map[string]any); m != nil {
		// Clear the map before reuse
		for k := range m {
			delete(m, k)
		}
		// Add the new key-value pair
		m[w.key] = val
		w.values = m
	} else {
		// Create a new map if pool is empty
		w.values = map[string]any{w.key: val}
	}
}

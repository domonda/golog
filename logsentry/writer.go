package logsentry

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
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

// Writer implements golog.Writer and handles the actual writing of log
// messages to Sentry. It accumulates log data during the logging process
// and sends it as a Sentry event when CommitMessage() is called.
//
// Writer instances are designed to be reused through object pooling to
// minimize memory allocations and improve performance. Each Writer
// accumulates:
//   - Message text and timestamp
//   - Sentry level (mapped from golog level)
//   - Key-value pairs as Sentry event extra data
//   - Optional stack trace information
//
// The Writer automatically maps golog levels to Sentry levels:
//
//	TRACE/DEBUG -> DEBUG, INFO -> INFO, WARN -> WARNING, ERROR -> ERROR, FATAL -> FATAL
//
// Example usage through golog:
//
//	logger.Error("Database error").Str("query", sql).Err(err).Log()
//	// Creates Sentry event with level=ERROR, message="Database error",
//	// and extra data: {"query": sql, "error": err.Error()}
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
//   - All key-value pairs as extra data (from both config.extra and values)
//   - Optional stack trace (if enabled in Sentry options)
//   - A fingerprint based on the message for grouping
//
// After sending the event, the Writer is reset and returned to the object
// pool for reuse, ensuring efficient memory management.
func (w *Writer) CommitMessage() {
	// Flush w.message
	if w.message.Len() > 0 {
		event := sentry.NewEvent()
		event.Timestamp = w.timestamp
		event.Level = w.level
		event.Message = w.message.String()
		event.Fingerprint = []string{event.Message}
		maps.Copy(event.Extra, w.config.extra)
		maps.Copy(event.Extra, w.values)
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
	w.writeVal(val.Error())
}

func (w *Writer) WriteTime(val time.Time) {
	w.writeVal(val)
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

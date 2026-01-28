package goslog

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"strings"
	"time"

	"github.com/domonda/golog"
)

// recorder is a golog.WriterConfig and golog.Writer implementation used for testing.
//
// It records all log messages as maps of attributes, allowing them to be inspected
// in tests. This is particularly useful for validating the slog.Handler implementation
// with slogtest.TestHandler.
//
// The recorder also prints log messages to stdout in a human-readable format for
// debugging purposes.
//
// Note: This type is primarily for internal testing and is not intended for
// production use.
type recorder struct {
	// Result contains all recorded log messages as maps of attribute key-value pairs.
	// Each map represents one log message with its attributes.
	Result []map[string]any

	// Internal state for building the current message
	key    string         // Current attribute key being written
	slice  []any          // Current slice being built (when writing slice attributes)
	values map[string]any // Current message attributes being collected
}

// WriterForNewMessage implements golog.WriterConfig.
// It returns the recorder itself as the writer for each new message.
func (w *recorder) WriterForNewMessage(context.Context, golog.Level) golog.Writer {
	return w
}

// FlushUnderlying implements golog.WriterConfig.
// The recorder doesn't buffer, so this is a no-op.
func (w *recorder) FlushUnderlying() {}

// BeginMessage implements golog.Writer.
// It initializes a new message with standard slog fields (time, level, message)
// and prints the message header to stdout.
func (w *recorder) BeginMessage(config golog.Config, timestamp time.Time, level golog.Level, prefix, text string) {
	if w.key != "" || w.values != nil {
		panic("last message not commited")
	}
	w.values = map[string]any{
		slog.LevelKey:   slog.Level(level),
		slog.MessageKey: text,
	}
	if !timestamp.IsZero() {
		w.values[slog.TimeKey] = timestamp
	}

	if timestamp.IsZero() {
		fmt.Printf("%s=%s %s=%q", slog.LevelKey, slog.Level(level), slog.MessageKey, text)
	} else {
		fmt.Printf("%s=%s %s=%s %s=%q", slog.TimeKey, timestamp.Format("2006-01-02T15:04:05.000"), slog.LevelKey, slog.Level(level), slog.MessageKey, text)
	}
}

// CommitMessage implements golog.Writer.
// It finalizes the current message by converting flat keys with dot notation
// (e.g., "group.key") into nested maps, adds the message to Result, and
// prints a newline to stdout.
func (w *recorder) CommitMessage() {
	// Split keys into groups when they have been prefixed with group names
	// For example, "request.method" becomes {"request": {"method": value}}
	valuesWithGroups := make(map[string]any, len(w.values))
	for key, val := range w.values {
		key, val = splitGroupKeyVal(key, val)
		if curVal, ok := valuesWithGroups[key].(map[string]any); ok {
			if newVal, ok := val.(map[string]any); ok {
				maps.Copy(curVal, newVal)
				val = curVal
			}
		}
		valuesWithGroups[key] = val
	}

	// Add valuesWithGroups to result
	w.Result = append(w.Result, valuesWithGroups)

	// Reset internal state
	w.key = ""
	w.slice = nil
	w.values = nil

	fmt.Println()
}

// String returns a Go syntax representation of all recorded messages.
func (w *recorder) String() string {
	return fmt.Sprintf("%#v", w.Result)
}

// WriteKey implements golog.Writer.
// It sets the current attribute key for the next value to be written.
func (w *recorder) WriteKey(key string) {
	if w.slice != nil {
		panic("already writing slice")
	}
	w.key = key
}

// WriteSliceKey implements golog.Writer.
// It begins writing a slice attribute with the given key.
func (w *recorder) WriteSliceKey(key string) {
	if w.slice != nil {
		panic("already writing slice")
	}
	w.key = key
	w.slice = make([]any, 0)
}

// WriteSliceEnd implements golog.Writer.
// It completes writing a slice attribute.
func (w *recorder) WriteSliceEnd() {
	if w.slice == nil {
		panic("not writing slice")
	}
	w.values[w.key] = w.slice

	fmt.Printf(" %s=%#v", w.key, w.slice)

	w.slice = nil
}

// WriteNil implements golog.Writer.
func (w *recorder) WriteNil() {
	w.writeVal(nil)
}

// WriteBool implements golog.Writer.
func (w *recorder) WriteBool(val bool) {
	w.writeVal(val)
}

// WriteInt implements golog.Writer.
func (w *recorder) WriteInt(val int64) {
	w.writeVal(val)
}

// WriteUint implements golog.Writer.
func (w *recorder) WriteUint(val uint64) {
	w.writeVal(val)
}

// WriteFloat implements golog.Writer.
func (w *recorder) WriteFloat(val float64) {
	w.writeVal(val)
}

// WriteString implements golog.Writer.
func (w *recorder) WriteString(val string) {
	w.writeVal(val)
}

// WriteError implements golog.Writer.
// The error is converted to its string representation.
func (w *recorder) WriteError(val error) {
	w.writeVal(val.Error())
}

// WriteTime implements golog.Writer.
// The time is stored as-is.
func (w *recorder) WriteTime(val time.Time) {
	w.writeVal(val)
}

// WriteUUID implements golog.Writer.
// The UUID is formatted as a string.
func (w *recorder) WriteUUID(val [16]byte) {
	w.writeVal(golog.FormatUUID(val))
}

// WriteJSON implements golog.Writer.
// The JSON bytes are stored as json.RawMessage.
func (w *recorder) WriteJSON(val []byte) {
	w.writeVal(json.RawMessage(val))
}

// writeVal stores a value for the current key, either in the slice being
// built or in the values map.
func (w *recorder) writeVal(val any) {
	if w.slice != nil {
		w.slice = append(w.slice, val)
		return
	}
	w.values[w.key] = val

	fmt.Printf(" %s=%#v", w.key, val)
}

// splitGroupKeyVal converts a dot-notated key into nested maps.
//
// For example:
//   - "key" → "key", val
//   - "G.a" → "G", {"a": val}
//   - "G.a.b" → "G", {"a": {"b": val}}
//   - "G.a.b.c" → "G", {"a": {"b": {"c": val}}}
//
// This allows slog group attributes to be represented as nested maps
// in the recorded results, matching the expected slog behavior.
func splitGroupKeyVal(key string, val any) (rootKey string, rootVal any) {
	if !strings.ContainsRune(key, '.') {
		return key, val
	}
	keys := strings.Split(key, ".")
	groupVals := map[string]any{keys[len(keys)-1]: val}
	for i := len(keys) - 2; i > 0; i-- {
		groupVals = map[string]any{keys[i]: groupVals}
	}
	return keys[0], groupVals
}

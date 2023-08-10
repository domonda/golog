// Reset internal state
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

type recorder struct {
	Result []map[string]any

	// Internal state
	key    string
	slice  []any
	values map[string]any
}

func (w *recorder) BeginMessage(_ context.Context, logger *golog.Logger, t time.Time, level golog.Level, text string) golog.Writer {
	if w.key != "" || w.values != nil {
		panic("last message not commited")
	}
	w.values = map[string]any{
		slog.LevelKey:   slog.Level(level),
		slog.MessageKey: text,
	}
	if !t.IsZero() {
		w.values[slog.TimeKey] = t
	}

	if t.IsZero() {
		fmt.Printf("%s=%s %s=%q", slog.LevelKey, slog.Level(level), slog.MessageKey, text)
	} else {
		fmt.Printf("%s=%s %s=%s %s=%q", slog.TimeKey, t.Format("2006-01-02T15:04:05.000"), slog.LevelKey, slog.Level(level), slog.MessageKey, text)
	}

	return w
}

func (w *recorder) CommitMessage() {
	// valuesWithGroups := w.values
	// TODO
	// Split keys into groups when they have been prefixed with group names
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

	w.key = ""
	w.slice = nil
	w.values = nil

	fmt.Println()
}

func (w *recorder) FlushUnderlying() {}

func (w *recorder) String() string {
	return fmt.Sprintf("%#v", w.Result)
}

func (w *recorder) WriteKey(key string) {
	if w.slice != nil {
		panic("already writing slice")
	}
	w.key = key
}

func (w *recorder) WriteSliceKey(key string) {
	if w.slice != nil {
		panic("already writing slice")
	}
	w.key = key
	w.slice = make([]any, 0)
}

func (w *recorder) WriteSliceEnd() {
	if w.slice == nil {
		panic("not writing slice")
	}
	w.values[w.key] = w.slice

	fmt.Printf(" %s=%#v", w.key, w.slice)

	w.slice = nil
}

func (w *recorder) WriteNil() {
	w.writeVal(nil)
}

func (w *recorder) WriteBool(val bool) {
	w.writeVal(val)
}

func (w *recorder) WriteInt(val int64) {
	w.writeVal(val)
}

func (w *recorder) WriteUint(val uint64) {
	w.writeVal(val)
}

func (w *recorder) WriteFloat(val float64) {
	w.writeVal(val)
}

func (w *recorder) WriteString(val string) {
	w.writeVal(val)
}

func (w *recorder) WriteError(val error) {
	w.writeVal(val.Error())
}

func (w *recorder) WriteUUID(val [16]byte) {
	w.writeVal(golog.FormatUUID(val))
}

func (w *recorder) WriteJSON(val []byte) {
	w.writeVal(json.RawMessage(val))
}

func (w *recorder) writeVal(val any) {
	if w.slice != nil {
		w.slice = append(w.slice, val)
		return
	}
	w.values[w.key] = val

	fmt.Printf(" %s=%#v", w.key, val)
}

func canSplitGroupKey(key string) bool {
	return strings.ContainsRune(key, '.')
}

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

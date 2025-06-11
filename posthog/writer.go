package posthog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/posthog/posthog-go"

	"github.com/domonda/golog"
)

var (
	_ golog.Writer       = new(Writer)
	_ golog.WriterConfig = new(WriterConfig)
)

type WriterConfig struct {
	format *golog.Format
	filter golog.LevelFilter
	client posthog.Client
}

// Config.Logger

// func NewWriterConfig(personalAPIKey string, format *golog.Format, filter golog.LevelFilter) (*WriterConfig, error) {
// 	client, err := posthog.NewWithConfig(
// 		os.Getenv("POSTHOG_API_KEY"),
// 		posthog.Config{
// 			PersonalApiKey: personalAPIKey, // Optional, but much more performant.  If this token is not supplied, then fetching feature flag values will be slower.
// 		},
// 	)
// 	if err != nil {
// 		return nil, err
// 	}
// }

func NewWriterConfigFromPostHogConfig(format *golog.Format, filter golog.LevelFilter, config posthog.Config) (*WriterConfig, error) {
	client, err := posthog.NewWithConfig(
		os.Getenv("POSTHOG_API_KEY"),
		config,
	)
	if err != nil {
		return nil, err
	}
	return &WriterConfig{
		format: format,
		filter: filter,
		client: client,
	}, nil
}

func (c *WriterConfig) WriterForNewMessage(ctx context.Context, level golog.Level) golog.Writer {
	if c.filter.IsInactive(ctx, level) || IsContextWithoutLogging(ctx) {
		return nil
	}
	return &Writer{config: c}
}

func (c *WriterConfig) FlushUnderlying() {

}

type Writer struct {
	config    *WriterConfig
	timestamp time.Time
	message   strings.Builder
	key       string
	slice     []any
}

func (w *Writer) BeginMessage(config golog.Config, timestamp time.Time, level golog.Level, prefix, text string) {
	w.timestamp = timestamp

	// levels := config.Levels()
	// switch level {
	// case levels.Fatal:
	// 	w.level = sentry.LevelFatal
	// case levels.Error:
	// 	w.level = sentry.LevelError
	// case levels.Warn:
	// 	w.level = sentry.LevelWarning
	// case levels.Info:
	// 	w.level = sentry.LevelInfo
	// case levels.Debug:
	// 	w.level = sentry.LevelDebug
	// case levels.Trace:
	// 	w.level = sentry.LevelDebug
	// default:
	// 	w.level = UnknownLevel
	// }

	if prefix != "" {
		fmt.Fprintf(&w.message, w.config.format.PrefixFmt, prefix, text)
	} else {
		w.message.WriteString(text)
	}
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
	w.config.client.Enqueue(posthog.Capture{
		DistinctId: "", // TODO requestID or configure what attribute to use as distinctID
		Event:      w.message.String(),
		Timestamp:  w.timestamp,
		Properties: posthog.Properties{},
	})

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

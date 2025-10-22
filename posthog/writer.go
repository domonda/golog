package posthog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/posthog/posthog-go"

	"github.com/domonda/golog"
)

var (
	_ golog.Writer       = new(Writer)
	_ golog.WriterConfig = new(WriterConfig)
)

type WriterConfig struct {
	format     *golog.Format
	filter     golog.LevelFilter
	client     posthog.Client
	distinctID string // PostHog's unique identifier for tracking events (see function docs for details)
	extra      map[string]any
	writerPool sync.Pool
	valsAsMsg  bool
}

// NewWriterConfigFromEnv returns a new WriterConfig using PostHog client created from environment variables.
// It reads POSTHOG_API_KEY from environment and creates a client with default configuration.
//
// distinctID is PostHog's unique identifier for tracking events. It serves to associate log events
// with specific users or system components. Common patterns include:
//   - "system" or "server" for system-generated logs
//   - "user_12345" or "user@example.com" for user-specific logs
//   - "service_api" or "service_database" for service-specific logs
//   - "cron_backup" or "worker_email" for job-specific logs
//
// PostHog restricts certain values: "anonymous", "guest", "distinctid", "undefined", "null", "true", "false", etc.
// The distinctID should be unique and meaningful for your analytics needs.
func NewWriterConfigFromEnv(format *golog.Format, filter golog.LevelFilter, distinctID string, valsAsMsg bool, extra map[string]any) (*WriterConfig, error) {
	apiKey := os.Getenv("POSTHOG_API_KEY")
	if apiKey == "" {
		return nil, errors.New("POSTHOG_API_KEY is not set")
	}
	client, err := posthog.NewWithConfig(
		apiKey,
		posthog.Config{
			Endpoint: "https://us.i.posthog.com", // Default endpoint
		},
	)
	if err != nil {
		return nil, err
	}
	return NewWriterConfig(client, format, filter, distinctID, valsAsMsg, extra), nil
}

// NewWriterConfig returns a new WriterConfig for PostHog.
// Any values passed as extra will be added to every log message.
//
// distinctID is PostHog's unique identifier for tracking events. It serves to associate log events
// with specific users or system components. Common patterns include:
//   - "system" or "server" for system-generated logs
//   - "user_12345" or "user@example.com" for user-specific logs
//   - "service_api" or "service_database" for service-specific logs
//   - "cron_backup" or "worker_email" for job-specific logs
//
// PostHog restricts certain values: "anonymous", "guest", "distinctid", "undefined", "null", "true", "false", etc.
// The distinctID should be unique and meaningful for your analytics needs.
func NewWriterConfig(client posthog.Client, format *golog.Format, filter golog.LevelFilter, distinctID string, valsAsMsg bool, extra map[string]any) *WriterConfig {
	return &WriterConfig{
		format:     format,
		filter:     filter,
		client:     client,
		distinctID: distinctID,
		valsAsMsg:  valsAsMsg,
		extra:      extra,
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
	// Posthog has no flush method
}

type Writer struct {
	config    *WriterConfig
	timestamp time.Time
	level     golog.Level
	levels    *golog.Levels
	message   strings.Builder
	values    map[string]any
	key       string
	slice     []any
}

func (w *Writer) BeginMessage(config golog.Config, timestamp time.Time, level golog.Level, prefix, text string) {
	w.timestamp = timestamp
	w.level = level
	w.levels = config.Levels()

	if prefix != "" {
		fmt.Fprintf(&w.message, w.config.format.PrefixFmt, prefix, text)
	} else {
		w.message.WriteString(text)
	}
}

func (w *Writer) CommitMessage() {
	// Flush w.message
	if w.message.Len() > 0 {
		// Create PostHog properties
		properties := posthog.NewProperties()

		// Add log level - use proper level name if available
		levelName := w.levels.Name(w.level)
		properties.Set("log_level", levelName)

		// Add extra properties from config
		for key, val := range w.config.extra {
			properties.Set(key, val)
		}

		// Add values from the log message
		for key, val := range w.values {
			properties.Set(key, val)
		}

		// Capture the log event
		w.config.client.Enqueue(posthog.Capture{
			DistinctId: w.config.distinctID,
			Event:      "log_message",
			Timestamp:  w.timestamp,
			Properties: properties.Set("message", w.message.String()),
		})
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

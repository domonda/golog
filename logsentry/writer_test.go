package logsentry

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"

	"github.com/domonda/golog"
)

// captureTransport implements sentry.Transport and records every event that
// would be sent, so tests can inspect the assembled *sentry.Event.
type captureTransport struct {
	events []*sentry.Event
}

func (t *captureTransport) Flush(time.Duration) bool              { return true }
func (t *captureTransport) FlushWithContext(context.Context) bool { return true }
func (t *captureTransport) Configure(sentry.ClientOptions)        {}
func (t *captureTransport) SendEvent(event *sentry.Event)         { t.events = append(t.events, event) }
func (t *captureTransport) Close()                                {}

func newTestHub(t *testing.T, transport sentry.Transport) *sentry.Hub {
	t.Helper()
	client, err := sentry.NewClient(sentry.ClientOptions{
		// A custom Transport is always used regardless of the DSN; the DSN is
		// only set so events are not dropped before reaching the transport.
		Dsn:       "https://public@example.com/1",
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("sentry.NewClient: %v", err)
	}
	return sentry.NewHub(client, sentry.NewScope())
}

// TestWriterContextValues verifies that golog typed values land in the
// event's "log" Context with the expected types, that time.Time values are
// formatted via the configured Format (TimeFormat + Location), and that nil
// values are passed through as null.
func TestWriterContextValues(t *testing.T) {
	transport := &captureTransport{}
	hub := newTestHub(t, transport)

	format := golog.NewDefaultFormat()
	format.TimeFormat = "2006-01-02 15:04:05"
	format.Location = time.UTC

	config := NewWriterConfig(hub, format, golog.AllLevelsActive, false, nil)
	logger := golog.NewLogger(golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, config))

	// 14:30 in +02:00 == 12:30 UTC, exercising Format.Location conversion.
	ts := time.Date(2026, 6, 3, 14, 30, 0, 0, time.FixedZone("CEST", 2*60*60))
	id := [16]byte{0x85, 0x69, 0x2e, 0x8d, 0x49, 0xbf, 0x41, 0x50, 0xa1, 0x69, 0x6c, 0x2a, 0xdb, 0x93, 0x46, 0x3c}

	logger.Error("boom").
		Str("query", "SELECT 1").
		Int("count", 42).
		Uint64("big", math.MaxUint64).
		Time("when", ts).
		UUID("id", id).
		Nil("missing").
		JSON("payload", []byte(`{"a":1}`)).
		Log()

	hub.Flush(time.Second)

	if len(transport.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(transport.events))
	}
	event := transport.events[0]
	if event.Message != "boom" {
		t.Errorf("event.Message = %q, want %q", event.Message, "boom")
	}

	logCtx, ok := event.Contexts["log"]
	if !ok {
		t.Fatalf(`event.Contexts has no "log" key; contexts: %v`, event.Contexts)
	}

	if got := logCtx["query"]; got != "SELECT 1" {
		t.Errorf(`log["query"] = %#v, want %q`, got, "SELECT 1")
	}
	if got := logCtx["count"]; got != int64(42) {
		t.Errorf(`log["count"] = %#v (%T), want int64(42)`, got, got)
	}
	if got := logCtx["big"]; got != uint64(math.MaxUint64) {
		t.Errorf(`log["big"] = %#v (%T), want uint64 max`, got, got)
	}
	if got := logCtx["when"]; got != "2026-06-03 12:30:00" {
		t.Errorf(`log["when"] = %#v, want %q`, got, "2026-06-03 12:30:00")
	}
	if got := logCtx["id"]; got != "85692e8d-49bf-4150-a169-6c2adb93463c" {
		t.Errorf(`log["id"] = %#v, want UUID string`, got)
	}
	if got, present := logCtx["missing"]; !present || got != nil {
		t.Errorf(`log["missing"] = %#v, present=%v; want nil and present`, got, present)
	}
	payload, ok := logCtx["payload"].(json.RawMessage)
	if !ok {
		t.Errorf(`log["payload"] = %#v (%T), want json.RawMessage`, logCtx["payload"], logCtx["payload"])
	} else if string(payload) != `{"a":1}` {
		t.Errorf(`log["payload"] = %s, want %s`, payload, `{"a":1}`)
	}
}

// TestWriterReservedTypeKey verifies that a value logged under the Sentry-
// reserved "type" key is remapped to "type_" so it survives as context data,
// for both per-message values and config-level extra.
func TestWriterReservedTypeKey(t *testing.T) {
	transport := &captureTransport{}
	hub := newTestHub(t, transport)

	config := NewWriterConfig(hub, golog.NewDefaultFormat(), golog.AllLevelsActive, false,
		map[string]any{"type": "from-extra"})
	logger := golog.NewLogger(golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, config))

	logger.Error("boom").Str("type", "invoice").Log()
	hub.Flush(time.Second)

	if len(transport.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(transport.events))
	}
	logCtx := transport.events[0].Contexts["log"]

	if _, present := logCtx["type"]; present {
		t.Errorf(`log["type"] should be absent (reserved); got %#v`, logCtx["type"])
	}
	// Per-message value wins over config extra under the remapped key.
	if got := logCtx["type_"]; got != "invoice" {
		t.Errorf(`log["type_"] = %#v, want %q`, got, "invoice")
	}
}

// TestWriterTimeDefaultFormat verifies time formatting falls back to
// golog.DefaultTimeFormat when Format.TimeFormat is empty, in the time's
// original location when Format.Location is nil.
func TestWriterTimeDefaultFormat(t *testing.T) {
	transport := &captureTransport{}
	hub := newTestHub(t, transport)

	format := golog.NewDefaultFormat()
	format.TimeFormat = "" // force DefaultTimeFormat fallback
	format.Location = nil  // keep original location

	config := NewWriterConfig(hub, format, golog.AllLevelsActive, false, nil)
	logger := golog.NewLogger(golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, config))

	loc := time.FixedZone("CEST", 2*60*60)
	ts := time.Date(2026, 6, 3, 14, 30, 0, 0, loc)

	logger.Error("boom").Time("when", ts).Log()
	hub.Flush(time.Second)

	if len(transport.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(transport.events))
	}
	want := ts.Format(golog.DefaultTimeFormat)
	if got := transport.events[0].Contexts["log"]["when"]; got != want {
		t.Errorf(`log["when"] = %#v, want %q`, got, want)
	}
}

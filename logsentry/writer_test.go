package logsentry

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"runtime"
	"slices"
	"strings"
	"sync"
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

func newTestHub(t *testing.T, transport sentry.Transport, opts ...func(*sentry.ClientOptions)) *sentry.Hub {
	t.Helper()
	options := sentry.ClientOptions{
		// A custom Transport is always used regardless of the DSN; the DSN is
		// only set so events are not dropped before reaching the transport.
		Dsn:       "https://public@example.com/1",
		Transport: transport,
	}
	for _, opt := range opts {
		opt(&options)
	}
	client, err := sentry.NewClient(options)
	if err != nil {
		t.Fatalf("sentry.NewClient: %v", err)
	}
	return sentry.NewHub(client, sentry.NewScope())
}

// attachStacktrace is a newTestHub option that enables
// ClientOptions.AttachStacktrace.
func attachStacktrace(o *sentry.ClientOptions) { o.AttachStacktrace = true }

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

// TestWriterErrAsException verifies that an error logged via Message.Err is
// surfaced as a Sentry exception: its message becomes the exception Value (the
// issue subtitle, replacing "(No error message)") while the log text stays the
// exception Type (the issue title). The error string is still kept in the
// "log" context for completeness.
func TestWriterErrAsException(t *testing.T) {
	transport := &captureTransport{}
	hub := newTestHub(t, transport)

	config := NewWriterConfig(hub, golog.NewDefaultFormat(), golog.AllLevelsActive, false, nil)
	logger := golog.NewLogger(golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, config))

	logger.Error("Failed to sync email message").Err(errors.New("connection refused")).Log()
	hub.Flush(time.Second)

	if len(transport.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(transport.events))
	}
	event := transport.events[0]

	if len(event.Exception) != 1 {
		t.Fatalf("expected 1 exception, got %d", len(event.Exception))
	}
	exc := event.Exception[0]
	if exc.Type != "Failed to sync email message" {
		t.Errorf("exception.Type = %q, want the log message", exc.Type)
	}
	if exc.Value != "connection refused" {
		t.Errorf("exception.Value = %q, want %q", exc.Value, "connection refused")
	}
	// The message (issue title) is preserved alongside the exception.
	if event.Message != "Failed to sync email message" {
		t.Errorf("event.Message = %q, want the log message", event.Message)
	}
	// The error string remains available in the "log" context too.
	if got := event.Contexts["log"]["error"]; got != "connection refused" {
		t.Errorf(`log["error"] = %#v, want %q`, got, "connection refused")
	}
}

// TestWriterErrorsSliceNoException verifies that errors logged via Message.Errs
// (a slice under the "errors" key) stay context data only and do not produce a
// Sentry exception, so the issue title/grouping stay clean. Only a standalone
// Message.Err under the reserved "error" key is promoted to an exception.
func TestWriterErrorsSliceNoException(t *testing.T) {
	transport := &captureTransport{}
	hub := newTestHub(t, transport)

	config := NewWriterConfig(hub, golog.NewDefaultFormat(), golog.AllLevelsActive, false, nil)
	logger := golog.NewLogger(golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, config))

	logger.Error("batch failed").
		Errs([]error{errors.New("first"), errors.New("second")}).
		Log()
	hub.Flush(time.Second)

	if len(transport.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(transport.events))
	}
	event := transport.events[0]
	if len(event.Exception) != 0 {
		t.Errorf("expected no exception for Errs slice, got %d", len(event.Exception))
	}
	if got, ok := event.Contexts["log"]["errors"].([]any); !ok || len(got) != 2 {
		t.Errorf(`log["errors"] = %#v, want a 2-element slice`, event.Contexts["log"]["errors"])
	}
}

// TestWriterErrStacktrace verifies that a stack trace carried by the logged
// error (via a pkg/errors-style StackTrace() method, which sentry.ExtractStacktrace
// recognizes) is extracted and attached to the exception even when the client
// does NOT have AttachStacktrace enabled — proving the error's own origin stack
// is used. A plain error on the same hub serves as the control: it must get no
// stack, otherwise the call-site fallback could satisfy the first assertion and
// the client's AttachStacktrace=false option would be ignored.
func TestWriterErrStacktrace(t *testing.T) {
	transport := &captureTransport{}
	hub := newTestHub(t, transport) // AttachStacktrace is off by default

	config := NewWriterConfig(hub, golog.NewDefaultFormat(), golog.AllLevelsActive, false, nil)
	logger := golog.NewLogger(golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, config))

	// Create the error inside a stdlib callback so its stack keeps a frame that
	// survives filterFrames (see callViaStdlib).
	var errWithStack error
	callViaStdlib(func() { errWithStack = newStackErr("with stack") })
	logger.Error("boom").Err(errWithStack).Log()
	hub.Flush(time.Second)

	if len(transport.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(transport.events))
	}
	event := transport.events[0]
	if len(event.Exception) != 1 {
		t.Fatalf("expected 1 exception, got %d", len(event.Exception))
	}
	// A plain error would yield no stack here (AttachStacktrace off), and the
	// log call itself is not inside the stdlib callback, so the sync frame can
	// only come from the error's own StackTrace(). Per-frame golog filtering is
	// covered by TestFilterFrames.
	if !hasModuleFrame(event.Exception[0].Stacktrace, "sync") {
		t.Fatalf("expected the stack extracted from the error's StackTrace(), got %+v", event.Exception[0].Stacktrace)
	}

	logger.Error("boom").Err(errors.New("no stack")).Log()
	hub.Flush(time.Second)

	if len(transport.events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(transport.events))
	}
	control := transport.events[1]
	if len(control.Exception) != 1 {
		t.Fatalf("expected 1 exception, got %d", len(control.Exception))
	}
	if control.Exception[0].Stacktrace != nil {
		t.Error("expected no stack trace for a plain error when AttachStacktrace is off")
	}
}

// TestWriterErrStacktracePreferredOverCallSite verifies that the logged error's
// own stack wins over the call-site stack even with AttachStacktrace enabled,
// so Sentry shows where the error happened, not where it was logged. The error
// origin is marked by sync.Once frames, the log call site by a slices frame.
func TestWriterErrStacktracePreferredOverCallSite(t *testing.T) {
	transport := &captureTransport{}
	hub := newTestHub(t, transport, attachStacktrace)

	config := NewWriterConfig(hub, golog.NewDefaultFormat(), golog.AllLevelsActive, false, nil)
	logger := golog.NewLogger(golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, config))

	var errWithStack error
	callViaStdlib(func() { errWithStack = newStackErr("with stack") })
	_ = slices.IndexFunc([]int{0}, func(int) bool {
		logger.Error("boom").Err(errWithStack).Log()
		return true
	})
	hub.Flush(time.Second)

	if len(transport.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(transport.events))
	}
	event := transport.events[0]
	if len(event.Exception) != 1 {
		t.Fatalf("expected 1 exception, got %d", len(event.Exception))
	}
	stack := event.Exception[0].Stacktrace
	if !hasModuleFrame(stack, "sync") || hasModuleFrame(stack, "slices") {
		t.Fatalf("expected the error's own origin stack, not the call-site stack, got %+v", stack)
	}
}

// TestWriterErrFallbackStacktrace verifies that when the logged error carries
// no stack of its own but the client has AttachStacktrace enabled, the current
// call-site stack is attached to the exception instead. A message logged without
// any error still gets the call-site stack, attached as the current thread, so
// AttachStacktrace also covers plain error-level log lines.
func TestWriterErrFallbackStacktrace(t *testing.T) {
	transport := &captureTransport{}
	hub := newTestHub(t, transport, attachStacktrace)

	config := NewWriterConfig(hub, golog.NewDefaultFormat(), golog.AllLevelsActive, false, nil)
	logger := golog.NewLogger(golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, config))

	// Log from a stdlib callback so the call-site stack keeps a frame that
	// survives filterFrames (see callViaStdlib).
	callViaStdlib(func() { logger.Error("boom").Err(errors.New("no stack")).Log() })
	hub.Flush(time.Second)

	if len(transport.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(transport.events))
	}
	event := transport.events[0]
	if len(event.Exception) != 1 {
		t.Fatalf("expected 1 exception, got %d", len(event.Exception))
	}
	if !hasModuleFrame(event.Exception[0].Stacktrace, "sync") {
		t.Fatalf("expected fallback call-site stack trace when AttachStacktrace is on, got %+v", event.Exception[0].Stacktrace)
	}
	// The call-site stack runs through golog and logsentry, whose frames must
	// not reach Sentry (TestFilterFrames covers the helper on its own).
	if slices.ContainsFunc(event.Exception[0].Stacktrace.Frames, func(f sentry.Frame) bool {
		return strings.HasPrefix(f.Module, "github.com/domonda/golog")
	}) {
		t.Errorf("expected golog-internal frames to be filtered from the stack, got %+v", event.Exception[0].Stacktrace.Frames)
	}

	callViaStdlib(func() { logger.Error("no error logged").Log() })
	hub.Flush(time.Second)

	if len(transport.events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(transport.events))
	}
	event = transport.events[1]
	if len(event.Exception) != 0 {
		t.Errorf("expected no exception without a logged error, got %d", len(event.Exception))
	}
	if len(event.Threads) != 1 {
		t.Fatalf("expected 1 thread carrying the call-site stack, got %d", len(event.Threads))
	}
	if !event.Threads[0].Current || !hasModuleFrame(event.Threads[0].Stacktrace, "sync") {
		t.Errorf("expected the current thread with the call-site stack trace, got %+v", event.Threads[0])
	}
}

// TestWriterRecoverStacktraceEndsAtPanicOrigin pins the documented behavior
// that, with AttachStacktrace enabled, a message logged from a recover handler
// carries a call-site stack that ends at the panic origin (sentry-go trims the
// frames above the panic). Stdlib frames serve as markers because this test's
// own frames are filtered out: the sync.Once frames below the panic must
// survive, the slices.IndexFunc frame inside the recover handler must not.
func TestWriterRecoverStacktraceEndsAtPanicOrigin(t *testing.T) {
	transport := &captureTransport{}
	hub := newTestHub(t, transport, attachStacktrace)

	config := NewWriterConfig(hub, golog.NewDefaultFormat(), golog.AllLevelsActive, false, nil)
	logger := golog.NewLogger(golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, config))

	func() {
		defer func() {
			if recover() != nil {
				_ = slices.IndexFunc([]int{0}, func(int) bool {
					logger.Error("recovered").Err(errors.New("no stack")).Log()
					return true
				})
			}
		}()
		callViaStdlib(func() { panic("boom") })
	}()
	hub.Flush(time.Second)

	if len(transport.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(transport.events))
	}
	event := transport.events[0]
	if len(event.Exception) != 1 {
		t.Fatalf("expected 1 exception, got %d", len(event.Exception))
	}
	stack := event.Exception[0].Stacktrace
	if !hasModuleFrame(stack, "sync") {
		t.Errorf("expected the panic-origin path (sync.Once) in the stack, got %+v", stack)
	}
	if hasModuleFrame(stack, "slices") {
		t.Errorf("expected the recover-handler frames above the panic to be trimmed, got %+v", stack)
	}
}

// TestWriterHubWithoutClient verifies that logging an error through a hub with
// no client bound (e.g. sentry.CurrentHub() before sentry.Init) is a silent
// no-op, so an unconfigured Sentry never breaks logging: a panic in the writer
// would be recovered and reported to the error handler.
func TestWriterHubWithoutClient(t *testing.T) {
	hub := sentry.NewHub(nil, sentry.NewScope())
	// Premise: a hub without a bound client reports a disabled client. If a
	// future sentry-go breaks this, revisit the AttachStacktrace guard in
	// CommitMessage.
	if hub.Client().IsEnabled() {
		t.Fatal("premise: a hub without a bound client must have a disabled client")
	}

	var captured error
	config := NewWriterConfig(hub, golog.NewDefaultFormat(), golog.AllLevelsActive, false, nil,
		WithErrorHandler(func(err error) { captured = err }))
	logger := golog.NewLogger(golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, config))

	logger.Error("boom").Err(errors.New("no stack")).Log()

	if captured != nil {
		t.Fatalf("expected no writer error without a client, got: %v", captured)
	}
}

// TestFilterFrames verifies that golog-internal frames (including the
// logsentry sub-package) are dropped while application and main frames survive.
func TestFilterFrames(t *testing.T) {
	in := []sentry.Frame{
		{Module: "github.com/domonda/golog", Function: "Log"},
		{Module: "github.com/domonda/golog/logsentry", Function: "CommitMessage"},
		{Module: "github.com/domonda/server/gmailsync", Function: "Sync"},
		{Module: "main", Function: "main"},
	}
	out := filterFrames(in)
	if len(out) != 2 {
		t.Fatalf("filterFrames kept %d frames, want 2: %+v", len(out), out)
	}
	for _, f := range out {
		if strings.HasPrefix(f.Module, "github.com/domonda/golog") {
			t.Errorf("golog-internal frame survived filtering: %s.%s", f.Module, f.Function)
		}
	}
}

// stackErr is a minimal error carrying a pkg/errors-style StackTrace() method
// (returning []uintptr) that sentry.ExtractStacktrace recognizes. newStackErr
// captures the real call stack at construction so the PCs resolve to genuine
// runtime frames.
type stackErr struct {
	msg     string
	callers []uintptr
}

func newStackErr(msg string) *stackErr {
	var pcs [32]uintptr
	n := runtime.Callers(2, pcs[:]) // skip runtime.Callers and newStackErr itself
	return &stackErr{msg: msg, callers: pcs[:n]}
}

func (e *stackErr) Error() string         { return e.msg }
func (e *stackErr) StackTrace() []uintptr { return e.callers }

// callViaStdlib runs fn through sync.Once so a stack captured inside fn
// contains standard library frames. Every frame of this test package belongs to
// the github.com/domonda/golog module and is removed by filterFrames, so a
// surviving "sync" frame proves that a real, non-empty stack was attached.
func callViaStdlib(fn func()) {
	var once sync.Once
	once.Do(fn)
}

// hasModuleFrame reports whether st contains a frame of the given module.
func hasModuleFrame(st *sentry.Stacktrace, module string) bool {
	return st != nil && slices.ContainsFunc(st.Frames, func(f sentry.Frame) bool { return f.Module == module })
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

// TestSentryDebugWriter verifies the DebugWriter adapter forwards genuine
// delivery failures (including the default async path's "error sending
// envelope") while ignoring routine output and intentional drops (sampling,
// BeforeSend). Lines are verbatim from sentry-go debuglog calls present in
// v0.50.0, the minimum version logsentry requires.
func TestSentryDebugWriter(t *testing.T) {
	cases := []struct {
		line    string
		forward bool
	}{
		{"Sending event-1 to sentry.io project: 42", false},                     // routine
		{"Event dropped due to SampleRate hit.", false},                         // intentional sampling
		{"Event dropped due to BeforeSend callback.", false},                    // intentional filter
		{"Event dropped by one of the Client EventProcessors: x", false},        // intentional processor
		{"error sending envelope: connection refused", true},                    // default-path send failure
		{"Sending event-2 failed because the request was too large: x", true},   // 413
		{"Sending event-3 failed with server error 500: boom", true},            // 5xx
		{"Event dropped due to transport buffer being full. event-4", true},     // overload drop
		{"Too many requests for \"error\", backing off till: 2026-06-03", true}, // 429
		{"Failed to build envelope, skipping delivery. evt: boom", true},        // serialization fail
		{"Failed to create request from envelope: bad DSN", true},               // request build fail
		{"Unexpected status code 418 for event event-5", true},                  // unexpected
		{"Failed to send client report: connection refused", true},              // client report send
		{"Failed to serialize client report: json: boom", true},                 // client report serialize
		{"Failed to encode client report: json: boom", true},                    // client report encode
		{"Failed to create client report request: bad DSN", true},               // client report build
	}

	var got []string
	w := NewSentryDebugWriter(func(err error) { got = append(got, err.Error()) })

	// Compare the exact forwarded lines, not just their count, so a marker that
	// stops matching cannot be masked by another line that starts matching.
	var want []string
	for _, c := range cases {
		if c.forward {
			want = append(want, "sentry: "+c.line)
		}
		if _, err := io.WriteString(w, c.line+"\n"); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}

	if !slices.Equal(got, want) {
		t.Fatalf("forwarded errors mismatch\n got: %q\nwant: %q", got, want)
	}
}

// TestWithErrorHandler verifies the per-config handler receives writer errors,
// and that reportError falls back to golog.ErrorHandler when unset.
func TestWithErrorHandler(t *testing.T) {
	transport := &captureTransport{}
	hub := newTestHub(t, transport)

	var captured error
	config := NewWriterConfig(hub, golog.NewDefaultFormat(), golog.AllLevelsActive, false, nil,
		WithErrorHandler(func(err error) { captured = err }))

	want := errors.New("boom")
	config.handleError(want)
	if captured != want {
		t.Errorf("configured handler got %v, want %v", captured, want)
	}

	// Without WithErrorHandler, reportError defers to golog.ErrorHandler().
	prev := golog.ErrorHandler()
	defer golog.SetErrorHandler(prev)
	var fallback error
	golog.SetErrorHandler(func(err error) { fallback = err })
	reportError(nil, want)
	if fallback != want {
		t.Errorf("fallback handler got %v, want %v", fallback, want)
	}

	// SetErrorHandler(nil) is valid and disables handling; ErrorHandlerOr
	// returns the fallback in that case.
	golog.SetErrorHandler(nil)
	if golog.ErrorHandler() != nil {
		t.Error("ErrorHandler() should be nil after SetErrorHandler(nil)")
	}
	sentinel := func(error) {}
	if got := golog.ErrorHandlerOr(sentinel); got == nil {
		t.Error("ErrorHandlerOr should return fallback when handler is nil")
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

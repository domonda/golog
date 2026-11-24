package logsentry

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"

	"github.com/domonda/golog/logfile"
)

// TestLogfileWriteCallbackWriteError verifies that a failed write
// of a real RotatingWriter is sent as Sentry event
// with the unwritten data in the "logfile" context.
func TestLogfileWriteCallbackWriteError(t *testing.T) {
	transport := &captureTransport{}
	hub := newTestHub(t, transport)

	filePath := filepath.Join(t.TempDir(), "app.log")
	writer, err := logfile.NewRotatingWriterWithCallback(filePath, "", 0o600, 1024*1024, NewLogfileWriteCallback(hub, 0))
	if err != nil {
		t.Fatal(err)
	}

	// A successful write sends no event
	if _, err := writer.Write([]byte("ok\n")); err != nil {
		t.Fatal(err)
	}
	if len(transport.events) != 0 {
		t.Fatalf("got %d events after successful write, want 0", len(transport.events))
	}

	// Writing after Close fails without writing anything
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	_, writeErr := writer.Write([]byte("lost message\n"))
	if !errors.Is(writeErr, os.ErrClosed) {
		t.Fatalf("Write after Close returned %v, want os.ErrClosed", writeErr)
	}

	if len(transport.events) != 1 {
		t.Fatalf("got %d events, want 1", len(transport.events))
	}
	event := transport.events[0]
	if event.Level != sentry.LevelError {
		t.Errorf("event level %q, want %q", event.Level, sentry.LevelError)
	}
	if event.Message != "error writing log file" {
		t.Errorf("event message %q", event.Message)
	}
	if len(event.Exception) != 1 || event.Exception[0].Value != writeErr.Error() {
		t.Errorf("event exception %+v, want value %q", event.Exception, writeErr.Error())
	}
	logCtx := event.Contexts["logfile"]
	want := sentry.Context{
		"file_path":     filePath,
		"bytes_total":   len("lost message\n"),
		"bytes_written": 0,
		"rotate_failed": false,
		"write_failed":  true,
		"unwritten":     "lost message",
	}
	for k, v := range want {
		if logCtx[k] != v {
			t.Errorf("logfile context %q = %#v, want %#v", k, logCtx[k], v)
		}
	}
	if _, ok := logCtx["skipped_errors"]; ok {
		t.Error("logfile context has skipped_errors without skipped errors")
	}
}

// TestLogfileWriteCallbackRotateError verifies that a failed rotation
// whose data was still written is reported as rotation error
// without unwritten data, and as write error when the write failed too.
func TestLogfileWriteCallbackRotateError(t *testing.T) {
	transport := &captureTransport{}
	hub := newTestHub(t, transport)
	callback := NewLogfileWriteCallback(hub, 0)

	data := []byte("message\n")
	rotateErr := &logfile.RotateError{FilePath: "app.log", Err: errors.New("permission denied")}

	callback("app.log", data, len(data), rotateErr)
	callback("app.log", data, 3, errors.Join(rotateErr, errors.New("disk full")))

	if len(transport.events) != 2 {
		t.Fatalf("got %d events, want 2", len(transport.events))
	}

	rotated := transport.events[0]
	if rotated.Message != "error rotating log file" {
		t.Errorf("rotation event message %q", rotated.Message)
	}
	if ctx := rotated.Contexts["logfile"]; ctx["rotate_failed"] != true || ctx["write_failed"] != false {
		t.Errorf("rotation event logfile context %v", ctx)
	}
	if _, ok := rotated.Contexts["logfile"]["unwritten"]; ok {
		t.Error("rotation event has unwritten data although all data was written")
	}

	both := transport.events[1]
	if both.Message != "error writing log file" {
		t.Errorf("rotation and write event message %q", both.Message)
	}
	ctx := both.Contexts["logfile"]
	if ctx["rotate_failed"] != true || ctx["write_failed"] != true || ctx["unwritten"] != "sage" {
		t.Errorf("rotation and write event logfile context %v", ctx)
	}
}

// TestLogfileWriteCallbackMinInterval verifies that errors within
// minInterval after a sent event are counted instead of sent,
// and that the next event reports their count.
func TestLogfileWriteCallbackMinInterval(t *testing.T) {
	transport := &captureTransport{}
	hub := newTestHub(t, transport)
	const minInterval = 50 * time.Millisecond
	callback := NewLogfileWriteCallback(hub, minInterval)

	data := []byte("message\n")
	writeErr := errors.New("disk full")
	for range 3 {
		callback("app.log", data, 0, writeErr)
	}
	if len(transport.events) != 1 {
		t.Fatalf("got %d events within minInterval, want 1", len(transport.events))
	}

	time.Sleep(minInterval)
	callback("app.log", data, 0, writeErr)
	if len(transport.events) != 2 {
		t.Fatalf("got %d events after minInterval, want 2", len(transport.events))
	}
	if skipped := transport.events[1].Contexts["logfile"]["skipped_errors"]; skipped != 2 {
		t.Errorf("skipped_errors = %#v, want 2", skipped)
	}
}

// TestLogfileWriteCallbackCurrentHub verifies that a nil hub
// sends the event via sentry.CurrentHub.
func TestLogfileWriteCallbackCurrentHub(t *testing.T) {
	transport := &captureTransport{}
	hub := newTestHub(t, transport)
	prevClient := sentry.CurrentHub().Client()
	sentry.CurrentHub().BindClient(hub.Client())
	defer sentry.CurrentHub().BindClient(prevClient)

	NewLogfileWriteCallback(nil, 0)("app.log", []byte("message\n"), 0, errors.New("disk full"))
	if len(transport.events) != 1 {
		t.Fatalf("got %d events via sentry.CurrentHub, want 1", len(transport.events))
	}
}

func TestUnwrittenString(t *testing.T) {
	prev := LogfileUnwrittenMaxBytes
	defer func() { LogfileUnwrittenMaxBytes = prev }()
	LogfileUnwrittenMaxBytes = 4

	tests := []struct {
		name string
		data string
		want string
	}{
		{name: "empty", data: "", want: ""},
		{name: "trailing newline", data: "abc\n", want: "abc"},
		{name: "truncated", data: "abcdef\n", want: "abcd…"},
		// "ä" is two bytes, the cut after 4 bytes splits the second one
		{name: "truncated in rune", data: "aäöü", want: "aä…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := unwrittenString([]byte(tt.data)); got != tt.want {
				t.Errorf("unwrittenString(%q) = %q, want %q", tt.data, got, tt.want)
			}
		})
	}

	// A huge unwritten part is cut to the limit
	LogfileUnwrittenMaxBytes = prev
	if got := unwrittenString([]byte(strings.Repeat("x", 2*prev))); len(got) != prev+len("…") {
		t.Errorf("unwrittenString of %d bytes returned %d bytes, want %d", 2*prev, len(got), prev+len("…"))
	}
}

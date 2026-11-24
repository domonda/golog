package logsentry

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"

	"github.com/domonda/golog/logfile"
)

// LogfileUnwrittenMaxBytes limits how many bytes of the data
// that a failed log file write did not write are sent
// with the Sentry event of a callback from [NewLogfileWriteCallback].
var LogfileUnwrittenMaxBytes = 4096

// NewLogfileWriteCallback returns a [logfile.WriteCallback] that sends
// every error of a [logfile.RotatingWriter] as Sentry event via hub.
// Writes without error are ignored.
// If hub is nil, then [sentry.CurrentHub] is used at the time of the error.
//
// The event has the level error and an exception with the error text.
// Its message and issue title is "error rotating log file" when only
// the rotation before the write failed (a [*logfile.RotateError],
// the data was still written to the current log file),
// or "error writing log file" when data was not written completely.
// The "logfile" context of the event contains the log file path,
// the byte counts, and the start of the data that was not written
// (at most [LogfileUnwrittenMaxBytes]), because that log message
// would be lost otherwise.
//
// A full disk makes every write fail, so minInterval limits
// how often events are sent: after an event, errors are counted
// but not sent until minInterval has passed. The next event reports
// the count of skipped errors as "skipped_errors" in its "logfile" context.
// A minInterval of zero or less sends an event for every error.
//
// The callback runs in the goroutine that called Write.
// Sentry's default HTTP transport sends events asynchronously,
// a synchronous transport like sentry.HTTPSyncTransport
// delays the write until the event was sent.
// Sentry must not log to a logger that writes to the same log file,
// for example via sentry.ClientOptions.DebugWriter,
// because that would call the callback again for errors of that write.
//
// Example:
//
//	writer, err := logfile.NewRotatingWriterWithCallback(
//		"/var/log/app.log", "", 0644, 10*1024*1024,
//		logsentry.NewLogfileWriteCallback(sentry.CurrentHub(), time.Minute),
//	)
func NewLogfileWriteCallback(hub *sentry.Hub, minInterval time.Duration) logfile.WriteCallback {
	var (
		mtx     sync.Mutex
		lastAt  time.Time // Time of the last sent event
		skipped int       // Errors not sent since the last sent event
	)
	return func(filePath string, data []byte, n int, err error) {
		if err == nil {
			return
		}
		now := time.Now()
		mtx.Lock()
		if minInterval > 0 && !lastAt.IsZero() && now.Sub(lastAt) < minInterval {
			skipped++
			mtx.Unlock()
			return
		}
		lastAt = now
		skippedBefore := skipped
		skipped = 0
		mtx.Unlock()

		h := hub
		if h == nil {
			h = sentry.CurrentHub()
		}
		h.CaptureEvent(newLogfileErrorEvent(h, now, filePath, data, n, err, skippedBefore))
	}
}

// newLogfileErrorEvent returns the Sentry event
// that a callback from NewLogfileWriteCallback sends for err.
func newLogfileErrorEvent(hub *sentry.Hub, now time.Time, filePath string, data []byte, n int, err error, skipped int) *sentry.Event {
	_, rotateFailed := errors.AsType[*logfile.RotateError](err)
	writeFailed := n < len(data) || !rotateFailed

	event := sentry.NewEvent()
	event.Timestamp = now
	event.Level = sentry.LevelError
	if writeFailed {
		event.Message = "error writing log file"
	} else {
		event.Message = "error rotating log file"
	}
	event.Fingerprint = []string{event.Message, filePath}

	logCtx := sentry.Context{
		"file_path":     filePath,
		"bytes_total":   len(data),
		"bytes_written": n,
		"rotate_failed": rotateFailed,
		"write_failed":  writeFailed,
	}
	if n >= 0 && n < len(data) {
		logCtx["unwritten"] = unwrittenString(data[n:])
	}
	if skipped > 0 {
		logCtx["skipped_errors"] = skipped
	}
	event.Contexts["logfile"] = logCtx

	// Same stack trace resolution as Writer.CommitMessage:
	// prefer a stack trace carried by the error,
	// else the call stack if the client has AttachStacktrace enabled.
	// filterFrames removes the golog frames including logfile,
	// so the trace ends at the code that logged.
	stackTrace := sentry.ExtractStacktrace(err)
	if stackTrace == nil {
		if client := hub.Client(); client.IsEnabled() && client.Options().AttachStacktrace {
			stackTrace = sentry.NewStacktrace()
		}
	}
	if stackTrace != nil {
		stackTrace.Frames = filterFrames(stackTrace.Frames)
	}
	event.Exception = []sentry.Exception{{
		Type:       event.Message,
		Value:      err.Error(),
		Stacktrace: stackTrace,
	}}
	return event
}

// unwrittenString returns the start of unwritten log data
// with at most LogfileUnwrittenMaxBytes bytes as valid UTF-8 string
// without trailing newline.
// The returned string does not reference unwritten,
// so it can be kept after the WriteCallback returned.
func unwrittenString(unwritten []byte) string {
	truncated := len(unwritten) > LogfileUnwrittenMaxBytes
	if truncated {
		unwritten = unwritten[:max(LogfileUnwrittenMaxBytes, 0)]
	}
	s := strings.ToValidUTF8(string(unwritten), "")
	s = strings.TrimRight(s, "\n")
	if truncated {
		s += "…"
	}
	return s
}

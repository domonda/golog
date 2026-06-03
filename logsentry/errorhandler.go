package logsentry

import (
	"fmt"
	"io"
	"strings"

	"github.com/domonda/golog"
)

// Option configures a [WriterConfig] passed to [NewWriterConfig].
type Option func(*WriterConfig)

// WithErrorHandler sets the callback invoked for errors that occur while
// writing log messages to Sentry, such as recovered panics in the writer.
// This lets Sentry logging errors surface in the normal, non-Sentry logs
// instead of being lost.
//
// When no handler is set, errors are passed to [golog.ErrorHandler].
//
// The same callback can be passed to [NewSentryDebugWriter] to additionally
// capture Sentry's own asynchronous transport errors (e.g. HTTP 413, network
// failures), which never reach the writer.
func WithErrorHandler(fn func(error)) Option {
	return func(c *WriterConfig) { c.onError = fn }
}

// handleError reports err to the configured error handler, falling back to
// [golog.ErrorHandler] when none was set via [WithErrorHandler].
func (c *WriterConfig) handleError(err error) {
	reportError(c.onError, err)
}

// reportError sends err to onError, or to the current [golog.ErrorHandler]
// when onError is nil. Resolving golog.ErrorHandler at call time (rather than
// capturing it) honors any later [golog.SetErrorHandler] change.
func reportError(onError func(error), err error) {
	switch {
	case onError != nil:
		onError(err)
	case golog.ErrorHandler != nil:
		golog.ErrorHandler(err)
	}
}

// sentryErrorMarkers are lowercase substrings that identify Sentry debug log
// lines reporting a delivery problem (send failure, drop, or rate-limit), as
// opposed to routine informational output. See the debuglog.Printf calls in
// sentry-go's transport, client and internal/util packages.
var sentryErrorMarkers = []string{
	"fail",              // "...failed because the request was too large", "Failed to build envelope"
	"issue",             // "There was an issue with sending an event"
	"drop",              // "Event dropped due to transport buffer being full"
	"too large",         // HTTP 413
	"too many requests", // HTTP 429 backoff
	"unexpected",        // "Unexpected status code %d"
}

// sentryDebugWriter forwards Sentry's error-level debug output to onError.
type sentryDebugWriter struct {
	onError func(error)
}

// NewSentryDebugWriter returns an io.Writer to assign to
// sentry.ClientOptions.DebugWriter (with ClientOptions.Debug set to true). It
// forwards Sentry's internal delivery errors — which happen asynchronously in
// the transport and otherwise never reach golog — to onError, so they appear
// in the normal, non-Sentry logs. Routine (non-error) debug lines are ignored.
//
// When onError is nil, errors are passed to the current [golog.ErrorHandler].
// Pass the same callback used with [WithErrorHandler] to route both the
// writer's errors and Sentry's transport errors to one place.
//
// Because filtering relies on matching sentry-go's debug message text, treat it
// as best-effort: it may miss a renamed message or forward an unexpected line.
func NewSentryDebugWriter(onError func(error)) io.Writer {
	return &sentryDebugWriter{onError: onError}
}

func (w *sentryDebugWriter) Write(p []byte) (int, error) {
	for line := range strings.SplitSeq(string(p), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		for _, marker := range sentryErrorMarkers {
			if strings.Contains(lower, marker) {
				reportError(w.onError, fmt.Errorf("sentry: %s", line))
				break
			}
		}
	}
	return len(p), nil
}

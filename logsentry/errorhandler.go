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
// capturing it) honors any later [golog.SetErrorHandler] change. Either
// handler may be nil, in which case the error is dropped.
func reportError(onError func(error), err error) {
	if onError == nil {
		onError = golog.ErrorHandler()
	}
	if onError != nil {
		onError(err)
	}
}

// sentryErrorMarkers are lowercase substrings that identify sentry-go debug log
// lines reporting a genuine delivery failure or overload-induced drop, as
// opposed to routine output or intentional drops (sampling, BeforeSend, etc.).
// Compiled from the debuglog.Printf/Println calls in sentry-go v0.46.2's
// transport, telemetry scheduler, client and internal/util packages. Matched
// case-insensitively. A bare "fail"/"drop" keyword is deliberately avoided
// because it both misses the default async path's "error sending envelope" and
// matches benign lines like "Event dropped due to SampleRate hit".
var sentryErrorMarkers = []string{
	"error sending envelope",                   // async scheduler + transport send failure (default path)
	"http request failed",                      // transport HTTP failure
	"there was an issue",                       // request build / send issue
	"failed because the request was too large", // HTTP 413
	"failed with server error",                 // HTTP 5xx
	"failed with client error",                 // HTTP 4xx
	"too many requests",                        // HTTP 429 backoff
	"rate limited for category",                // rate-limit drop
	"unexpected status code",                   //
	"skipping delivery",                        // "Failed to build/convert envelope, skipping delivery"
	"could not encode event as json",           // serialization failure -> delivery skipped
	"error while converting to envelope",       // envelope item conversion failure
	"error creating",                           // "error creating log/trace metric batch envelope item"
	"buffer being full",                        // "Event dropped due to transport buffer being full"
	"buffer full",                              // "Dropping log/metric: buffer full", telemetry buffer full
	"failed to flush",                          // flush timed out / transport closed
	"failed to send client report",             //
	"failed to serialize client report",        //
}

// sentryIgnoreMarkers identify intentional, non-error sentry-go debug lines
// (sampling, BeforeSend/Ignore filters, callbacks) that must never be forwarded
// even if a future message rewording makes one match an error marker. Defense
// in depth: with the current marker set none of these match anyway.
var sentryIgnoreMarkers = []string{
	"samplerate",         // "Event dropped due to SampleRate hit"
	"beforesend",         // BeforeSend / BeforeSendLog / BeforeSendMetric / BeforeSendTransaction
	"beforebreadcrumb",   //
	"ignoreerrors",       //
	"ignoretransactions", //
	"eventprocessors",    // "Event dropped by one of the ... EventProcessors"
	"tracessampler",      //
	"callback",           //
}

// sentryDebugWriter forwards Sentry's error-level debug output to onError.
type sentryDebugWriter struct {
	onError func(error)
}

// NewSentryDebugWriter returns an io.Writer to assign to
// sentry.ClientOptions.DebugWriter (with ClientOptions.Debug set to true). It
// forwards Sentry's internal delivery failures — which happen asynchronously in
// the transport and otherwise never reach golog — to onError, so they appear
// in the normal, non-Sentry logs. Routine output and intentional drops
// (sampling, BeforeSend filtering) are ignored.
//
// When onError is nil, errors are passed to the current [golog.ErrorHandler].
// Pass the same callback used with [WithErrorHandler] to route both the
// writer's errors and Sentry's transport errors to one place.
//
// Important: onError must not log back through the same Sentry-backed golog
// logger. A delivery failure forwarded to such a handler would emit a new
// event whose own failure produces another debug line, amplifying into a
// feedback loop under sustained backpressure. Route it to a plain logger or to
// the default [golog.ErrorHandler] (stderr).
//
// Because filtering relies on matching sentry-go's debug message text, treat it
// as best-effort: a renamed message in a future sentry-go release may be missed
// (and should be added to sentryErrorMarkers).
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
		if containsAny(lower, sentryIgnoreMarkers) {
			continue
		}
		if containsAny(lower, sentryErrorMarkers) {
			reportError(w.onError, fmt.Errorf("sentry: %s", line))
		}
	}
	return len(p), nil
}

// containsAny reports whether s contains any of the substrings in subs.
func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

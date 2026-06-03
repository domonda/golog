package golog

import (
	"fmt"
	"os"
	"sync/atomic"
)

// errorHandler holds the current error handler function. It is stored behind an
// atomic pointer so [ErrorHandler], [SetErrorHandler] and [ErrorHandlerOr] are
// safe for concurrent use. The pointed-to func may be nil (a valid value that
// disables error handling).
var errorHandler atomic.Pointer[func(error)]

func init() {
	def := func(err error) {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}
	errorHandler.Store(&def)
}

// ErrorHandler returns the function called when an error occurs while writing
// logs. The default handler prints to stderr. The returned func is nil if the
// handler was disabled via SetErrorHandler(nil); callers must nil-check it (or
// use [ErrorHandlerOr]).
func ErrorHandler() func(error) {
	if p := errorHandler.Load(); p != nil {
		return *p
	}
	return nil
}

// SetErrorHandler sets the function called when an error occurs while writing
// logs. It is safe for concurrent use. A nil handler is valid and disables
// error handling.
func SetErrorHandler(handler func(error)) {
	errorHandler.Store(&handler)
}

// ErrorHandlerOr returns the configured error handler, or fallback if the
// configured handler is nil.
func ErrorHandlerOr(fallback func(error)) func(error) {
	if h := ErrorHandler(); h != nil {
		return h
	}
	return fallback
}

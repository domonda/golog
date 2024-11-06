package golog

import (
	"fmt"
	"runtime"
	"strings"
)

// TrimCallStackPathPrefix will be trimmed from the
// beginning of every call-stack file-path.
// Defaults to $GOPATH/src/ of the build environment
// or will be empty if go build gets called with -trimpath.
var TrimCallStackPathPrefix = filePathPrefix()

func filePathPrefix() string {
	// This Go package is hosted on GitHub
	// so there should always be "github.com"
	// in the path of this source file
	// if it was cloned using standard go get
	_, file, _, _ := runtime.Caller(1)
	end := strings.Index(file, "github.com")
	if end == -1 {
		panic("expected github.com in call-stack file-path, but got: " + file)
	}
	return file[:end]
}

func callstack(skip int) string {
	skip = max(2+skip, 0) // Prefer robustness in logging over negative index panics
	stack := make([]uintptr, 32)
	n := runtime.Callers(skip, stack)
	stack = stack[:n]

	var b strings.Builder
	frames := runtime.CallersFrames(stack)
	for {
		frame, _ := frames.Next()
		if frame.Function == "" || strings.HasPrefix(frame.Function, "runtime.") {
			break
		}
		_, _ = fmt.Fprintf(
			&b,
			"%s\n    %s:%d\n",
			frame.Function,
			strings.TrimPrefix(frame.File, TrimCallStackPathPrefix),
			frame.Line,
		)
	}
	return b.String()
}

// CallingFunction returns the name of the function that called it.
func CallingFunction() string {
	stack := make([]uintptr, 1)
	n := runtime.Callers(2, stack)
	if n == 0 {
		// Should never happen, but better safe than sory because of a panic
		return "UNKNOWN"
	}
	return runtime.FuncForPC(stack[0]).Name()
}

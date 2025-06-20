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
	end := strings.LastIndex(file, "github.com")
	if end == -1 {
		// panic("expected github.com in call-stack file-path, but got: " + file)
		return "" // GitHub action might have it under /home/runner/work/...
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

// CallingFunction returns the fully qualified name
// including the package import path of the calling function.
//
// The sum of the optional skipFrames
// callstack frames will be skipped.
func CallingFunction(skipFrames ...int) string {
	skip := 2 // This function plus runtime.Callers
	for _, n := range skipFrames {
		skip += n
	}
	var stack [1]uintptr
	n := runtime.Callers(skip, stack[:])
	if n == 0 {
		return "" // Should never happen, but better safe than sorry because of a panic
	}
	return runtime.FuncForPC(stack[0]).Name()
}

// CallingFunctionName returns the name of the calling function
// without the package prefix.
//
// The sum of the optional skipFrames
// callstack frames will be skipped.
func CallingFunctionName(skipFrames ...int) string {
	skip := 2 // This function plus runtime.Callers
	for _, n := range skipFrames {
		skip += n
	}
	var stack [1]uintptr
	n := runtime.Callers(skip, stack[:])
	if n == 0 {
		return "" // Should never happen, but better safe than sorry because of a panic
	}
	name := runtime.FuncForPC(stack[0]).Name()
	return name[strings.LastIndex(name, ".")+1:]
}

// CallingFunctionPackageName returns the name of the calling function
// without the package prefix.
//
// The sum of the optional skipFrames
// callstack frames will be skipped.
func CallingFunctionPackageName(skipFrames ...int) string {
	skip := 2 // This function plus runtime.Callers
	for _, n := range skipFrames {
		skip += n
	}
	var stack [1]uintptr
	runtime.Callers(skip, stack[:])
	name := runtime.FuncForPC(stack[0]).Name()
	name = name[strings.LastIndexByte(name, '/')+1:]
	return name[:strings.IndexByte(name, '.')]
}

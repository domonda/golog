package golog

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

// failingWriter is an io.Writer that always fails.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestErrorHandlerOr(t *testing.T) {
	prev := ErrorHandler()
	t.Cleanup(func() { SetErrorHandler(prev) })

	var called atomic.Int32
	fallback := func(error) { called.Add(1) }

	SetErrorHandler(nil)
	require.Nil(t, ErrorHandler())
	ErrorHandlerOr(fallback)(errors.New("error"))
	require.Equal(t, int32(1), called.Load(), "fallback called for nil handler")

	SetErrorHandler(func(error) {})
	ErrorHandlerOr(fallback)(errors.New("error"))
	require.Equal(t, int32(1), called.Load(), "fallback not called for set handler")
}

// TestSetErrorHandler_Concurrent changes the error handler
// while other goroutines log to a failing writer, for go test -race.
func TestSetErrorHandler_Concurrent(t *testing.T) {
	const (
		numGoroutines = 8
		numMessages   = 100
	)
	prev := ErrorHandler()
	t.Cleanup(func() { SetErrorHandler(prev) })

	var numHandled atomic.Int32
	handler := func(error) { numHandled.Add(1) }
	SetErrorHandler(handler)
	logger := NewLogger(NewConfig(&DefaultLevels, AllLevelsActive,
		NewJSONWriterConfig(failingWriter{}, nil),
		NewTextWriterConfig(failingWriter{}, nil, nil),
	))

	done := make(chan struct{})
	var setterWG sync.WaitGroup
	setterWG.Go(func() {
		for i := 0; ; i++ {
			select {
			case <-done:
				return
			default:
			}
			if i%2 == 0 {
				SetErrorHandler(nil)
			} else {
				SetErrorHandler(handler)
			}
		}
	})
	var wg sync.WaitGroup
	for range numGoroutines {
		wg.Go(func() {
			for i := range numMessages {
				logger.Info("msg").Int("i", i).Log()
			}
		})
	}
	wg.Wait()
	close(done)
	setterWG.Wait()

	// Two failing writers per message, each error is handled
	// unless the handler was nil at the time of the error
	require.LessOrEqual(t, numHandled.Load(), int32(2*numGoroutines*numMessages))
}

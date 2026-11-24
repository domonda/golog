package logfile_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/domonda/golog/logfile"
)

// ExampleRotatingWriter_SetWriteCallback shows how to handle failed writes
// asynchronously, so slow work like sending an alarm over the network
// does not delay the code that is logging:
//
//   - One worker goroutine handles the failed writes one after another,
//     in the order the callbacks were called (with concurrent writes
//     not necessarily the order of the writes), instead of starting
//     a goroutine per write which would start an unlimited number
//     of goroutines when every write fails because the disk is full.
//   - The callback copies only the unwritten part data[n:],
//     because the caller of Write reuses the slice.
//   - The callback never blocks: when the channel buffer is full,
//     it counts the failed write as dropped instead of waiting.
//   - The worker recovers panics, because the RotatingWriter only recovers
//     panics in the callback itself, not in goroutines started by it.
//   - The channel is never closed, because a Write running concurrently
//     with the shutdown can still call the callback, and sending on a closed
//     channel panics. The worker is stopped with a separate channel instead.
//   - The worker does not log to the RotatingWriter it handles errors for.
func ExampleRotatingWriter_SetWriteCallback() {
	dir, err := os.MkdirTemp("", "logfile-example")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	writer, err := logfile.NewRotatingWriter(filepath.Join(dir, "app.log"), "", 0o600, 10*1024*1024)
	if err != nil {
		panic(err)
	}

	type failedWrite struct {
		filePath string
		data     []byte
		err      error
	}
	var (
		failedWrites = make(chan failedWrite, 100) // Buffer for bursts of failed writes
		dropped      atomic.Int64                  // Failed writes that did not fit into the buffer
		stopWorker   = make(chan struct{})
		workerDone   = make(chan struct{})
	)

	// sendAlarm stands in for slow work like an HTTP request to an alerting service
	sendAlarm := func(w failedWrite) {
		fmt.Printf("alarm: writing %s failed (file closed: %t): %s", filepath.Base(w.filePath), errors.Is(w.err, os.ErrClosed), w.data)
	}
	handleFailedWrite := func(w failedWrite) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintln(os.Stderr, "panic while handling failed log write:", r)
			}
		}()
		sendAlarm(w)
	}

	go func() {
		defer close(workerDone)
		for {
			select {
			case w := <-failedWrites:
				handleFailedWrite(w)
			case <-stopWorker:
				// Handle the failed writes that are still buffered
				for {
					select {
					case w := <-failedWrites:
						handleFailedWrite(w)
					default:
						return
					}
				}
			}
		}
	}()

	writer.SetWriteCallback(func(filePath string, data []byte, n int, err error) {
		if err == nil {
			return
		}
		// data[:n] reached the file, so only pass on the unwritten rest
		select {
		case failedWrites <- failedWrite{filePath: filePath, data: bytes.Clone(data[n:]), err: err}:
		default:
			dropped.Add(1)
		}
	})

	_, _ = writer.Write([]byte("written to the file\n"))

	// Writing after Close fails like writing to a full disk would
	_ = writer.Close()
	_, _ = writer.Write([]byte("not written to the file\n"))

	// Shutdown: remove the callback, then stop the worker
	// after it has handled the buffered failed writes
	writer.SetWriteCallback(nil)
	close(stopWorker)
	<-workerDone
	fmt.Println("dropped failed writes:", dropped.Load())

	// Output:
	// alarm: writing app.log failed (file closed: true): not written to the file
	// dropped failed writes: 0
}

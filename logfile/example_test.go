package logfile_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/domonda/golog"
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

// ExampleFreeDiskSpace shows a WriteCallback that warns when the volume
// of the log file runs out of space, so there is time to act
// before writes fail because the disk is full:
//
//   - Low disk space is reported only once, so the alarm is not repeated
//     for every write. The atomic Swap lets only one of the goroutines
//     that write concurrently report it.
//   - The free space is checked after every write until it has been reported,
//     which costs one or two system calls per write.
//   - The threshold leaves room for 100 rotated files,
//     because the rotated files keep using space until they are deleted.
//   - The warning is printed, not logged to the logger that writes to the file,
//     because writing to the same RotatingWriter from its callback
//     calls the callback again. Hand slow work like sending an alarm
//     over the network to a worker goroutine, as the example
//     of RotatingWriter.SetWriteCallback shows.
//
// The example simulates a volume that loses 1 MiB of free space per write
// with MockFreeDiskSpace, which only the tests of this package can call.
// In production code FreeDiskSpace returns the real free space.
func ExampleFreeDiskSpace() {
	const (
		mib          = 1024 * 1024
		rotateSize   = 10 * mib   // Rotate at 10 MiB
		minFreeSpace = 1024 * mib // Warn when less than 1 GiB is free
	)

	// Simulate a volume with 2 MiB more than minFreeSpace free
	// that loses 1 MiB with every check after a write
	var simulatedFree atomic.Int64
	simulatedFree.Store(minFreeSpace + 2*mib)
	restore := logfile.MockFreeDiskSpace(func(string) (int64, error) {
		return simulatedFree.Add(-mib), nil
	})
	defer restore()

	dir, err := os.MkdirTemp("", "logfile-example")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	var reported atomic.Bool // Report only once, so the alarm is not repeated for every write

	writer, err := logfile.NewRotatingWriterWithCallback(
		filepath.Join(dir, "app.log"), "", 0o600, rotateSize,
		func(filePath string, data []byte, n int, err error) {
			if reported.Load() {
				return
			}
			free, checkErr := logfile.FreeDiskSpace(filePath)
			if checkErr == nil && free >= minFreeSpace {
				return
			}
			if reported.Swap(true) {
				return // Another goroutine reported it at the same time
			}
			if checkErr != nil {
				fmt.Println("can't check free disk space for the log file:", checkErr)
				return
			}
			fmt.Printf("the volume of %s has only %d MiB free\n", filepath.Base(filePath), free/mib)
		},
	)
	if err != nil {
		panic(err)
	}
	defer writer.Close()

	logger := golog.NewLogger(golog.NewConfig(
		&golog.DefaultLevels,
		golog.AllLevelsActive,
		golog.NewJSONWriterConfig(writer, nil),
	))
	for i := range 5 {
		logger.Info("request handled").Int("request", i).Log()
	}

	// Output:
	// the volume of app.log has only 1023 MiB free
}

// ExampleFreeDiskSpace_opsHubWarning shows a WriteCallback that warns
// the operations team via an ops hub, like a chat channel, a pager
// or a monitoring service, when the log files fill up their volume:
//
//   - The free space is checked after every write and the warning
//     is sent only once, like in ExampleFreeDiskSpace.
//   - The warning threshold is a multiple of the rotation size,
//     because the rotated files keep using space until they are deleted.
//   - The warning tells how many more rotated files fit,
//     which is what the ops team needs to decide how urgent it is.
//
// The example simulates a volume of 10,000 bytes that only the log files use,
// with MockFreeDiskSpace, which only the tests of this package can call.
// In production code FreeDiskSpace returns the real free space
// and the volume is gigabytes large.
func ExampleFreeDiskSpace_opsHubWarning() {
	dir, err := os.MkdirTemp("", "logfile-example")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	// Simulate a small volume that fills up with the log files in dir
	const volumeSize = 10_000
	restore := logfile.MockFreeDiskSpace(func(string) (int64, error) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return 0, err
		}
		free := int64(volumeSize)
		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				return 0, err
			}
			free -= info.Size()
		}
		return max(free, 0), nil
	})
	defer restore()

	const (
		rotateSize   = 1000           // Rotate at 1000 bytes
		minFreeSpace = 5 * rotateSize // Warn when less than 5 more rotated files fit
	)

	// sendOpsHubWarning stands in for sending a warning to the ops hub.
	// Hand slow work like a network request to a worker goroutine,
	// as the example of RotatingWriter.SetWriteCallback shows.
	sendOpsHubWarning := func(msg string) {
		fmt.Println("ops hub warning:", msg)
	}

	var reported atomic.Bool // Warn only once, so the ops hub is not spammed for every write
	writer, err := logfile.NewRotatingWriterWithCallback(
		filepath.Join(dir, "app.log"),
		"20060102T150405", // Without colons, which Windows does not allow in file names
		0o600,             // filePerm
		rotateSize,
		func(filePath string, data []byte, n int, err error) {
			if reported.Load() {
				return
			}
			free, checkErr := logfile.FreeDiskSpace(filePath)
			if checkErr == nil && free >= minFreeSpace {
				return
			}
			if reported.Swap(true) {
				return // Another goroutine warned at the same time
			}
			if checkErr != nil {
				sendOpsHubWarning(fmt.Sprintf("can't check the free space of the log volume: %s", checkErr))
				return
			}
			sendOpsHubWarning(fmt.Sprintf("the volume of %s has %d bytes free, room for %d more rotated log files",
				filepath.Base(filePath), free, free/rotateSize))
		},
	)
	if err != nil {
		panic(err)
	}
	defer writer.Close()

	// Write 80 log lines of 100 bytes, 8000 bytes in total
	line := []byte(strings.Repeat("-", 99) + "\n")
	for range 80 {
		_, _ = writer.Write(line)
	}

	// Output:
	// ops hub warning: the volume of app.log has 4900 bytes free, room for 4 more rotated log files
}

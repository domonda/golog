/*
Package logfile provides file-based log writers with automatic rotation capabilities.

The main type is RotatingWriter, which implements io.WriteCloser and automatically
rotates log files when they reach a specified size threshold. This is useful for
managing disk space in long-running applications and preventing log files from
growing indefinitely.

# Basic Usage

	writer, err := logfile.NewRotatingWriter(
		"/var/log/app.log",                      // File path
		logfile.RotatingWriterDefaultTimeFormat, // Time format for rotated files
		0644,                                    // File permissions
		10*1024*1024,                            // Rotate at 10MB
	)
	if err != nil {
		log.Fatal(err)
	}
	defer writer.Close()

	// Use with golog
	config := golog.NewConfig(
		&golog.DefaultLevels,
		golog.AllLevelsActive,
		golog.NewJSONWriterConfig(writer, nil),
	)

When the log file reaches the specified size, it is automatically rotated:
  - The current file is renamed with a timestamp suffix (e.g., app.log.2024-01-15_10:30:45)
  - A new file is created at the original path
  - If a rotated file with the same timestamp exists, a numeric suffix is added

If the rotation fails, for example because the directory of the log file
is not writable, the data is still written to the current log file,
which keeps growing past the rotation size, and Write returns a *RotateError.
The rotation is retried no earlier than one minute later.

# Write Callback

NewRotatingWriterWithCallback creates a writer with a WriteCallback,
and SetWriteCallback sets or removes it later.
The callback is called after every Write,
including writes that rotated the file first. It receives the log file path,
the bytes that were passed to Write, the number of those bytes written
to the file, and the error of the rotation or write, or nil on success.
Use it to report log write errors by other means, to retry or redirect
the unwritten part of failed writes, or to check the free disk space
and send an alarm when the log volume fills up.
Use errors.As with a *RotateError to tell a failed rotation
apart from a failed write:

	writer.SetWriteCallback(func(filePath string, data []byte, n int, err error) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error writing log file %s: %s\n%s", filePath, err, data[n:])
		}
	})

# Thread Safety

RotatingWriter is safe for concurrent use by multiple goroutines. All write
operations are protected by an internal mutex.
*/
package logfile

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/domonda/golog"
)

// RotatingWriterDefaultTimeFormat is the default time format used for naming rotated log files.
// It produces filenames like: original.log.2006-01-02_15:04:05
const RotatingWriterDefaultTimeFormat = "2006-01-02_15:04:05"

// rotateRetryInterval is the minimum time between a failed rotation
// and the next attempt, so that a log file that can't be renamed
// does not cost extra system calls and an error for every Write.
const rotateRetryInterval = time.Minute

// renameFile renames the log file when rotating it.
// It is a variable so tests can make renaming fail.
var renameFile = os.Rename

// statFile returns the file info of an opened log file.
// It is a variable so tests can make it fail.
var statFile = (*os.File).Stat

// RotateError is the error of a failed log file rotation,
// returned by RotatingWriter.Write and passed to the WriteCallback.
// It wraps the error of closing or renaming the log file.
//
// A failed rotation does not lose the written data:
// it is still written to the current log file, which keeps growing
// past the rotation size until a later rotation succeeds.
// The rotation is retried no earlier than one minute after the failure.
//
// Write returns a *RotateError as is when the data was written completely,
// and joined with errors.Join with the error of opening or writing
// the log file when that failed too.
// Use errors.As to detect a failed rotation,
// and the byte count n returned by Write to detect a failed write:
//
//	var rotateErr *logfile.RotateError
//	if errors.As(err, &rotateErr) {
//		// The log file could not be rotated and grows past the rotation size
//	}
//	if n < len(data) {
//		// data[n:] was not written
//	}
type RotateError struct {
	// FilePath is the path of the log file that could not be rotated
	FilePath string
	// Err is the error of closing or renaming the log file
	Err error
}

// Error implements the error interface.
func (e *RotateError) Error() string {
	return fmt.Sprintf("error rotating log file %q: %s", e.FilePath, e.Err)
}

// Unwrap returns the error of closing or renaming the log file.
func (e *RotateError) Unwrap() error {
	return e.Err
}

// WriteCallback is called by RotatingWriter after every Write call
// with the path of the log file, the data passed to Write,
// and the byte count n and error returned by Write.
// On success n is len(data) and err is nil.
// If rotating the file before the write failed, err is or wraps
// a *RotateError (detect it with errors.As) and data is still written
// to the current log file, so n is len(data) unless writing failed too.
// If opening or writing the log file failed, data[:n] was written
// and data[n:] was not, so n < len(data) always means a failed write.
// To retry or redirect a failed write, use data[n:] so that
// the part that already reached the file is not written twice.
//
// The callback is called after the writer's mutex has been released,
// so it can call methods of the RotatingWriter like Sync.
// It is called concurrently when Write is called concurrently.
// A panic in the callback is recovered and passed as error
// to the golog.ErrorHandler, it does not change the result of Write.
// It must not write to the same RotatingWriter, for example by logging
// to a logger that uses it, because every Write calls the callback again,
// which results in an endless recursion.
//
// The callback runs synchronously in the goroutine that called Write,
// so Write does not return before the callback returns.
// Other goroutines can write in the meantime because the mutex is released.
// A callback that does slow work, like sending an alarm over the network,
// should hand it to a goroutine so it does not delay the logging code.
// Prefer a single worker goroutine fed by a buffered channel
// over a goroutine per call, because every write fails when the disk is full.
// Panics in such goroutines are not recovered by the RotatingWriter.
// See the example of RotatingWriter.SetWriteCallback.
//
// Like io.Writer implementations, the callback must not retain data
// after it returns because the caller of Write may reuse the slice.
// Copy data with bytes.Clone before passing it to a goroutine
// or keeping it in any other way, for example to retry the write later.
type WriteCallback func(filePath string, data []byte, n int, err error)

// RotatingWriter implements io.WriteCloser and provides automatic log file rotation
// based on file size. When the file reaches the configured size threshold, it is
// renamed with a timestamp suffix and a new file is created.
//
// The writer is thread-safe and can be used concurrently from multiple goroutines.
//
// Example:
//
//	writer, err := logfile.NewRotatingWriter("/var/log/app.log", "", 0644, 10*1024*1024)
//	if err != nil {
//		return err
//	}
//	defer writer.Close()
type RotatingWriter struct {
	mtx sync.Mutex

	filePath   string      // Path to the log file
	timeFormat string      // Time format for rotated file names
	filePerm   os.FileMode // File permissions
	file       *os.File    // Currently open log file, nil after a rotation until the next write opens it
	size       int64       // Current size of the log file in bytes
	rotateSize int64       // Size threshold for rotation in bytes
	closed     bool        // Set by Close, so writes don't open the file again

	rotateRetryAt time.Time // No rotation before this time after a failed rotation

	writeCallback WriteCallback // Called after every Write, can be nil
}

// NewRotatingWriter creates a new RotatingWriter that writes to the specified file path.
//
// Parameters:
//   - filePath: Path to the log file. If the file doesn't exist, it will be created.
//     If it exists, logs will be appended to it.
//   - timeFormat: Time format for naming rotated files (Go time layout string).
//     Pass empty string to use RotatingWriterDefaultTimeFormat.
//   - filePerm: File permissions (e.g., 0644) for the log file.
//   - rotateSize: Size threshold in bytes. When the file reaches or exceeds this size,
//     it will be rotated. Pass 0 to disable rotation (file will grow indefinitely).
//
// Example:
//
//	// Rotate at 10MB
//	writer, err := NewRotatingWriter("/var/log/app.log", "", 0644, 10*1024*1024)
func NewRotatingWriter(filePath, timeFormat string, filePerm os.FileMode, rotateSize int64) (*RotatingWriter, error) {
	return NewRotatingWriterWithCallback(filePath, timeFormat, filePerm, rotateSize, nil)
}

// NewRotatingWriterWithCallback creates a new RotatingWriter like NewRotatingWriter
// with writeCallback already set, so it is called from the first Write on.
// Passing nil for writeCallback is the same as calling NewRotatingWriter.
// See WriteCallback for the rules a callback must follow
// and SetWriteCallback to change or remove it later.
//
// Example:
//
//	writer, err := NewRotatingWriterWithCallback("/var/log/app.log", "", 0644, 10*1024*1024,
//		func(filePath string, data []byte, n int, err error) {
//			if err != nil {
//				fmt.Fprintf(os.Stderr, "error writing log file %s: %s\n%s", filePath, err, data[n:])
//			}
//		},
//	)
func NewRotatingWriterWithCallback(filePath, timeFormat string, filePerm os.FileMode, rotateSize int64, writeCallback WriteCallback) (*RotatingWriter, error) {
	filePath = filepath.Clean(filePath)
	timeFormat = cmp.Or(timeFormat, RotatingWriterDefaultTimeFormat)
	file, size, err := openFile(filePath, filePerm)
	if err != nil {
		return nil, err
	}
	return &RotatingWriter{
		filePath:      filePath,
		timeFormat:    timeFormat,
		filePerm:      filePerm,
		file:          file,
		size:          size,
		rotateSize:    rotateSize,
		writeCallback: writeCallback,
	}, nil
}

func openFile(filePath string, filePerm os.FileMode) (file *os.File, size int64, err error) {
	file, err = os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, filePerm) //#nosec G304
	if err != nil {
		return nil, 0, fmt.Errorf("error opening rotating log file %q: %w", filePath, err)
	}
	stat, err := statFile(file)
	if err != nil {
		_ = file.Close()
		return nil, 0, fmt.Errorf("error getting size of rotating log file %q: %w", filePath, err)
	}
	return file, stat.Size(), nil
}

// FilePath returns the path to the log file.
func (rw *RotatingWriter) FilePath() string {
	return rw.filePath
}

// TimeFormat returns the time format string used for naming rotated files.
func (rw *RotatingWriter) TimeFormat() string {
	return rw.timeFormat
}

// RotateSize returns the size threshold in bytes at which the file will be rotated.
// Returns 0 if rotation is disabled.
func (rw *RotatingWriter) RotateSize() int64 {
	return rw.rotateSize
}

// SetWriteCallback sets a callback that is called after every Write
// with the log file path, the data passed to Write,
// and the byte count and error returned by Write.
// Pass nil to remove the callback. See WriteCallback for details.
//
// A Write that is already running when SetWriteCallback is called
// may still call the previous callback after SetWriteCallback returned.
//
// The method is thread-safe and can be called concurrently from multiple goroutines.
func (rw *RotatingWriter) SetWriteCallback(callback WriteCallback) {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()

	rw.writeCallback = callback
}

// Write writes the given bytes to the log file and implements io.Writer.
// If writing would cause the file to exceed the rotation size threshold,
// the file is automatically rotated before writing.
//
// If rotating the file fails, msg is still written to the current log file,
// which keeps growing past the rotation size, and a *RotateError is returned
// with n == len(msg), or joined with the error of the write if that failed too.
// The rotation is retried by the first Write that happens
// at least one minute after the failure.
//
// If no log file is open because opening it after a rotation failed,
// Write tries to open it again before writing msg.
//
// After the write, the callback set with SetWriteCallback is called
// with the log file path, msg and the returned byte count and error.
//
// The method is thread-safe and can be called concurrently from multiple goroutines.
func (rw *RotatingWriter) Write(msg []byte) (n int, err error) {
	callback, n, err := rw.writeFile(msg)
	if callback == nil {
		return n, err
	}

	// The mutex is already released here, so the callback can use the writer
	defer recoverWriteCallbackPanic(rw.filePath)
	callback(rw.filePath, msg, n, err)
	return n, err
}

// writeFile writes msg to the file, rotating the file first if needed,
// while holding the mutex. It also returns the write callback,
// read under the same lock, so Write can call it after releasing the mutex.
func (rw *RotatingWriter) writeFile(msg []byte) (callback WriteCallback, n int, err error) {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()

	callback = rw.writeCallback
	if rw.closed {
		return callback, 0, &os.PathError{Op: "write", Path: rw.filePath, Err: os.ErrClosed}
	}

	// size is incremented optimistically before the write.
	// If the write fails (e.g. disk full), size drifts ahead
	// of the actual file size, causing earlier-than-needed rotations.
	// This is acceptable because opening the file after a rotation
	// resets size to the actual file size via stat, self-correcting the drift.
	rw.size += int64(len(msg))
	var rotateErr *RotateError
	// Without an open file, because opening it failed at an earlier write,
	// there is nothing to rotate: the file is opened below,
	// which resets size, and the next write rotates it if needed
	if rw.file != nil && rw.rotateSize > 0 && rw.size >= rw.rotateSize {
		now := time.Now()
		if !now.Before(rw.rotateRetryAt) {
			rotateErr = rw.rotate(now)
			if rotateErr != nil {
				rw.rotateRetryAt = now.Add(rotateRetryInterval)
			}
		}
	}

	if rw.file == nil {
		// Open the file after a rotation, or again if that failed
		// at an earlier write, so a transient error does not
		// break the writer permanently
		file, size, openErr := openFile(rw.filePath, rw.filePerm)
		if openErr != nil {
			return callback, 0, joinRotateError(rotateErr, openErr)
		}
		rw.file = file
		rw.size = size + int64(len(msg))
	}

	n, err = rw.file.Write(msg)
	return callback, n, joinRotateError(rotateErr, err)
}

// joinRotateError returns rotateErr and err joined,
// or the one that is not nil, or nil if both are nil.
// It takes rotateErr as *RotateError to avoid returning
// a nil *RotateError as non-nil error interface.
func joinRotateError(rotateErr *RotateError, err error) error {
	switch {
	case rotateErr == nil:
		return err
	case err == nil:
		return rotateErr
	default:
		return errors.Join(rotateErr, err)
	}
}

// recoverWriteCallbackPanic recovers a panic from a WriteCallback
// and passes it as error to the golog.ErrorHandler.
// It must be called directly by defer to be able to recover.
func recoverWriteCallbackPanic(filePath string) {
	r := recover()
	if r == nil {
		return
	}
	if handleErr := golog.ErrorHandler(); handleErr != nil {
		handleErr(fmt.Errorf("logfile.RotatingWriter recovered panic from WriteCallback for %q: %v\n%s", filePath, r, debug.Stack()))
	}
}

// rotate closes the log file and renames it to a rotated file name.
// It always leaves rw.file nil, also when it fails,
// because os.File.Close releases the file even when it returns an error.
// writeFile then opens a new file at rw.filePath,
// or the old file again if closing or renaming it failed.
func (rw *RotatingWriter) rotate(now time.Time) *RotateError {
	err := rw.file.Close()
	rw.file = nil
	if err != nil {
		return &RotateError{FilePath: rw.filePath, Err: err}
	}
	err = renameFile(rw.filePath, rw.rotatedFilePath(now))
	if err != nil {
		return &RotateError{FilePath: rw.filePath, Err: err}
	}
	return nil
}

func (rw *RotatingWriter) rotatedFilePath(now time.Time) string {
	rotatedBase := rw.filePath + "." + now.Format(rw.timeFormat)

	rotated := rotatedBase
	for i := 1; fileExists(rotated); i++ {
		rotated = rotatedBase + "." + strconv.Itoa(i)
	}

	return rotated
}

// Sync flushes the file to disk by calling os.File.Sync.
// This ensures that all buffered data is written to the underlying storage device.
//
// The method is thread-safe and can be called concurrently from multiple goroutines.
func (rw *RotatingWriter) Sync() error {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()

	if rw.file == nil {
		// The file was closed by a rotation, the next Write opens it
		return nil
	}
	return rw.file.Sync()
}

// Close closes the underlying log file and implements io.Closer.
// After calling Close, the RotatingWriter should not be used.
//
// The method is thread-safe and can be called concurrently from multiple goroutines.
func (rw *RotatingWriter) Close() error {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()

	rw.closed = true
	if rw.file == nil {
		// The file was closed by a rotation and not opened again
		return nil
	}
	return rw.file.Close()
}

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

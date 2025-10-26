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

# Thread Safety

RotatingWriter is safe for concurrent use by multiple goroutines. All write
operations are protected by an internal mutex.
*/
package logfile

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// RotatingWriterDefaultTimeFormat is the default time format used for naming rotated log files.
// It produces filenames like: original.log.2006-01-02_15:04:05
const RotatingWriterDefaultTimeFormat = "2006-01-02_15:04:05"

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
	file       *os.File    // Currently open log file
	size       int64       // Current size of the log file in bytes
	rotateSize int64       // Size threshold for rotation in bytes
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
	filePath = filepath.Clean(filePath)
	timeFormat = cmp.Or(timeFormat, RotatingWriterDefaultTimeFormat)
	file, size, err := openFile(filePath, filePerm)
	if err != nil {
		return nil, err
	}
	return &RotatingWriter{
		filePath:   filePath,
		timeFormat: timeFormat,
		filePerm:   filePerm,
		file:       file,
		size:       size,
		rotateSize: rotateSize,
	}, nil
}

func openFile(filePath string, filePerm os.FileMode) (file *os.File, size int64, err error) {
	file, err = os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, filePerm) //#nosec G304
	if err != nil {
		return nil, 0, fmt.Errorf("error opening rotating log file %q: %w", filePath, err)
	}
	stat, err := file.Stat()
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

// Write writes the given bytes to the log file and implements io.Writer.
// If writing would cause the file to exceed the rotation size threshold,
// the file is automatically rotated before writing.
//
// The method is thread-safe and can be called concurrently from multiple goroutines.
func (rw *RotatingWriter) Write(msg []byte) (n int, err error) {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()

	rw.size += int64(len(msg))
	if rw.rotateSize > 0 && rw.size >= rw.rotateSize {
		err := rw.rotate()
		if err != nil {
			return 0, err
		}
		rw.size += int64(len(msg))
	}

	return rw.file.Write(msg)
}

func (rw *RotatingWriter) rotate() error {
	err := rw.file.Close()
	if err != nil {
		return fmt.Errorf("error closing rotating log file %q: %w", rw.filePath, err)
	}

	err = os.Rename(rw.filePath, rw.rotatedFilePath())
	if err != nil {
		return err
	}

	file, size, err := openFile(rw.filePath, rw.filePerm)
	if err != nil {
		return err
	}
	rw.file = file
	rw.size = size
	return nil
}

func (rw *RotatingWriter) rotatedFilePath() string {
	rotatedBase := rw.filePath + "." + time.Now().Format(rw.timeFormat)

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

	return rw.file.Sync()
}

// Close closes the underlying log file and implements io.Closer.
// After calling Close, the RotatingWriter should not be used.
//
// The method is thread-safe and can be called concurrently from multiple goroutines.
func (rw *RotatingWriter) Close() error {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()

	return rw.file.Close()
}

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

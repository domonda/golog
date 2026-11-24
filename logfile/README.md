# logfile

Package logfile provides file-based log writers with automatic rotation capabilities for the [golog](https://github.com/domonda/golog) logging library.

## Features

- **Automatic File Rotation**: Rotates log files based on size thresholds
- **Thread-Safe**: Safe for concurrent use by multiple goroutines
- **Timestamp-Based Naming**: Rotated files are named with timestamps for easy identification
- **Configurable**: Customizable file paths, permissions, rotation sizes, and time formats
- **Write Callback**: Get notified after every write to report errors, retry failed writes, or watch disk space
- **Seamless Integration**: Works with any `io.Writer` compatible logging system

## Installation

```bash
go get github.com/domonda/golog/logfile
```

## Quick Start

### Basic Usage

```go
package main

import (
    "github.com/domonda/golog"
    "github.com/domonda/golog/logfile"
)

func main() {
    // Create a rotating writer that rotates at 10MB
    writer, err := logfile.NewRotatingWriter(
        "/var/log/myapp.log",                    // File path
        logfile.RotatingWriterDefaultTimeFormat, // Time format for rotated files
        0644,         // File permissions
        10*1024*1024, // Rotate at 10MB
    )
    if err != nil {
        panic(err)
    }
    defer writer.Close()

    // Use with golog
    config := golog.NewConfig(
        &golog.DefaultLevels,
        golog.AllLevelsActive,
        golog.NewJSONWriterConfig(writer, nil),
    )

    log := golog.NewLogger(config)

    log.Info("Application started").Log()
}
```

### Multiple Writers (Console + Rotating File)

```go
package main

import (
    "os"

    "github.com/domonda/golog"
    "github.com/domonda/golog/log"
    "github.com/domonda/golog/logfile"
)

func main() {
    // Create rotating file writer
    fileWriter, err := logfile.NewRotatingWriter(
        "/var/log/myapp.log",
        "",          // Use default time format
        0644,        // File permissions
        5*1024*1024, // Rotate at 5MB
    )
    if err != nil {
        panic(err)
    }
    defer fileWriter.Close()

    // Configure logger with both console and file output
    config := golog.NewConfig(
        &golog.DefaultLevels,
        golog.AllLevelsActive,
        // Colored text output to console
        golog.NewTextWriterConfig(
            os.Stdout,
            nil,
            log.NewStyledColorizer(),
        ),
        // JSON output to rotating file
        golog.NewJSONWriterConfig(fileWriter, nil),
    )

    logger := golog.NewLogger(config)

    // Logs will appear on console in color AND be written to file in JSON
    logger.Info("Server starting").
        Str("host", "localhost").
        Int("port", 8080).
        Log()
}
```

### Different Log Levels to Different Files

```go
package main

import (
    "github.com/domonda/golog"
    "github.com/domonda/golog/logfile"
)

func main() {
    // All logs file (rotates at 10MB)
    allLogsWriter, _ := logfile.NewRotatingWriter(
        "/var/log/myapp.log",
        "",
        0644,
        10*1024*1024,
    )
    defer allLogsWriter.Close()

    // Errors only file (rotates at 5MB)
    errorLogsWriter, _ := logfile.NewRotatingWriter(
        "/var/log/myapp-errors.log",
        "",
        0644,
        5*1024*1024,
    )
    defer errorLogsWriter.Close()

    // Configure with level filtering
    config := golog.NewConfig(
        &golog.DefaultLevels,
        golog.AllLevelsActive,
        // All logs
        golog.NewJSONWriterConfig(allLogsWriter, nil),
        // Errors and above only
        golog.NewJSONWriterConfig(
            errorLogsWriter,
            nil,
            golog.DefaultLevels.Error.FilterOutBelow(),
        ),
    )

    log := golog.NewLogger(config)

    log.Info("This goes to all logs").Log()
    log.Error("This goes to both all logs and error logs").Log()
}
```

## How Rotation Works

Before a write that would make the log file reach or exceed the configured size threshold:

1. The current file is closed
2. The file is renamed with a timestamp suffix (e.g., `myapp.log.2024-01-15_10:30:45`)
3. If a file with that name already exists, a numeric suffix is added (e.g., `myapp.log.2024-01-15_10:30:45.1`)
4. A new file is created at the original path
5. The write that triggered the rotation goes to the new file

Rotated files therefore stay just under the threshold.

### Failed rotations

If the rotation fails, for example because the directory of the log file is not writable,
no data is lost:

- The write that triggered the rotation still goes to the current log file,
  which keeps growing past the threshold until a rotation succeeds.
- `Write` returns a `*logfile.RotateError` that wraps the error of closing or renaming the file.
  If writing the data failed too, both errors are joined with `errors.Join`.
- The rotation is retried by the first write at least one minute after the failure,
  so a file that can't be renamed doesn't cost extra system calls and an error for every write.
- If opening the log file after a rotation fails, every following write tries to open it again,
  so the writer recovers once the cause is gone.

### Example File Rotation

```
# Before rotation
myapp.log (just under 10MB)

# After first rotation
myapp.log (the triggering message)
myapp.log.2024-01-15_10:30:45 (just under 10MB)

# After second rotation (same second)
myapp.log (the triggering message)
myapp.log.2024-01-15_10:30:45 (just under 10MB)
myapp.log.2024-01-15_10:30:45.1 (just under 10MB)
```

## Configuration Options

### NewRotatingWriter Parameters

```go
func NewRotatingWriter(
    filePath string,      // Path to log file
    timeFormat string,    // Time format for rotated files (empty = default)
    filePerm os.FileMode, // File permissions (e.g., 0644)
    rotateSize int64,     // Size in bytes (0 = no rotation)
) (*RotatingWriter, error)
```

`NewRotatingWriterWithCallback` takes the same parameters plus a `WriteCallback`
that is set before the first write (`nil` = no callback), see [Write Callback](#write-callback).

### Time Format

The `timeFormat` parameter uses Go's time layout format. Common formats:

```go
// Default format (recommended)
logfile.RotatingWriterDefaultTimeFormat // "2006-01-02_15:04:05"

// Date only
"2006-01-02" // myapp.log.2024-01-15

// Date and hour
"2006-01-02_15" // myapp.log.2024-01-15_10

// Custom format
"2006-01-02_15-04-05" // myapp.log.2024-01-15_10-30-45
```

### Rotation Size Guidelines

```go
// No rotation (file grows indefinitely)
0

// Common sizes
1024 * 1024           // 1 MB
5 * 1024 * 1024       // 5 MB
10 * 1024 * 1024      // 10 MB
100 * 1024 * 1024     // 100 MB
1024 * 1024 * 1024    // 1 GB
```

## Write Callback

`NewRotatingWriterWithCallback` creates a writer with a callback,
and `SetWriteCallback` sets or removes it later.
The callback is called after every `Write`, including writes that rotated the file first:

```go
type WriteCallback func(filePath string, data []byte, n int, err error)
```

- `filePath` is the path of the log file
- `data` is the slice that was passed to `Write`
- `n` is the number of bytes of `data` written to the file, as returned by `Write`
- `err` is the error returned by `Write`, or `nil` on success.
  On success `n` is `len(data)`.
  If rotating the file failed, `err` is or wraps a `*logfile.RotateError`,
  and `data` is still written to the current log file, so `n` is `len(data)` unless writing failed too.
  If opening or writing the file failed, `data[:n]` was written and `data[n:]` was not.

Use it to report a failing log file by other means, to retry or redirect the unwritten part
`data[n:]` of failed writes (so the part that reached the file is not written twice),
or to check the free disk space and send an alarm when the log volume fills up:

```go
writer.SetWriteCallback(func(filePath string, data []byte, n int, err error) {
    var rotateErr *logfile.RotateError
    if errors.As(err, &rotateErr) {
        // The data was written, but the log file grows past the rotation size
        fmt.Fprintf(os.Stderr, "can't rotate log file: %s\n", rotateErr)
    }
    if n < len(data) {
        // The log file can't be written, so report the unwritten part to stderr instead
        fmt.Fprintf(os.Stderr, "error writing log file %s: %s\n%s", filePath, err, data[n:])
        return
    }
    // Check the free disk space at most once per minute
    if now := time.Now(); now.Sub(lastDiskCheck) > time.Minute {
        lastDiskCheck = now
        checkFreeDiskSpace(filepath.Dir(filePath)) // e.g. with syscall.Statfs
    }
})
```

Rules for the callback:

- It is called after the writer's mutex has been released, so it can call methods
  of the writer like `Sync`, and it is called concurrently when `Write` is called concurrently
  (the example above needs a mutex for `lastDiskCheck` if you write concurrently).
- A panic in the callback is recovered and passed as error to `golog.ErrorHandler()`
  (stderr by default). It does not change the result of `Write`.
- It must not write to the same `RotatingWriter`, for example by logging to a golog logger
  that uses it: every write calls the callback again, which results in an endless recursion.
- It runs synchronously in the goroutine that called `Write`, so `Write` does not return
  before the callback returns. Other goroutines can write in the meantime.
  For slow work, like sending an alarm over the network, start a goroutine from the callback
  (see below) so it does not delay the logging code.
- It must not retain `data` after it returns, because the caller of `Write` may reuse the slice.
  Copy `data` with `bytes.Clone` before passing it to a goroutine or keeping it in any other way.

### Slow work in a goroutine

Starting a goroutine per call is not safe: when the disk is full every write fails,
so every log call would start another goroutine.
Instead, hand the failed writes to one worker goroutine through a buffered channel:

```go
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

handleFailedWrite := func(w failedWrite) {
    defer func() {
        if r := recover(); r != nil {
            fmt.Fprintln(os.Stderr, "panic while handling failed log write:", r)
        }
    }()
    sendAlarm(w.filePath, w.data, w.err) // Slow work, e.g. an HTTP request
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
        dropped.Add(1) // Never block the logging code
    }
})

// Shutdown: remove the callback, then stop the worker
// after it has handled the buffered failed writes
writer.SetWriteCallback(nil)
close(stopWorker)
<-workerDone
```

Why it is written this way:

- **One worker goroutine** limits the goroutines and memory used when every write fails,
  and handles the failed writes in the order the callbacks were called.
  With concurrent writes that order can differ from the order of the writes,
  because the callback runs after the writer's mutex has been released.
- **`bytes.Clone(data[n:])`** because the caller of `Write` reuses the slice after the callback returns,
  and only the unwritten part `data[n:]` needs handling.
  Copying allocates, so the callback copies only after a failed write.
- **Non-blocking send**: when the buffer is full, the failed write is counted as dropped
  instead of making the logging code wait.
- **`recover` in the worker**, because the `RotatingWriter` only recovers panics in the callback
  itself, not in goroutines started by it.
- **The channel is never closed.** A `Write` running concurrently with the shutdown can still call
  the callback, even after `SetWriteCallback(nil)` returned, and sending on a closed channel panics.
  The worker is stopped with `stopWorker` instead. A failed write sent by such a `Write`
  after the worker stopped stays unhandled in the buffer.
- **The worker does not log to the same `RotatingWriter`**, which would fail again
  and feed the worker its own errors.

The runnable version is `ExampleRotatingWriter_SetWriteCallback` in [example_test.go](example_test.go).

Pass `nil` to `SetWriteCallback` to remove the callback.

## Thread Safety

`RotatingWriter` is fully thread-safe. All operations (Write, Sync, Close) are protected by an internal mutex, making it safe to use from multiple goroutines simultaneously.

```go
// Safe to use from multiple goroutines
go func() {
    log.Info("From goroutine 1").Log()
}()

go func() {
    log.Info("From goroutine 2").Log()
}()
```

## Methods

### Write([]byte) (int, error)

Writes data to the log file. Automatically rotates if size threshold is exceeded.

### SetWriteCallback(WriteCallback)

Sets a callback that is called after every write with the log file path,
the data passed to `Write`, the number of bytes written and the write error. See [Write Callback](#write-callback).

### Sync() error

Flushes buffered data to disk. Useful for ensuring logs are persisted before shutdown.

```go
defer func() {
    writer.Sync()  // Ensure all data is written
    // Do something else...
    writer.Close() // Note that Close also writes all buffered data to disk
}()
```

## Integration with Standard Library

Since `RotatingWriter` implements `io.Writer`, it can be used with any logging library that accepts an `io.Writer`:

```go
// Standard library log
import "log"

writer, _ := logfile.NewRotatingWriter("/var/log/app.log", "", 0644, 10*1024*1024)
defer writer.Close()

log.SetOutput(writer)
log.Println("This will be written to a rotating log file")
```

## License

This package is part of the [golog](https://github.com/domonda/golog) project and is licensed under the MIT License.

## See Also

- [golog documentation](https://pkg.go.dev/github.com/domonda/golog)
- [Main golog README](https://github.com/domonda/golog)

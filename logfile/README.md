# logfile

Package logfile provides file-based log writers with automatic rotation capabilities for the [golog](https://github.com/domonda/golog) logging library.

## Features

- **Automatic File Rotation**: Rotates log files based on size thresholds
- **Thread-Safe**: Safe for concurrent use by multiple goroutines
- **Timestamp-Based Naming**: Rotated files are named with timestamps for easy identification
- **Configurable**: Customizable file paths, permissions, rotation sizes, and time formats
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
    "os"

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
    log.Error("An error occurred").Err(err).Log()
}
```

### Multiple Writers (Console + Rotating File)

```go
package main

import (
    "os"

    "github.com/domonda/golog"
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
            golog.NewStyledColorizer(),
        ),
        // JSON output to rotating file
        golog.NewJSONWriterConfig(fileWriter, nil),
    )

    log := golog.NewLogger(config)

    // Logs will appear on console in color AND be written to file in JSON
    log.Info("Server starting").
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
            golog.LevelFilterFrom(golog.DefaultLevels.Error),
        ),
    )

    log := golog.NewLogger(config)

    log.Info("This goes to all logs").Log()
    log.Error("This goes to both all logs and error logs").Log()
}
```

## How Rotation Works

When a log file reaches the configured size threshold:

1. The current file is closed
2. The file is renamed with a timestamp suffix (e.g., `myapp.log.2024-01-15_10:30:45`)
3. If a file with that name already exists, a numeric suffix is added (e.g., `myapp.log.2024-01-15_10:30:45.1`)
4. A new file is created at the original path
5. Logging continues to the new file

### Example File Rotation

```
# Before rotation
myapp.log (10MB)

# After first rotation
myapp.log (0 bytes, new file)
myapp.log.2024-01-15_10:30:45 (10MB)

# After second rotation (same second)
myapp.log (0 bytes, new file)
myapp.log.2024-01-15_10:30:45 (10MB)
myapp.log.2024-01-15_10:30:45.1 (10MB)
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

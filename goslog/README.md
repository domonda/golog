# goslog

Package goslog provides a bridge between Go's standard `log/slog` package and the [golog](https://github.com/domonda/golog) structured logging library.

## Overview

This package implements a `slog.Handler` that routes log records from slog to golog, enabling applications to:
- Use the standard library's slog API
- Benefit from golog's flexible output formatting
- Leverage golog's multiple writers
- Take advantage of golog's performance optimizations

The handler passes the official `slogtest.TestHandler` compliance suite, ensuring full compatibility with slog's expected behavior.

## Installation

```bash
go get github.com/domonda/golog/goslog
```

## Quick Start

### Basic Usage

```go
package main

import (
    "log/slog"
    "os"

    "github.com/domonda/golog"
    "github.com/domonda/golog/goslog"
)

func main() {
    // Create a golog logger with JSON output
    gologLogger := golog.NewLogger(
        golog.NewConfig(
            &golog.DefaultLevels,
            golog.AllLevelsActive,
            golog.NewJSONWriterConfig(os.Stdout, nil),
        ),
    )

    // Create a slog handler that uses the golog logger
    handler := goslog.Handler(gologLogger, goslog.ConvertDefaultLevels)

    // Use it with slog
    logger := slog.New(handler)

    logger.Info("Application started", "version", "1.0.0")
    logger.Error("Failed to connect", "host", "localhost", "port", 5432)
}
```

### With Multiple Output Formats

```go
import (
    "log/slog"
    "os"

    "github.com/domonda/golog"
    "github.com/domonda/golog/goslog"
    "github.com/domonda/golog/logfile"
)

// Create rotating file writer
fileWriter, _ := logfile.NewRotatingWriter(
    "/var/log/app.log",
    "",
    0644,
    10*1024*1024, // 10MB
)
defer fileWriter.Close()

// Create golog logger with multiple outputs
gologLogger := golog.NewLogger(
    golog.NewConfig(
        &golog.DefaultLevels,
        golog.AllLevelsActive,
        // Console: colored text
        golog.NewTextWriterConfig(os.Stdout, nil, golog.NewStyledColorizer()),
        // File: JSON with rotation
        golog.NewJSONWriterConfig(fileWriter, nil),
    ),
)

// Use with slog
handler := goslog.Handler(gologLogger, goslog.ConvertDefaultLevels)
logger := slog.New(handler)

logger.Info("Logs to both console and file")
```

### Set as Default Logger

```go
package main

import (
    "log/slog"

    "github.com/domonda/golog"
    "github.com/domonda/golog/goslog"
)

func main() {
    // Create golog logger
    gologLogger := golog.NewLogger(/* config */)

    // Create slog handler
    handler := goslog.Handler(gologLogger, goslog.ConvertDefaultLevels)

    // Set as default slog logger
    slog.SetDefault(slog.New(handler))

    // Now all slog calls use golog
    slog.Info("Using default logger")
}
```

## Features

### Level Conversion

The `ConvertDefaultLevels` function provides standard mapping between slog and golog levels:

| slog Level | golog Level |
|------------|-------------|
| `LevelDebug` and below | `DefaultLevels.Debug` and below |
| `LevelInfo` | `DefaultLevels.Info` |
| `LevelWarn` | `DefaultLevels.Warn` |
| `LevelError` and above | `DefaultLevels.Error` and above |

The mapping preserves relative level differences, so custom slog levels map correctly to golog's level system.

### Custom Level Conversion

You can provide your own level conversion function:

```go
// Custom converter that maps everything to golog INFO
customConverter := func(level slog.Level) golog.Level {
    return golog.DefaultLevels.Info
}

handler := goslog.Handler(gologLogger, customConverter)
```

### Structured Attributes

All slog attribute types are fully supported:

```go
logger.Info("User action",
    slog.String("username", "john"),
    slog.Int("user_id", 42),
    slog.Bool("authenticated", true),
    slog.Float64("response_time", 0.123),
    slog.Time("timestamp", time.Now()),
    slog.Duration("elapsed", 150*time.Millisecond),
)
```

### Attribute Groups

Groups create nested attribute structures:

```go
logger.Info("HTTP request",
    slog.Group("request",
        slog.String("method", "GET"),
        slog.String("path", "/api/users"),
        slog.Int("status", 200),
    ),
    slog.Group("client",
        slog.String("ip", "192.168.1.1"),
        slog.String("user_agent", "Mozilla/5.0"),
    ),
)
```

**JSON Output:**
```json
{
  "level": "INFO",
  "msg": "HTTP request",
  "request": {
    "method": "GET",
    "path": "/api/users",
    "status": 200
  },
  "client": {
    "ip": "192.168.1.1",
    "user_agent": "Mozilla/5.0"
  }
}
```

### Child Loggers with Attributes

Use `WithAttrs` to create child loggers with common attributes:

```go
// Base logger
baseLogger := slog.New(handler)

// Request-scoped logger with common attributes
requestLogger := baseLogger.With(
    slog.String("request_id", "abc-123"),
    slog.String("user_id", "user-456"),
)

// All logs from requestLogger include request_id and user_id
requestLogger.Info("Processing request")
requestLogger.Error("Validation failed")
```

### Child Loggers with Groups

Use `WithGroup` to create loggers that prefix all attributes:

```go
baseLogger := slog.New(handler)

// All attributes will be under "service" group
serviceLogger := baseLogger.WithGroup("service")

serviceLogger.Info("Starting", "name", "api", "version", "2.0")
// Output: {"level":"INFO","msg":"Starting","service":{"name":"api","version":"2.0"}}
```

### Lazy Evaluation with LogValuer

Expensive computations can be deferred:

```go
type User struct {
    ID   int
    Name string
}

func (u User) LogValue() slog.Value {
    return slog.GroupValue(
        slog.Int("id", u.ID),
        slog.String("name", u.Name),
    )
}

user := User{ID: 42, Name: "John"}
logger.Info("User logged in", "user", user) // LogValue only called if level is active
```

## Advanced Usage

### Multiple Handlers

Combine handlers for different purposes:

```go
// Development handler: pretty text to console
devHandler := goslog.Handler(
    golog.NewLogger(
        golog.NewConfig(
            &golog.DefaultLevels,
            golog.AllLevelsActive,
            golog.NewTextWriterConfig(os.Stdout, nil, golog.NewStyledColorizer()),
        ),
    ),
    goslog.ConvertDefaultLevels,
)

// Production handler: JSON to file
prodHandler := goslog.Handler(
    golog.NewLogger(
        golog.NewConfig(
            &golog.DefaultLevels,
            golog.AllLevelsActive,
            golog.NewJSONWriterConfig(logFile, nil),
        ),
    ),
    goslog.ConvertDefaultLevels,
)

// Choose based on environment
var handler slog.Handler
if os.Getenv("ENV") == "production" {
    handler = prodHandler
} else {
    handler = devHandler
}

logger := slog.New(handler)
```

### Filtering by Level

Use golog's level filtering to control output:

```go
// Only log WARN and above to file
gologLogger := golog.NewLogger(
    golog.NewConfig(
        &golog.DefaultLevels,
        golog.AllLevelsActive,
        // Console: all levels
        golog.NewTextWriterConfig(os.Stdout, nil, nil),
        // File: warnings and errors only
        golog.NewJSONWriterConfig(
            fileWriter,
            nil,
            golog.LevelFilterFrom(golog.DefaultLevels.Warn),
        ),
    ),
)

handler := goslog.Handler(gologLogger, goslog.ConvertDefaultLevels)
logger := slog.New(handler)

logger.Info("This goes to console only")
logger.Warn("This goes to both console and file")
```

### Context Integration

Use slog's context-aware methods:

```go
import "context"

ctx := context.Background()

logger.InfoContext(ctx, "Operation started", "operation", "backup")
logger.ErrorContext(ctx, "Operation failed", "error", err)
```

## Migration from slog

If you're already using slog, migrating to use golog is straightforward:

### Before (pure slog)
```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
logger.Info("Hello, World!")
```

### After (slog → golog)
```go
gologLogger := golog.NewLogger(
    golog.NewConfig(
        &golog.DefaultLevels,
        golog.AllLevelsActive,
        golog.NewJSONWriterConfig(os.Stdout, nil),
    ),
)
logger := slog.New(goslog.Handler(gologLogger, goslog.ConvertDefaultLevels))
logger.Info("Hello, World!")
```

All your existing slog calls work without modification!

## Benefits Over Pure slog

1. **Multiple Output Formats**: Log to console, files, and custom writers simultaneously
2. **Rotating Log Files**: Built-in support for automatic log rotation
3. **Colorized Output**: Beautiful colored console output for development
4. **Performance**: golog's zero-allocation design and memory pooling
5. **Flexible Configuration**: Advanced filtering, formatting, and writer options
6. **Familiar API**: Keep using slog's standard API

## Compatibility

- ✅ Passes `slogtest.TestHandler` compliance suite
- ✅ Supports all slog attribute types
- ✅ Full support for groups and child loggers
- ✅ Compatible with slog.LogValuer interface
- ✅ Works with context-aware logging methods

## Performance Considerations

The handler introduces minimal overhead by:
- Converting slog levels to golog levels (simple arithmetic)
- Routing attribute writes through golog's Message API
- Handling group prefixes with dot notation

For maximum performance in high-throughput scenarios, consider using golog directly instead of through the slog adapter.

## Testing

The handler is tested with Go's official slog test suite:

```go
import "testing/slogtest"

func TestHandler(t *testing.T) {
    var rec recorder
    config := golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, &rec)

    handler := goslog.Handler(golog.NewLogger(config), goslog.ConvertDefaultLevels)

    err := slogtest.TestHandler(handler, func() []map[string]any {
        return rec.Result
    })
    if err != nil {
        t.Fatal(err)
    }
}
```

## See Also

- [golog documentation](https://pkg.go.dev/github.com/domonda/golog)
- [Main golog README](https://github.com/domonda/golog)
- [Go slog documentation](https://pkg.go.dev/log/slog)
- [logfile package](../logfile) - Rotating log file support

## License

This package is part of the [golog](https://github.com/domonda/golog) project and is licensed under the MIT License.

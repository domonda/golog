# golog

Fast and flexible structured logging library for Go inspired by zerolog

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/domonda/golog)
[![Go Reference](https://pkg.go.dev/badge/github.com/domonda/golog.svg)](https://pkg.go.dev/github.com/domonda/golog)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Features

- **High Performance**: Zero-allocation logging with memory pooling
- **Structured Logging**: Type-safe field methods for all Go primitives
- **Multiple Output Formats**: JSON and human-readable text output
- **Configurable Log Levels**: TRACE, DEBUG, INFO, WARN, ERROR, FATAL
- **Context Support**: Log attributes can be stored in and retrieved from context
- **Colorized Output**: Beautiful colored console output with customizable colorizers
- **Flexible Configuration**: Multiple writers, filters, and level configurations
- **Rotating Log Files**: Automatic file rotation based on size thresholds
- **slog Integration**: Use as a backend for Go's standard log/slog package
- **HTTP Middleware**: Built-in HTTP request/response logging
- **UUID Support**: Native UUID logging support
- **Call Stack Tracing**: Capture and log call stacks for debugging
- **Memory Safety**: Nil-safe logger implementation prevents panics
- **Sub-loggers**: Create child loggers with inherited attributes

## Installation

```bash
go get github.com/domonda/golog
```

## Quick Start

### Basic Usage

```go
package main

import (
    "os"
    
    "github.com/domonda/golog"
)

func main() {
    // Create a basic text logger
    config := golog.NewConfig(
        &golog.DefaultLevels,
        golog.AllLevelsActive,
        golog.NewTextWriterConfig(os.Stdout, nil, nil),
    )
    
    log := golog.NewLogger(config)
    
    // Simple logging
    log.Info("Hello, World!").Log()
    log.Error("Something went wrong").Err(errors.New("example error")).Log()
}
```

### JSON Output

```go
// Create a JSON logger
config := golog.NewConfig(
    &golog.DefaultLevels,
    golog.AllLevelsActive,
    golog.NewJSONWriterConfig(os.Stdout, nil),
)

log := golog.NewLogger(config)

log.Info("User login").
    Str("username", "john_doe").
    Str("ip", "192.168.1.1").
    Duration("login_time", time.Since(start)).
    Log()
```

Output:
```json
{"timestamp":"2024-01-15T10:30:45Z","level":"INFO","message":"User login","username":"john_doe","ip":"192.168.1.1","login_time":"150ms"}
```

### Structured Logging with All Data Types

```go
log.Info("Processing request").
    Str("method", "POST").
    Str("path", "/api/users").
    Int("user_id", 12345).
    Bool("authenticated", true).
    Float("response_time", 0.145).
    UUID("request_id", requestID).
    Time("started_at", time.Now()).
    Strs("tags", []string{"api", "user", "create"}).
    JSON("metadata", jsonBytes).
    Log()
```

## Log Levels

golog supports six standard log levels:

- **TRACE** (-20): Most verbose, for tracing execution flow
- **DEBUG** (-10): Debug information for development
- **INFO** (0): General information messages
- **WARN** (10): Warning messages for potentially harmful situations
- **ERROR** (20): Error conditions that don't require immediate attention
- **FATAL** (30): Critical errors that may cause application termination

### Level-specific Methods

```go
log.Trace("Entering function").Str("function", "processData").Log()
log.Debug("Variable state").Int("counter", counter).Log()
log.Info("Operation completed").Log()
log.Warn("Deprecated API used").Str("api", "/old/endpoint").Log()
log.Error("Failed to connect").Err(err).Log()
log.Fatal("Critical system failure").Log() // Calls panic after logging
```

## Sub-loggers and Context

### Creating Sub-loggers

```go
// Create a sub-logger with common attributes
subLog := log.With().
    Str("service", "user-management").
    UUID("request_id", requestID).
    SubLogger()

// All logs from subLog will include the above attributes
subLog.Info("User created").Str("username", "john").Log()
subLog.Error("User validation failed").Err(validationErr).Log()
```

### Context Integration

```go
// Add attributes to context
ctx = golog.ContextWithAttribs(ctx, 
    golog.Str("correlation_id", correlationID),
    golog.Str("user_id", userID),
)

// Create logger from context
ctxLogger := log.WithCtx(ctx)
ctxLogger.Info("Operation started").Log() // Includes context attributes
```

## Multiple Writers and Filtering

```go
// Log to multiple outputs with different formats
config := golog.NewConfig(
    &golog.DefaultLevels,
    golog.AllLevelsActive,
    golog.NewTextWriterConfig(os.Stdout, nil, golog.NewStyledColorizer()),
    golog.NewJSONWriterConfig(logFile, nil, golog.LevelFilterFrom(golog.DefaultLevels.Warn)),
)

log := golog.NewLogger(config)

// This will appear in colored text on stdout and as JSON in the file (if WARN+)
log.Error("Database connection failed").Err(err).Log()
```

## Terminal Detection

golog can automatically switch between human-readable text and machine-readable JSON output based on whether the process is attached to a terminal (TTY):

```go
config := golog.NewConfig(
    &golog.DefaultLevels,
    golog.AllLevelsActive,
    golog.DecideWriterConfigForTerminal(
        golog.NewTextWriterConfig(os.Stdout, format, golog.NewStyledColorizer()), // Used when running in terminal
        golog.NewJSONWriterConfig(os.Stdout, format),                             // Used when output is piped/redirected
    ),
)

log := golog.NewLogger(config)
```

- **Terminal (TTY)**: Outputs colored, human-readable text format
- **Non-terminal**: Outputs machine-readable JSON format (ideal for log aggregation systems)

You can also check the terminal status directly:

```go
if golog.IsTerminal() {
    // Running in a terminal
}
```

This feature is useful for:
- **Development**: Human-readable logs in the terminal
- **Production**: Machine-parseable JSON logs for log aggregation systems
- **CI/CD**: Automatic format switching based on the environment
- **Containerized Applications**: Proper format when logs are piped to files or log collectors

## Ready-to-Use Logger (`log` subpackage)

For quick setup, the `log` subpackage provides a pre-configured logger with sensible defaults:

```go
import "github.com/domonda/golog/log"

func main() {
    log.Info("Application started").Log()
    log.Error("Something went wrong").Err(err).Log()
    log.Debug("Debug information").Str("key", "value").Log()
}
```

### Default Configuration

The `log` package is configured with:

- **Log Levels**: Uses `golog.DefaultLevels` (TRACE, DEBUG, INFO, WARN, ERROR, FATAL)
- **Level Filter**: Filters out levels below `LOG_LEVEL` environment variable (defaults to DEBUG if not set)
- **Output**: Writes to stdout with automatic format selection:
  - **Terminal**: Colorized text format for human readability
  - **Non-terminal**: JSON format for log aggregation systems

### Customizing the Default Logger

The `log.Config` variable can be modified at runtime:

```go
import "github.com/domonda/golog/log"

func init() {
    // Change the minimum log level
    log.Config = golog.NewConfig(
        log.Levels,
        log.Levels.Info.FilterOutBelow(), // Only INFO and above
        golog.DecideWriterConfigForTerminal(
            golog.NewTextWriterConfig(os.Stdout, &log.Format, &log.Colorizer),
            golog.NewJSONWriterConfig(os.Stdout, &log.Format),
        ),
    )
}
```

The logger uses a `DerivedConfig` that references `log.Config`, so changes to `log.Config` take effect immediately without recreating the logger.

### Available Package-Level Functions

The `log` package exposes convenience functions for all log levels:

```go
log.Trace("message").Log()
log.Debug("message").Log()
log.Info("message").Log()
log.Warn("message").Log()
log.Error("message").Log()
log.Fatal("message").Log()

// With context
log.InfoCtx(ctx, "message").Log()

// Formatted messages
log.Infof("User %s logged in", username).Log()

// Create sub-loggers
subLog := log.With().Str("component", "auth").SubLogger()
```

## Rotating Log Files

Automatic file rotation based on size thresholds using the `logfile` subpackage:

```go
import (
    "github.com/domonda/golog"
    "github.com/domonda/golog/logfile"
)

// Create a rotating writer that rotates at 10MB
writer, err := logfile.NewRotatingWriter(
    "/var/log/myapp.log",                    // File path
    logfile.RotatingWriterDefaultTimeFormat, // Time format for rotated files
    0644,         // File permissions
    10*1024*1024, // Rotate at 10MB
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

log := golog.NewLogger(config)

log.Info("Application started").Log()
```

When the log file reaches 10MB:
- The current file is renamed with a timestamp (e.g., `myapp.log.2024-01-15_10:30:45`)
- A new file is created at the original path
- Logging continues seamlessly to the new file

### Multiple Writers with Rotation

```go
// Console output with colors + rotating JSON file
fileWriter, _ := logfile.NewRotatingWriter("/var/log/app.log", "", 0644, 50*1024*1024)
defer fileWriter.Close()

config := golog.NewConfig(
    &golog.DefaultLevels,
    golog.AllLevelsActive,
    golog.NewTextWriterConfig(os.Stdout, nil, golog.NewStyledColorizer()),
    golog.NewJSONWriterConfig(fileWriter, nil),
)

log := golog.NewLogger(config)
```

See the [logfile package documentation](logfile/README.md) for more details.

## Standard Library Integration (slog)

Use golog as a backend for Go's standard `log/slog` package via the `goslog` adapter:

```go
import (
    "log/slog"

    "github.com/domonda/golog"
    "github.com/domonda/golog/goslog"
)

// Create golog logger
gologLogger := golog.NewLogger(
    golog.NewConfig(
        &golog.DefaultLevels,
        golog.AllLevelsActive,
        golog.NewJSONWriterConfig(os.Stdout, nil),
    ),
)

// Create slog handler that uses golog
handler := goslog.Handler(gologLogger, goslog.ConvertDefaultLevels)

// Use with slog
logger := slog.New(handler)
logger.Info("Hello from slog", "key", "value")
```

### Benefits of slog Integration

- **Standard API**: Use Go's standard library slog API
- **Existing Code**: Works with existing code that uses slog
- **golog Features**: Get all golog benefits (multiple writers, rotation, colors)
- **Full Compatibility**: Passes slogtest compliance suite

See the [goslog package documentation](goslog/README.md) for more details.

## HTTP Middleware

```go
func loggingMiddleware(log *golog.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            
            // Log request
            log.Info("HTTP request").
                Request(r).
                Log()
            
            next.ServeHTTP(w, r)
            
            // Log response
            log.Info("HTTP response").
                Str("method", r.Method).
                Str("path", r.URL.Path).
                Duration("duration", time.Since(start)).
                Log()
        })
    }
}
```

## Advanced Features

### Custom Colorizers

```go
colorizer := golog.NewStyledColorizer()
// or implement your own Colorizer interface

config := golog.NewConfig(
    &golog.DefaultLevels,
    golog.AllLevelsActive,
    golog.NewTextWriterConfig(os.Stdout, nil, colorizer),
)
```

### Call Stack Logging

```go
log.Error("Panic recovered").
    CallStack("stack").
    Any("panic_value", panicValue).
    Log()
```

### Redacting Sensitive Data

Use the `golog:"redact"` struct tag to automatically redact sensitive fields when logging structs:

```go
type User struct {
    ID       int    `json:"id"`
    Username string `json:"username"`
    Password string `json:"password" golog:"redact"`
    APIKey   string `json:"api_key"  golog:"redact"`
}

user := User{
    ID:       123,
    Username: "john_doe",
    Password: "secret123",
    APIKey:   "sk-abc123",
}

log.Info("User logged in").StructFields(user).Log()
// Output: ... id=123 username="john_doe" password="***REDACTED***" api_key="***REDACTED***"
```

The `golog:"redact"` tag works with both `StructFields()` and `TaggedStructFields()` methods.

### Custom Levels

```go
customLevels := &golog.Levels{
    Trace: -20,
    Debug: -10,
    Info:  0,
    Warn:  10,
    Error: 20,
    Fatal: 30,
    Names: map[golog.Level]string{
        -20: "TRACE",
        -10: "DEBUG",
        0:   "INFO",
        10:  "WARN",
        20:  "ERROR",
        30:  "FATAL",
    },
}
```

### Level Filtering

```go
// Only log WARN and above
filter := golog.LevelFilterFrom(golog.DefaultLevels.Warn)

// Multiple filters
filter := golog.JoinLevelFilters(
    golog.LevelFilterFrom(golog.DefaultLevels.Debug),
    golog.NewLevelFilterFunction(func(ctx context.Context, level golog.Level) bool {
        // Custom filter logic
        return shouldLog(ctx, level)
    }),
)
```

## Performance

golog is designed for high performance:

- Zero-allocation logging in most cases
- Object pooling for message and writer instances
- Efficient JSON encoding
- Lazy evaluation of expensive operations
- Minimal overhead for inactive log levels

## API Reference

### Core Types

- **Logger**: Main logging interface
- **Message**: Fluent message builder
- **Config**: Logger configuration
- **WriterConfig**: Output writer configuration
- **Level**: Log level type
- **LevelFilter**: Level filtering interface

### Writer Types

- **JSONWriter**: Structured JSON output
- **TextWriter**: Human-readable text output
- **CallbackWriter**: Custom callback-based writer
- **MultiWriter**: Multiple writer composition
- **NopWriter**: No-operation writer for testing

For complete API documentation, see [pkg.go.dev](https://pkg.go.dev/github.com/domonda/golog).

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.


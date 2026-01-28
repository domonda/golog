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

### Benchmarks

Run internal benchmarks:
```bash
go test -bench=. -benchmem
```

For comparative benchmarks against other popular Go logging libraries (zerolog, zap, slog, logrus),
the benchmarks are in a separate module to avoid adding those dependencies to the main golog module:
```bash
go test ./benchmarks -bench=. -benchmem -benchtime=1s
```

See [benchmarks/README.md](benchmarks/README.md) for detailed comparative analysis and performance insights.

## Comparison with Other Logging Libraries

golog is designed to strike a balance between performance and flexibility. While libraries like zerolog and zap prioritize raw speed, golog provides a richer feature set that makes it more adaptable to complex logging requirements.

### Performance vs Flexibility Tradeoffs

| Feature                                  | zerolog          | zap              | golog                 |
|------------------------------------------|------------------|------------------|-----------------------|
| **Multi-writer support**                 | Single output    | Limited          | Native, unlimited     |
| **Duplicate key prevention**             | No               | No               | Yes                   |
| **Context attribute integration**        | Manual           | Manual           | Automatic             |
| **Sub-logger with inherited attributes** | Basic            | Basic            | Full support with attrib recording |
| **Zero allocations (simple message)**    | Yes              | Yes              | Yes                   |
| **Zero allocations (with fields)**       | Yes              | No (1 alloc)     | Yes                   |
| **slog compatibility**                   | Separate adapter | Separate adapter | Native goslog package |

### Architectural Differences

**zerolog: Extreme Minimalism**
- Optimized for a single use case: fast JSON logging to a single output
- Minimal abstraction layers result in the fastest raw performance
- Disabled log levels have near-zero overhead (~4 ns/op)
- Trade-off: Limited flexibility for complex logging scenarios

**zap: Performance + Type Safety**
- Typed `Field` structs provide compile-time safety
- Separate "Sugar" logger offers convenience at the cost of performance
- Trade-off: Allocates a slice for variadic field arguments (~1 alloc per log call with fields)

**golog: Flexibility + Features**
- **Native multi-writer architecture**: Log to console, files, and external services simultaneously with different formats and filters per destination
- **Automatic context integration**: Attributes added to `context.Context` are automatically included in log messages without manual plumbing
- **Sub-logger attribute recording**: The `With().SubLogger()` pattern creates child loggers that efficiently inherit and extend parent attributes
- **Duplicate key prevention**: Prevents accidental duplicate keys in log output, ensuring clean structured data
- **Zero allocations for standard logging**: Despite the richer feature set, golog achieves zero allocations for JSON logging with fields
- **Nil-safe design**: A nil logger is safe to use and won't panic, simplifying error handling

### When to Choose golog

golog is the right choice when you need:

- **Multiple output destinations**: Log to stdout with colors for development and JSON files for production simultaneously
- **Request-scoped logging**: Automatically propagate correlation IDs, user IDs, and other context through your application
- **Sub-loggers with inherited context**: Create child loggers for specific components that include parent attributes
- **Clean structured data**: Prevent duplicate keys from appearing in your logs
- **slog compatibility**: Use golog as a backend for Go's standard library logging interface
- **Rotating log files**: Built-in support for size-based log rotation

### When to Choose Alternatives

- **zerolog**: When raw JSON logging speed is the only priority and you don't need multi-writer support or context integration
- **zap**: When you prefer a variadic field API with compile-time type checking and can accept one allocation per log call
- **slog**: When you want zero external dependencies and good-enough performance from the standard library

### Real-World Performance

For most applications, the performance difference between logging libraries is negligible. At 332 ns/op for a simple message, golog can handle over 3 million log messages per second on a single core. The additional features golog provides—multi-writer support, context integration, and duplicate key prevention—often save more development time than the nanoseconds saved by faster alternatives.

The performance gap becomes meaningful only in extreme high-throughput scenarios (100K+ logs/second sustained), where zerolog's 55 ns/op provides measurable benefits. For typical applications, golog's flexibility and rich feature set make it a more productive choice.

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


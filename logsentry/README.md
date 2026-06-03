# logsentry

A Sentry integration package for [golog](https://github.com/domonda/golog), providing seamless structured logging to Sentry error monitoring platform.

## Overview

The `logsentry` package implements the `golog.Writer` and `golog.WriterConfig` interfaces to bridge golog's structured logging capabilities with Sentry's error tracking and monitoring system. It automatically maps golog log levels to Sentry event levels and captures structured log data as Sentry events.

## Features

- **Automatic Level Mapping**: Maps golog log levels to appropriate Sentry event levels
- **Structured Data Capture**: Converts golog key-value pairs into a Sentry event `log` context (typed: time values are formatted via the golog Format, numbers/bools stay native, JSON stays structured)
- **Stack Trace Filtering**: Automatically filters out golog internal frames from stack traces
- **Memory Pooling**: Uses object pooling for efficient memory management
- **Context-Aware Logging**: Supports disabling Sentry logging via context
- **Configurable Formatting**: Supports both message-only and key-value message formats

## Installation

```bash
go get github.com/domonda/golog/logsentry
```

## Dependencies

- `github.com/domonda/golog` - Core logging library
- `github.com/getsentry/sentry-go` - Sentry Go SDK

## Quick Start

### Basic Setup

```go
package main

import (
    "context"
    "os"
    "time"

    "github.com/getsentry/sentry-go"
    "github.com/domonda/golog"
    "github.com/domonda/golog/logsentry"
)

func main() {
    // Initialize Sentry
    err := sentry.Init(sentry.ClientOptions{
        Dsn: os.Getenv("SENTRY_DSN"),
        // Enable stack traces for better debugging
        AttachStacktrace: true,
        // Set sample rate to control event volume
        SampleRate: 1.0,
    })
    if err != nil {
        panic("sentry.Init: " + err.Error())
    }
    defer sentry.Flush(2 * time.Second)

    // Create logsentry writer config
    sentryWriterConfig := logsentry.NewWriterConfig(
        sentry.CurrentHub(),           // Use current Sentry hub
        golog.NewDefaultFormat(),      // Use default golog format
        golog.AllLevelsActive,          // Log all levels
        false,                         // Don't include values in message text
        map[string]any{                // Extra data for all events
            "service": "my-app",
            "version": "1.0.0",
        },
    )

    // Create golog config with Sentry writer
    config := golog.NewConfig(
        &golog.DefaultLevels,
        golog.AllLevelsActive,
        sentryWriterConfig,
    )

    // Create logger
    logger := golog.NewLogger(config)

    // Log some events
    logger.Info("Application started").
        Str("port", "8080").
        Str("environment", "production").
        Log()

    logger.Error("Database connection failed").
        Err(errors.New("connection timeout")).
        Str("host", "db.example.com").
        Int("port", 5432).
        Log()
}
```

### Multiple Writers (Console + Sentry)

```go
// Create both console and Sentry writers
consoleWriter := golog.NewTextWriterConfig(os.Stdout, nil, nil)
sentryWriter := logsentry.NewWriterConfig(
    sentry.CurrentHub(),
    golog.NewDefaultFormat(),
    golog.DefaultLevels.Error.FilterOutBelow(), // Only send ERROR and FATAL to Sentry
    false,
    map[string]any{"service": "my-app"},
)

// Combine writers in golog config
config := golog.NewConfig(
    &golog.DefaultLevels,
    golog.AllLevelsActive,
    consoleWriter,
    sentryWriter,
)

logger := golog.NewLogger(config)
```

## Log Level Mapping

The package automatically maps golog log levels to Sentry event levels:

| golog Level | Sentry Level | Description |
|-------------|--------------|-------------|
| `TRACE` (-20) | `DEBUG` | Most verbose tracing information |
| `DEBUG` (-10) | `DEBUG` | Debug information for development |
| `INFO` (0) | `INFO` | General information messages |
| `WARN` (10) | `WARNING` | Warning messages for potentially harmful situations |
| `ERROR` (20) | `ERROR` | Error conditions that don't require immediate attention |
| `FATAL` (30) | `FATAL` | Critical errors that may cause application termination |
| Unknown/Other | `ERROR` | Fallback for unmapped levels (uses `UnknownLevel` variable) |

## Structured Data Handling

### Key-Value Pairs

All golog key-value pairs are automatically captured in a Sentry event `log` context
(sentry-go v0.46.0 removed `Event.Extra`, so the data is attached as a named context):

```go
logger.Error("User authentication failed").
    Str("username", "john_doe").
    Str("ip_address", "192.168.1.100").
    Int("attempt_count", 3).
    Bool("account_locked", false).
    Float("response_time", 0.145).
    Log()
```

This creates a Sentry event with:
- **Message**: "User authentication failed"
- **Level**: ERROR
- **Context** (`log`):
  - `username`: "john_doe"
  - `ip_address`: "192.168.1.100"
  - `attempt_count`: 3
  - `account_locked`: false
  - `response_time`: 0.145

> A value logged under the reserved key `type` is stored as `type_` instead, because
> Sentry reserves `type` inside every context object to denote the context kind.

### Slice Data

Slice data is captured as arrays in the `log` context:

```go
logger.Info("Processing batch").
    Strs("tags", []string{"batch", "processing", "urgent"}).
    Ints("user_ids", []int{123, 456, 789}).
    Log()
```

### JSON Data

JSON data is preserved as raw JSON in Sentry:

```go
metadata := map[string]any{
    "request_id": "req-123",
    "user_agent": "Mozilla/5.0...",
}
jsonData, _ := json.Marshal(metadata)

logger.Info("Request processed").
    JSON("metadata", jsonData).
    Log()
```

## Context Control

### Disabling Sentry Logging

You can disable Sentry logging for specific contexts:

```go
// Disable Sentry logging for this context
ctx := logsentry.ContextWithoutLogging(context.Background())

// This will not be sent to Sentry
logger.WithContext(ctx).Error("This won't go to Sentry").Log()

// Check if context has Sentry disabled
if logsentry.IsContextWithoutLogging(ctx) {
    // Handle accordingly
}
```

### Context-Aware Filtering

The writer respects golog's level filtering system:

```go
// Only send ERROR and FATAL to Sentry
sentryWriter := logsentry.NewWriterConfig(
    sentry.CurrentHub(),
    golog.NewDefaultFormat(),
    golog.DefaultLevels.Error.FilterOutBelow(), // Only ERROR and FATAL
    false,
    nil,
)
```

## Configuration Options

### WriterConfig Parameters

```go
func NewWriterConfig(
    hub *sentry.Hub,           // Sentry hub instance
    format *golog.Format,      // golog message format
    filter golog.LevelFilter,   // Level filtering
    valsAsMsg bool,            // Include values in message text
    extra map[string]any,      // Data added to the "log" context of every event
    opts ...Option,            // Optional settings, e.g. WithErrorHandler
) *WriterConfig
```

#### Parameters Explained

- **`hub`**: The Sentry hub instance to send events to. Use `sentry.CurrentHub()` for the default hub.
- **`format`**: golog format configuration for message formatting. Use `golog.NewDefaultFormat()` for standard formatting.
- **`filter`**: Level filter to control which log levels are sent to Sentry. Use `golog.AllLevelsActive` to send all levels.
- **`valsAsMsg`**: If `true`, includes key-value pairs in the message text. If `false`, only sends them in the `log` context.
- **`extra`**: Additional data to include in the `log` context of every Sentry event (e.g., service name, version).
- **`opts`**: Optional settings. Use `WithErrorHandler(func(error))` to route errors that occur while logging to Sentry into your normal logs (defaults to `golog.ErrorHandler()`).

### Global Configuration

```go
// Customize unknown level mapping
logsentry.UnknownLevel = sentry.LevelWarning

// Customize flush timeout
logsentry.FlushTimeout = 5 * time.Second
```

## How to route Sentry logging errors into your normal logs

By default, problems that happen while logging to Sentry are easy to miss: writer-side
failures go to `golog.ErrorHandler()` (stderr), and Sentry's own transport failures (HTTP 413
"payload too large", network errors) happen asynchronously inside the Sentry SDK and never
reach golog at all. This how-to routes both into a handler of your choice so they show up in
your normal, non-Sentry logs.

There are two independent error sources, each with its own hook:

| Source | What it catches | Hook |
|----------------------|----------------------------------------------|--------------------------------|
| Writer side | Recovered panics while building/sending event | `WithErrorHandler` option |
| Sentry transport | Async delivery failures (413, 5xx, network)  | `NewSentryDebugWriter` |

### Prerequisites

- A golog logger with a logsentry writer (see [Quick Start](#quick-start)).
- A place to send the errors. This how-to uses stderr; in a real app use your existing
  non-Sentry logger.

### Steps

1. Define one error handler and reuse it for both hooks.

   ```go
   onSentryError := func(err error) {
       fmt.Fprintln(os.Stderr, "sentry logging error:", err)
   }
   ```

   > **Do not** let this handler log back through the same Sentry-backed golog logger. A
   > failed send would emit another event, whose failure produces another error, looping under
   > backpressure. Route it to a plain sink (stderr, a file, or a console-only logger).

2. Wire `NewSentryDebugWriter` into the Sentry client to capture transport failures. This
   only works when `Debug: true` is also set (Sentry ignores `DebugWriter` otherwise).

   ```go
   err := sentry.Init(sentry.ClientOptions{
       Dsn:         os.Getenv("SENTRY_DSN"),
       Debug:       true,
       DebugWriter: logsentry.NewSentryDebugWriter(onSentryError),
   })
   ```

3. Pass `WithErrorHandler` to `NewWriterConfig` to capture writer-side errors.

   ```go
   sentryWriter := logsentry.NewWriterConfig(
       sentry.CurrentHub(),
       golog.NewDefaultFormat(),
       golog.DefaultLevels.Error.FilterOutBelow(),
       false,
       nil,
       logsentry.WithErrorHandler(onSentryError),
   )
   ```

### Full example

```go
package main

import (
    "fmt"
    "os"
    "time"

    "github.com/getsentry/sentry-go"
    "github.com/domonda/golog"
    "github.com/domonda/golog/logsentry"
)

func main() {
    // One handler, reused for both error sources. Writes to stderr — never
    // back through the Sentry-backed logger (that would loop).
    onSentryError := func(err error) {
        fmt.Fprintln(os.Stderr, "sentry logging error:", err)
    }

    // Debug:true + DebugWriter captures Sentry's async transport failures.
    err := sentry.Init(sentry.ClientOptions{
        Dsn:         os.Getenv("SENTRY_DSN"),
        Debug:       true,
        DebugWriter: logsentry.NewSentryDebugWriter(onSentryError),
    })
    if err != nil {
        panic("sentry.Init: " + err.Error())
    }
    defer sentry.Flush(2 * time.Second)

    // WithErrorHandler captures writer-side errors (recovered panics).
    sentryWriter := logsentry.NewWriterConfig(
        sentry.CurrentHub(),
        golog.NewDefaultFormat(),
        golog.DefaultLevels.Error.FilterOutBelow(),
        false,
        nil,
        logsentry.WithErrorHandler(onSentryError),
    )

    logger := golog.NewLogger(golog.NewConfig(
        &golog.DefaultLevels,
        golog.AllLevelsActive,
        sentryWriter,
    ))

    logger.Error("something broke").Log()
}
```

### Setting a default handler instead

If you do not pass `WithErrorHandler`, writer-side errors go to `golog.ErrorHandler()`, which
prints to stderr by default. Set a process-wide handler once at startup with
`golog.SetErrorHandler`:

```go
golog.SetErrorHandler(func(err error) {
    consoleLogger.Error("logging error").Err(err).Log()
})
```

`golog.SetErrorHandler(nil)` is valid and disables handling. Pass that same handler to
`NewSentryDebugWriter` to cover the transport side too.

### Verification

Point the DSN at an unreachable host and log an error:

```bash
SENTRY_DSN="https://public@10.255.255.1/1" go run .
```

After the `sentry.Flush` on shutdown, the Sentry SDK fails to deliver the event and your
handler prints a line such as:

```
sentry logging error: sentry: error sending envelope: ...
```

### Troubleshooting

- **Nothing from the transport side.** You set `DebugWriter` but not `Debug: true` — Sentry
  ignores the writer without it.
- **Errors appear twice, or output explodes.** Your handler logs back through the
  Sentry-backed logger. Route it to a plain sink.
- **You expected a line for a dropped event but saw none.** `NewSentryDebugWriter` forwards
  only genuine delivery failures. Intentional drops (sampling via `SampleRate`, `BeforeSend`
  filtering) and routine debug output are ignored on purpose.

## Runtime Behavior

### Memory Management

- **Object Pooling**: Writers are pooled and reused to minimize allocations
- **Value Map Pooling**: Key-value maps are pooled for efficient memory usage
- **Automatic Cleanup**: Writers are automatically reset and returned to pools after each message

### Performance Characteristics

- **Non-blocking**: Logging operations don't block the calling goroutine
- **Asynchronous**: Sentry SDK handles event transmission asynchronously
- **Batched**: Sentry SDK batches events for efficient network usage

### Error Handling

- **Graceful Degradation**: Application continues running even if Sentry integration fails
- **No Panics**: The package is designed to never panic during normal operation
- **Surfaceable Errors**: By default errors are sent to `golog.ErrorHandler()` (stderr). Pass
  `WithErrorHandler` to `NewWriterConfig` to route writer errors into your normal logs, and
  assign `NewSentryDebugWriter(handler)` to `sentry.ClientOptions.DebugWriter` (with
  `Debug: true`) to capture Sentry's asynchronous transport failures (HTTP 413, network errors)
  that otherwise never reach golog.

## Limitations

### Sentry-Specific Limitations

1. **Rate Limiting**: Sentry imposes rate limits on event ingestion. High-frequency logging may result in dropped events.

2. **Event Size Limits**: Sentry has limits on event size. Very large log messages or excessive extra data may be truncated.

3. **Network Dependency**: Requires network connectivity to Sentry servers. Events may be lost if network is unavailable.

4. **Sample Rate**: Sentry's sample rate setting affects which events are actually sent, not just which are logged.

### golog Integration Limitations

1. **Level Mapping**: Only standard golog levels are mapped. Custom levels default to `ERROR`.

2. **Format Dependencies**: Message formatting depends on the provided `golog.Format`. Changes to format affect Sentry message content.

3. **Context Propagation**: Context-based logging control only works when explicitly using `WithContext()`.

4. **Stack Trace Filtering**: Only filters frames from `github.com/domonda/golog` module. Other logging-related frames may still appear.

### General Limitations

1. **No Retry Logic**: Failed Sentry events are not retried by this package.

2. **No Local Buffering**: Events are sent immediately to Sentry (subject to Sentry SDK's internal buffering).

3. **No Compression**: Large log messages are not compressed before sending.

4. **Single Hub**: Each writer config is tied to a single Sentry hub instance.

## Troubleshooting

### Common Issues

1. **Events Not Appearing in Sentry**
   - Check Sentry DSN configuration
   - Verify network connectivity
   - Check Sentry project settings
   - Ensure sample rate is not too low

2. **Missing Stack Traces**
   - Enable `AttachStacktrace: true` in Sentry options
   - Check if frames are being filtered out

### Debugging

```go
// Enable Sentry debug mode
err := sentry.Init(sentry.ClientOptions{
    Dsn:   os.Getenv("SENTRY_DSN"),
    Debug: true, // Enable debug logging
})

// Check if context has Sentry disabled
if logsentry.IsContextWithoutLogging(ctx) {
    log.Println("Sentry logging is disabled for this context")
}
```

## Examples

### Web Application Integration

```go
package main

import (
    "net/http"
    "os"
    "time"

    "github.com/getsentry/sentry-go"
    "github.com/domonda/golog"
    "github.com/domonda/golog/logsentry"
)

func main() {
    // Initialize Sentry
    sentry.Init(sentry.ClientOptions{
        Dsn: os.Getenv("SENTRY_DSN"),
        AttachStacktrace: true,
    })
    defer sentry.Flush(2 * time.Second)

    // Create logger with Sentry integration
    sentryWriter := logsentry.NewWriterConfig(
        sentry.CurrentHub(),
        golog.NewDefaultFormat(),
        golog.DefaultLevels.Error.FilterOutBelow(), // Only errors and above
        false,
        map[string]any{
            "service": "web-api",
            "version": "1.0.0",
        },
    )

    config := golog.NewConfig(
        &golog.DefaultLevels,
        golog.AllLevelsActive,
        sentryWriter,
    )

    logger := golog.NewLogger(config)

    // HTTP handler with logging
    http.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        logger.Info("Request started").
            Str("method", r.Method).
            Str("path", r.URL.Path).
            Str("ip", r.RemoteAddr).
            Log()

        // Process request...
        
        logger.Info("Request completed").
            Str("method", r.Method).
            Str("path", r.URL.Path).
            Duration("duration", time.Since(start)).
            Int("status", 200).
            Log()
    })

    logger.Info("Server starting").Str("port", "8080").Log()
    http.ListenAndServe(":8080", nil)
}
```

### Microservice Integration

```go
package main

import (
    "context"
    "os"

    "github.com/getsentry/sentry-go"
    "github.com/domonda/golog"
    "github.com/domonda/golog/logsentry"
)

type Service struct {
    logger *golog.Logger
}

func NewService() *Service {
    // Initialize Sentry with service-specific configuration
    sentry.Init(sentry.ClientOptions{
        Dsn: os.Getenv("SENTRY_DSN"),
        AttachStacktrace: true,
        Environment: os.Getenv("ENVIRONMENT"),
        Release: os.Getenv("VERSION"),
    })

    // Create service-specific logger
    sentryWriter := logsentry.NewWriterConfig(
        sentry.CurrentHub(),
        golog.NewDefaultFormat(),
        golog.DefaultLevels.Warn.FilterOutBelow(), // Warnings and above
        false,
        map[string]any{
            "service": "user-service",
            "version": os.Getenv("VERSION"),
            "environment": os.Getenv("ENVIRONMENT"),
        },
    )

    config := golog.NewConfig(
        &golog.DefaultLevels,
        golog.AllLevelsActive,
        sentryWriter,
    )

    return &Service{
        logger: golog.NewLogger(config),
    }
}

func (s *Service) ProcessUser(ctx context.Context, userID string) error {
    s.logger.Info("Processing user").
        Str("user_id", userID).
        Log()

    // Process user...
    
    if err != nil {
        s.logger.Error("User processing failed").
            Str("user_id", userID).
            Err(err).
            Log()
        return err
    }

    s.logger.Info("User processed successfully").
        Str("user_id", userID).
        Log()

    return nil
}
```

## License

This package is part of the golog project and follows the same MIT license. See the main golog repository for license details.

# logsentry

A Sentry integration package for [golog](https://github.com/domonda/golog), providing seamless structured logging to Sentry error monitoring platform.

## Overview

The `logsentry` package implements the `golog.Writer` and `golog.WriterConfig` interfaces to bridge golog's structured logging capabilities with Sentry's error tracking and monitoring system. It automatically maps golog log levels to Sentry event levels and captures structured log data as Sentry events.

## Features

- **Automatic Level Mapping**: Maps golog log levels to appropriate Sentry event levels
- **Structured Data Capture**: Converts golog key-value pairs to Sentry event extra data
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
    golog.ErrorLevel().FilterOutBelow(), // Only send ERROR and FATAL to Sentry
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

All golog key-value pairs are automatically captured as Sentry event extra data:

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
- **Extra Data**:
  - `username`: "john_doe"
  - `ip_address`: "192.168.1.100"
  - `attempt_count`: 3
  - `account_locked`: false
  - `response_time`: 0.145

### Slice Data

Slice data is captured as arrays in Sentry extra data:

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
    golog.ErrorLevel().FilterOutBelow(), // Only ERROR and FATAL
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
    extra map[string]any,      // Extra data for all events
) *WriterConfig
```

#### Parameters Explained

- **`hub`**: The Sentry hub instance to send events to. Use `sentry.CurrentHub()` for the default hub.
- **`format`**: golog format configuration for message formatting. Use `golog.NewDefaultFormat()` for standard formatting.
- **`filter`**: Level filter to control which log levels are sent to Sentry. Use `golog.AllLevelsActive` to send all levels.
- **`valsAsMsg`**: If `true`, includes key-value pairs in the message text. If `false`, only sends them as extra data.
- **`extra`**: Additional data to include with every Sentry event (e.g., service name, version).

### Global Configuration

```go
// Customize unknown level mapping
logsentry.UnknownLevel = sentry.LevelWarning

// Customize flush timeout
logsentry.FlushTimeout = 5 * time.Second
```

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

- **Silent Failures**: If Sentry is unavailable, logging continues without errors
- **Graceful Degradation**: Application continues running even if Sentry integration fails
- **No Panics**: The package is designed to never panic during normal operation

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
        golog.ErrorLevel().FilterOutBelow(), // Only errors and above
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
        golog.WarnLevel().FilterOutBelow(), // Warnings and above
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

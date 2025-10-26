# PostHog Writer for golog

This package provides a PostHog writer for the [golog](https://github.com/domonda/golog) logging library.

## Features

- Sends log messages as PostHog events
- Supports structured logging with key-value pairs
- Configurable extra properties for all log messages
- Proper level name handling
- Writer pooling for performance
- Context-based logging control

## Usage

### Basic Setup

```go
package main

import (
    "context"
    "os"
    
    "github.com/domonda/golog"
    "github.com/domonda/golog/posthog"
)

func main() {
    // Set your PostHog API key
    os.Setenv("POSTHOG_API_KEY", "your-api-key-here")
    
    // Create PostHog writer config
    config, err := posthog.NewWriterConfigFromEnv(
        golog.NewDefaultFormat(),     // Use default format
        golog.AllLevelsActive,        // Use level filter that allows all levels
        "system",                     // DistinctId for tracking events
        true,                         // Include values in message
        map[string]any{               // Extra properties for all logs
            "service": "my-service",
            "version": "1.0.0",
        },
    )
    if err != nil {
        panic(err)
    }
    
    // Create logger with PostHog writer
    logger := golog.NewLogger(golog.NewConfig(
        &golog.DefaultLevels,          // Use default levels
        golog.AllLevelsActive,         // Use level filter that allows all levels
        config,
    ))
    
    // Log messages
    ctx := context.Background()
    logger.NewMessage(ctx, golog.DefaultLevels.Info, "User action").
        Str("user_id", "12345").
        Str("action", "login").
        Log()
}
```

### Custom PostHog Client

```go
import (
    "github.com/posthog/posthog-go"
    "github.com/domonda/golog/posthog"
)

// Create custom PostHog client
client, err := posthog.NewWithConfig(
    "your-api-key",
    posthog.Config{
        Endpoint: "https://eu.i.posthog.com", // EU endpoint
        PersonalApiKey: "your-personal-api-key",
    },
)
if err != nil {
    panic(err)
}

// Create writer config with custom client
config := posthog.NewWriterConfig(
    client,
    golog.NewDefaultFormat(),
    golog.AllLevelsActive,
    "system",                        // DistinctId for tracking events
    true,
    map[string]any{"service": "my-service"},
)
```

## Configuration Options

- `format`: golog.Format for message formatting
- `filter`: golog.LevelFilter for level filtering
- `distinctID`: PostHog's unique identifier for tracking events (see DistinctId section below)
- `valsAsMsg`: Whether to include key-value pairs in the message text
- `extra`: Map of extra properties added to every log event

## PostHog Event Structure

Each log message is sent as a PostHog event with:

- **Event Name**: `"log_message"`
- **Distinct ID**: Configurable via `distinctID` parameter
- **Properties**:
  - `message`: The log message text
  - `log_level`: The log level name (e.g., "INFO", "ERROR")
  - Additional properties from `extra` config
  - Key-value pairs from structured logging

## DistinctId Configuration

The `distinctID` parameter is PostHog's unique identifier for tracking events. It serves to associate log events with specific users or system components.

### Common Patterns

- **System logs**: `"system"`, `"server"`, `"backend"`
- **User-specific logs**: `"user_12345"`, `"user@example.com"`
- **Service-specific logs**: `"service_api"`, `"service_database"`
- **Job-specific logs**: `"cron_backup"`, `"worker_email"`

### Restricted Values

PostHog restricts certain values that cannot be used as `distinctID`:
- `"anonymous"`, `"guest"`, `"distinctid"`, `"undefined"`
- `"null"`, `"true"`, `"false"`, `"0"`
- `"[object Object]"`, `"NaN"`, `"None"`, `"none"`

### Examples

```go
// System logs
config, _ := posthog.NewWriterConfigFromEnv(format, filter, "system", valsAsMsg, extra)

// User-specific logs (when you have user context)
config, _ := posthog.NewWriterConfigFromEnv(format, filter, "user_12345", valsAsMsg, extra)

// Service-specific logs
config, _ := posthog.NewWriterConfigFromEnv(format, filter, "service_api", valsAsMsg, extra)
```

## Disabling PostHog Logging

Use the context functions to disable PostHog logging for specific operations:

```go
ctx := posthog.ContextWithoutLogging(context.Background())
logger.NewMessage(ctx, golog.DefaultLevels.Info, "This won't be sent to PostHog").Log()
```

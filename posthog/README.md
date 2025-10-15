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
        golog.NewDefaultFormat(),      // Use default format
        golog.AllLevelsActive,         // Use level filter that allows all levels
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
    true,
    map[string]any{"service": "my-service"},
)
```

## Configuration Options

- `format`: golog.Format for message formatting
- `filter`: golog.LevelFilter for level filtering
- `valsAsMsg`: Whether to include key-value pairs in the message text
- `extra`: Map of extra properties added to every log event

## PostHog Event Structure

Each log message is sent as a PostHog event with:

- **Event Name**: `"log_message"`
- **Distinct ID**: `"system"` (configurable)
- **Properties**:
  - `message`: The log message text
  - `log_level`: The log level name (e.g., "INFO", "ERROR")
  - Additional properties from `extra` config
  - Key-value pairs from structured logging

## Disabling PostHog Logging

Use the context functions to disable PostHog logging for specific operations:

```go
ctx := posthog.ContextWithoutLogging(context.Background())
logger.NewMessage(ctx, golog.DefaultLevels.Info, "This won't be sent to PostHog").Log()
```

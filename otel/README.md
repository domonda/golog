# golog/otel

OpenTelemetry Log bridge for [golog](https://github.com/domonda/golog). Emits golog messages as OpenTelemetry log records via the [OTel Log API](https://pkg.go.dev/go.opentelemetry.io/otel/log).

## Installation

```bash
go get github.com/domonda/golog/otel
```

## Usage

```go
package main

import (
	"context"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	"github.com/domonda/golog"
	logotel "github.com/domonda/golog/otel"
)

func main() {
	ctx := context.Background()

	// 1. Set up an OTel SDK log provider with an exporter
	exporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint("localhost:4318"),
		otlploghttp.WithInsecure(),
	)
	if err != nil {
		panic(err)
	}
	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)
	defer provider.Shutdown(ctx)

	// 2. Create a golog WriterConfig backed by the OTel provider
	writerConfig := logotel.NewWriterConfig(
		provider,
		golog.NewDefaultFormat(),
		golog.AllLevelsActive,
		log.String("service", "my-app"), // static attributes added to every record
	)

	// 3. Use it as a golog writer (standalone or alongside other writers)
	logger := golog.NewLogger(golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, writerConfig))

	logger.Info("User logged in").Str("user", "erik").Log()
	logger.Error("Query failed").Err(err).Str("table", "users").Log()
}
```

### Adding OTel alongside existing writers

```go
config := golog.NewConfig(
	&golog.DefaultLevels,
	golog.AllLevelsActive,
	existingTextWriter,  // keeps console output
	otelWriterConfig,    // adds OTel export
)
logger := golog.NewLogger(config)
```

### Disabling OTel logging per context

```go
ctx := logotel.ContextWithoutLogging(ctx)
logger.InfoCtx(ctx, "This won't be sent to OTel").Log()
```

## Severity mapping

| golog Level | OTel Severity      |
|-------------|--------------------|
| Trace (-20) | SeverityTrace (1)  |
| Debug (-10) | SeverityDebug (5)  |
| Info (0)    | SeverityInfo (9)   |
| Warn (10)   | SeverityWarn (13)  |
| Error (20)  | SeverityError (17) |
| Fatal (30)  | SeverityFatal (21) |

Custom or unmapped levels use `UnknownSeverity` (defaults to `SeverityError`). Override it before creating the WriterConfig:

```go
logotel.UnknownSeverity = log.SeverityWarn
```

## Differences from golog defaults

### FlushUnderlying is a no-op

The OTel Log API (`log.LoggerProvider`) does not expose flush or shutdown methods. Unlike `logsentry` where `FlushUnderlying` calls `hub.Flush()`, the OTel bridge cannot flush the export pipeline.

You must call shutdown/flush on the SDK provider directly:

```go
provider := sdklog.NewLoggerProvider(...)
defer provider.Shutdown(ctx)   // flushes and shuts down
provider.ForceFlush(ctx)       // flushes without shutdown
```

### Type conversions

The OTel Log API has a fixed set of value types. Some golog types are converted:

| golog type         | OTel value type | Notes                                                            |
|--------------------|-----------------|------------------------------------------------------------------|
| `int64`            | `Int64Value`    |                                                                  |
| `uint64`           | `Int64Value`    | Values > `math.MaxInt64` are stored as string to avoid data loss |
| `float64`          | `Float64Value`  |                                                                  |
| `bool`             | `BoolValue`     |                                                                  |
| `string`           | `StringValue`   |                                                                  |
| `error`            | `StringValue`   | Stored as `error.Error()`                                        |
| `time.Time`        | `StringValue`   | Formatted as RFC3339Nano                                         |
| `[16]byte` (UUID)  | `StringValue`   | Formatted as standard UUID string                                |
| `[]byte` (JSON)    | `StringValue`   | Raw JSON stored as string                                        |
| `nil`              | empty `Value`   | OTel `KindEmpty`                                                 |
| slice              | `SliceValue`    |                                                                  |

### Record structure

Each golog message becomes one OTel `log.Record`:

- **Body**: the message text (with prefix formatting applied)
- **Severity**: mapped from golog level (see table above)
- **SeverityText**: level name string (e.g. "INFO", "ERROR")
- **Timestamp**: the original golog message timestamp
- **Attributes**: static attrs from `NewWriterConfig` + per-message key-value pairs

## Local testing

A test setup with a dockerized OTel Collector is included in `testserver/`:

```bash
cd otel/testserver

# Start the collector (receives OTLP on port 4318, prints to stdout)
docker-compose up -d

# Send test messages at all log levels
go run .

# View received log records
docker-compose logs otel-collector

# Clean up
docker-compose down
```

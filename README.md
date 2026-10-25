# golog

Zero-allocation structured logging for Go. The same logger writes colored text in a terminal and JSON everywhere else.

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/domonda/golog)
[![Go Reference](https://pkg.go.dev/badge/github.com/domonda/golog.svg)](https://pkg.go.dev/github.com/domonda/golog)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

In production at [domonda](https://www.domonda.com) since 2023.

## Install

```bash
go get github.com/domonda/golog
```

## Quick Start

```go
package main

import (
    "errors"

    "github.com/domonda/golog/log"
)

func main() {
    log.Info("Server started").Int("port", 8080).Log()
    log.Error("Payment failed").Str("order", "A-1001").Err(errors.New("card declined")).Log()
}
```

In a terminal the output is colored text:

```text
2026-10-31 13:24:01.270 |INFO | Server started port=8080
2026-10-31 13:24:01.270 |ERROR| Payment failed order="A-1001" error=`card declined`
```

Piped to a file or a log collector, the same program writes JSON:

```json
{"time":"2026-10-31 13:24:01.270","level":"INFO","message":"Server started","port":8080}
{"time":"2026-10-31 13:24:01.270","level":"ERROR","message":"Payment failed","order":"A-1001","error":"card declined"}
```

Set the `LOG_LEVEL` environment variable to `TRACE`, `DEBUG`, `INFO`, `WARN`, `ERROR` or `FATAL` to choose the lowest level that gets logged. The default is `DEBUG`.

## Features

- **Logs without allocating** — 0 allocations per message for JSON and uncolored text output, including errors and `time.Time` fields.
- **Picks the output format** — the `log` package writes colored text to a terminal and JSON when output is piped, with no setup.
- **Writes to several places at once** — for example colored text on stdout plus warnings and errors as JSON in a size-rotated file, each writer with its own format and level filter.
- **Reports errors to Sentry** — the `logsentry` module turns errors logged with `Err` into Sentry exceptions, keeping the stack trace the error carries.
- **Carries request context** — attributes stored in a `context.Context` appear on every message logged with it. The HTTP middleware puts a request ID there, taken from `X-Request-ID` or generated.
- **Protects inherited keys** — a message can't duplicate or override a key its logger or sub-logger already set, such as `service`.
- **Logs structs by their tags** — reuses your `json` tags (or `golog` and `log` tags) with `omitempty`-style modifiers, and prints `***REDACTED***` for fields tagged `redact`.
- **Backs `log/slog`** — the `goslog` module turns a golog logger into a `slog.Handler` that passes the standard library's `slogtest` suite.
- **Ignores a nil logger** — `Info`, `Error` and the other level methods on a nil `*Logger` return a message that logs nothing, instead of panicking.

`goslog` and `logsentry` are separate modules, so the core module doesn't pull in the Sentry SDK:

```bash
go get github.com/domonda/golog/goslog
go get github.com/domonda/golog/logsentry
```

## Performance

| Apple M2, per message | golog            | zerolog            | zap              | slog             |
|-----------------------|------------------|--------------------|------------------|------------------|
| JSON, 4 fields        | 353 ns, 0 allocs | 145 ns, 0 allocs   | 451 ns, 1 alloc  | 860 ns, 9 allocs |
| Text, no colors       | 394 ns, 0 allocs | 2363 ns, 46 allocs | 514 ns, 4 allocs | 677 ns, 3 allocs |
| Level disabled        | 42 ns, 0 allocs  | 6 ns, 0 allocs     | 35 ns, 1 alloc   | 47 ns, 3 allocs  |

zerolog is faster at JSON. golog is the fastest at uncolored text output in these benchmarks and allocates nothing for either. Messages and writers come from object pools, which is how the per-message allocations reach zero. Colored terminal output allocates a few strings per message for the color codes.

The comparative benchmarks live in their own module, so zerolog, zap and logrus never become dependencies of golog:

```bash
go test ./benchmarks -bench=. -benchmem   # golog vs zerolog, zap, slog, logrus
go test -bench=. -benchmem                # golog's own benchmarks
```

All scenarios and results: [benchmarks/README.md](benchmarks/README.md).

## Configuring a Logger

The `log` package covers the common case. Build your own logger with `golog.NewConfig` when you need other writers, formats or levels.

### Text Output

```go
package main

import (
    "errors"
    "os"

    "github.com/domonda/golog"
)

func main() {
    config := golog.NewConfig(
        &golog.DefaultLevels,
        golog.AllLevelsActive,
        golog.NewTextWriterConfig(os.Stdout, nil, nil),
    )

    log := golog.NewLogger(config)

    log.Info("Hello, World!").Log()
    log.Error("Something went wrong").Err(errors.New("example error")).Log()
}
```

### JSON Output

```go
config := golog.NewConfig(
    &golog.DefaultLevels,
    golog.AllLevelsActive,
    golog.NewJSONWriterConfig(os.Stdout, nil),
)

log := golog.NewLogger(config)

start := time.Now()
// ... perform login ...
log.Info("User login").
    Str("username", "john_doe").
    Str("ip", "192.168.1.1").
    Duration("login_time", time.Since(start)).
    Log()
```

Output (matches `NewDefaultFormat`: `"time"` key, `"2006-01-02 15:04:05.000"` layout):
```json
{"time":"2024-01-15 10:30:45.000","level":"INFO","message":"User login","username":"john_doe","ip":"192.168.1.1","login_time":"150ms"}
```

### Typed Fields

```go
requestID := golog.UUIDv4()           // [16]byte UUID
jsonBytes := []byte(`{"key":"value"}`) // Raw JSON

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
log.Debug("Variable state").Int("counter", 42).Log()
log.Info("Operation completed").Log()
log.Warn("Deprecated API used").Str("api", "/old/endpoint").Log()
log.Error("Failed to connect").Err(err).Log()
log.Fatal("Critical system failure").Log()
```

## Sub-loggers and Context

### Creating Sub-loggers

```go
requestID := golog.UUIDv4()

// Create a sub-logger with common attributes
subLog := log.With().
    Str("service", "user-management").
    UUID("request_id", requestID).
    SubLogger()

// All logs from subLog will include the above attributes
subLog.Info("User created").Str("username", "john").Log()
subLog.Error("User validation failed").Err(errors.New("invalid email")).Log()
```

### Context Integration

```go
// Add attributes to context
ctx = golog.ContextWithAttribs(ctx,
    golog.NewString("correlation_id", "abc-123"),
    golog.NewString("user_id", "user-456"),
)

// Create logger from context
ctxLogger := log.WithCtx(ctx)
ctxLogger.Info("Operation started").Log() // Includes context attributes

// Or use the logger's InfoCtx method
log.InfoCtx(ctx, "Operation started").Log() // Includes context attributes
```

If a message sets a key that the logger or a sub-logger already provides, the message's attribute is dropped and the inherited value is kept. Context attributes that repeat a logger key are dropped the same way. A message attribute that repeats a context key is written as a second field, unless the logger was created with `WithCtx(ctx)`, which turns the context attributes into logger attributes.

## Multiple Writers and Filtering

```go
logFile, _ := os.Create("app.log")
defer logFile.Close()

// Log to multiple outputs with different formats
config := golog.NewConfig(
    &golog.DefaultLevels,
    golog.AllLevelsActive,
    golog.NewTextWriterConfig(os.Stdout, nil, log.NewStyledColorizer()), // github.com/domonda/golog/log
    // FilterOutBelow filters out less severe levels (lower numeric values)
    // Warn=10, so this filters out Trace=-20, Debug=-10, Info=0
    // Only Warn=10, Error=20, Fatal=30 will be logged to the file
    golog.NewJSONWriterConfig(logFile, nil, golog.DefaultLevels.Warn.FilterOutBelow()),
)

logger := golog.NewLogger(config)

// This will appear in colored text on stdout and as JSON in the file (if WARN+)
logger.Error("Database connection failed").Err(errors.New("connection refused")).Log()
```

## Terminal Detection

golog can switch between human-readable text and machine-readable JSON output based on whether the process is attached to a terminal (TTY):

```go
config := golog.NewConfig(
    &golog.DefaultLevels,
    golog.AllLevelsActive,
    golog.DecideWriterConfigForTerminal(
        golog.NewTextWriterConfig(os.Stdout, nil, log.NewStyledColorizer()), // Used when running in terminal
        golog.NewJSONWriterConfig(os.Stdout, nil),                           // Used when output is piped/redirected
    ),
)

logger := golog.NewLogger(config)
```

- **Terminal (TTY)**: colored, human-readable text
- **Non-terminal**: JSON for log collectors and files

You can also check the terminal status directly:

```go
if golog.IsTerminal() {
    // Running in a terminal
}
```

## Ready-to-Use Logger (`log` subpackage)

The `log` subpackage provides a pre-configured logger:

```go
import (
    "errors"

    "github.com/domonda/golog/log"
)

func main() {
    log.Info("Application started").Log()
    log.Error("Something went wrong").Err(errors.New("example error")).Log()
    log.Debug("Debug information").Str("key", "value").Log()
}
```

### Default Configuration

The `log` package is configured with:

- **Log Levels**: Uses `golog.DefaultLevels` (TRACE, DEBUG, INFO, WARN, ERROR, FATAL)
- **Level Filter**: Filters out levels below the `LOG_LEVEL` environment variable (defaults to DEBUG if not set)
- **Output**: Writes to stdout with automatic format selection:
  - **Terminal**: Colorized text format for human readability
  - **Non-terminal**: JSON format for log aggregation systems

### Customizing the Default Logger

The `log.Config` variable can be modified at runtime:

```go
import "github.com/domonda/golog/log"

func init() {
    // Change the minimum log level
    // Info=0, so FilterOutBelow filters out Trace=-20 and Debug=-10
    // Only Info=0, Warn=10, Error=20, Fatal=30 will be logged
    log.Config = golog.NewConfig(
        log.Levels,
        log.Levels.Info.FilterOutBelow(),
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

// With context - attributes added to context are automatically included
ctx := golog.ContextWithAttribs(context.Background(),
    golog.NewString("request_id", "req-123"),
    golog.NewString("user_id", "user-456"),
)
log.InfoCtx(ctx, "processing request").Log() // Includes request_id and user_id

// Formatted messages
log.Infof("User %s logged in", "john_doe").Log()

// Create sub-loggers
subLog := log.With().Str("component", "auth").SubLogger()
```

## Rotating Log Files

The `logfile` subpackage rotates log files when they reach a size threshold:

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
    panic(err)
}
defer writer.Close()

// Use with golog
config := golog.NewConfig(
    &golog.DefaultLevels,
    golog.AllLevelsActive,
    golog.NewJSONWriterConfig(writer, nil),
)

logger := golog.NewLogger(config)

logger.Info("Application started").Log()
```

Before a write that would make the file reach 10MB:
- The current file is renamed with a timestamp (e.g., `myapp.log.2024-01-15_10:30:45`)
- A new file is created at the original path, and the write goes to the new file

### Multiple Writers with Rotation

```go
// Console output with colors + rotating JSON file
fileWriter, _ := logfile.NewRotatingWriter("/var/log/app.log", "", 0644, 50*1024*1024)
defer fileWriter.Close()

config := golog.NewConfig(
    &golog.DefaultLevels,
    golog.AllLevelsActive,
    golog.NewTextWriterConfig(os.Stdout, nil, log.NewStyledColorizer()), // github.com/domonda/golog/log
    golog.NewJSONWriterConfig(fileWriter, nil),
)

logger := golog.NewLogger(config)
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

Code written against `slog` keeps its API and gets golog's writers and level filters. The handler passes the `slogtest` compliance suite. See the [goslog package documentation](goslog/README.md) for more details.

## Sentry

The `logsentry` module provides a writer that sends log messages to Sentry. An error logged with `Err` becomes a Sentry exception. Its stack trace comes from the error when it carries one, otherwise from the call site if the Sentry client has `AttachStacktrace` enabled. See the [logsentry package documentation](logsentry/README.md) for setup.

## HTTP Middleware

`golog.HTTPMiddlewareHandler` logs each incoming request and attaches a request ID to it:

```go
logger := golog.NewLogger(config)

mux := http.NewServeMux()
mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
    // Includes the requestID attribute set by the middleware
    logger.InfoCtx(r.Context(), "Listing users").Log()
})

handler := golog.HTTPMiddlewareHandler(mux, logger, golog.DefaultLevels.Info, "HTTP request")
http.ListenAndServe(":8080", handler)
```

The middleware:
- takes the request ID from the `X-Request-ID` header, or from `X-Correlation-ID` when `X-Request-ID` is missing; if that value isn't a UUID, it generates a random v4 UUID
- stores it as the `requestID` attribute in the request context and sets it as the `X-Request-ID` response header
- logs the remote address, method, URI and headers; pass header names after the message to log only those, or `golog.HTTPNoHeaders` to log none

`golog.HTTPMiddlewareFunc` returns the same middleware as a `func(http.Handler) http.Handler`, the form routers like gorilla/mux expect.

## Advanced Features

### Custom Colorizers

```go
colorizer := log.NewStyledColorizer() // github.com/domonda/golog/log
// or implement your own golog.Colorizer interface

config := golog.NewConfig(
    &golog.DefaultLevels,
    golog.AllLevelsActive,
    golog.NewTextWriterConfig(os.Stdout, nil, colorizer),
)
```

`log.NewStyledColorizer` honors the `NO_COLOR` and `CLICOLOR`/`CLICOLOR_FORCE` environment variables.

### Call Stack Logging

```go
defer func() {
    if r := recover(); r != nil {
        log.Error("Panic recovered").
            CallStack("stack").
            Any("panic_value", r).
            Log()
    }
}()
```

### Struct Field Logging: Tags and Modifiers

`StructFields` and `TaggedStructFields` walk a struct via reflection and log
each matching field as an attribute. They are driven entirely by struct tags,
with modifiers that mirror `encoding/json.Marshal` semantics plus a handful of
golog-specific additions.

#### Tag resolution

`StructFields(s)` looks for tags in this order — **first match wins**:

1. `golog:"..."`
2. `log:"..."`
3. `json:"..."`

Only the first tag present on a field is consulted. Other tags on the same field
are ignored. Fields with **none** of those tags are skipped entirely (this is a
deliberate difference from `encoding/json.Marshal`, which would use the Go field
name: golog errs on the side of not leaking untagged internal fields into logs).

`TaggedStructFields(s, "json")` does the same with a single caller-chosen tag.

**Wildcard escape hatch:** passing the empty string as the key tag,
`TaggedStructFields(s, "")`, matches every exported field with no rename and no
modifiers — every field is logged under its Go name, struct tags ignored. Use
this when you want to dump a struct wholesale and you don't care about per-field
tags. (`StructFields(s)` deliberately stays strict tag-driven; the wildcard is
opt-in.)

#### Tag value grammar

Each tag value is `name,mod1,mod2,...`:

| Tag value                    | Log key       | Notes                                     |
|------------------------------|---------------|-------------------------------------------|
| `"field_name"`               | `field_name`  | explicit name                             |
| `""`                         | Go field name | empty → fall back to field name            |
| `",omitempty"`               | Go field name | empty name with modifier → fall back       |
| `"-"` (bare, no comma)       | *skipped*     | only the bare form skips                  |
| `"-,..."` (comma follows)    | `-`           | literal field name `-` + modifiers        |

Whitespace around the name and each modifier is trimmed. Unknown modifier
tokens are ignored silently, so `json:"name,omitnull"` is a valid tag for both
`encoding/json.Marshal` (which ignores `omitnull`) and golog (which honors it).

#### Modifiers

| Modifier    | Suppress when…                                                                 |
|-------------|---------------------------------------------------------------------------------|
| `omitempty` | `false`, `0`, nil pointer/interface, empty array/slice/map/string. Exactly matches `encoding/json.Marshal`. Does **not** catch zero structs or `time.Time{}`. |
| `omitzero`  | The type's `IsZero() bool` method returns true, **or** the value is the Go zero value (via `reflect.Value.IsZero`). Catches `time.Time{}` and any zero struct. Strict superset of `encoding/json`'s Go 1.24 `omitzero`. Both value-receiver and pointer-receiver `IsZero` methods are honored. |
| `omitnull`  | The type's `IsNull() bool` method returns true. Falls back to `omitzero` semantics when the type has no `IsNull` method. Use for nullable wrappers with an `IsNull` method (`golog.Timestamp`, `uu.NullableID`) where "null" is richer than "all bytes zero". Both value-receiver and pointer-receiver `IsNull` methods are honored. |
| `redact`    | (not a suppression modifier) — replaces the value with `"***REDACTED***"` before it reaches the writer. Also spelled `redacted`. Suppression modifiers win over redact: `json:",redact,omitempty"` on an empty string emits nothing, not the marker. |

Modifiers OR together — any passing check suppresses the field.

#### Complete example

```go
type User struct {
    ID        int             `json:"id"`
    Name      string          `json:"name,omitempty"`
    APIKey    string          `json:",redact"`
    CreatedAt time.Time       `json:"created_at,omitzero"`
    DeletedAt golog.Timestamp `json:"deleted_at,omitnull"`
    Internal  string          // no json/log/golog tag → never logged
}

user := User{
    ID:        123,
    Name:      "john_doe",
    APIKey:    "sk-abc123",
    CreatedAt: time.Now(),
    // DeletedAt left as zero Timestamp
    Internal:  "debug-only",
}

log.Info("user").TaggedStructFields(user, "json").Log()
// Output: ... id=123 name="john_doe" APIKey="***REDACTED***" created_at="2026-04-14T..."
// (DeletedAt suppressed by omitnull, Internal never considered.)
```

> **Changed in v1.0.6**
>
> - `log:"redact"` (bare, no comma) **no longer triggers redaction** — under the unified parser it names the field `"redact"`. Put the modifier after a comma: `log:",redact"` or `golog:",redact"` (combinable with other modifiers: `golog:",redact,omitempty"`).
> - `StructFields(s)` on an untagged struct now logs **nothing**. Previously it logged every exported field by its Go name. The cleanest replacement is `TaggedStructFields(s, "")`, the wildcard escape hatch documented above. Per-field, you can also add an empty tag like `json:""` or `golog:""` to opt in.
> - `TaggedStructFields(s, "json")` with `json:""` now logs the field (Go field name), previously it skipped. This is `encoding/json.Marshal` parity.

`omitnull` vs `omitzero`, concretely: take a type `NullString struct { String string; Valid bool }` with `func (n NullString) IsNull() bool { return !n.Valid }`. The value `NullString{String: "stale", Valid: false}` is null but not the Go zero value, so `omitnull` drops it and `omitzero` keeps it. Types without an `IsNull` method, such as `sql.NullString`, get `omitzero` semantics.

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

Log levels have numeric values where lower = more verbose, higher = more severe:
`Trace=-20 < Debug=-10 < Info=0 < Warn=10 < Error=20 < Fatal=30`

```go
// FilterOutBelow filters out less severe levels (lower numeric values)
// Warn=10, so this filters out Trace=-20, Debug=-10, Info=0
// Result: only Warn, Error, Fatal are logged
filter := golog.DefaultLevels.Warn.FilterOutBelow()

// Filter out only DEBUG level (Debug=-10)
debugFilter := golog.DefaultLevels.Debug.FilterOut()

// Filter out everything except INFO (Info=0)
infoOnlyFilter := golog.DefaultLevels.Info.FilterOutAllOther()

// Combine multiple filters (filters are OR'd together)
// This filters out both Trace and Debug
combinedFilter := golog.JoinLevelFilters(
    golog.DefaultLevels.Trace.FilterOut(),
    golog.DefaultLevels.Debug.FilterOut(),
)
```

### Logging in a Fixed Timezone

Set `Format.Location` to render every formatted time value, both the log line
timestamp and structured `time.Time` attributes, in a fixed timezone. The writer
calls `time.Time.In` on each value before formatting, so the underlying instant
is preserved. Leave it nil (the default) to keep each value's original location.

```go
tokyo, _ := time.LoadLocation("Asia/Tokyo")

format := golog.NewDefaultFormat()
format.Location = tokyo

config := golog.NewConfig(
    &golog.DefaultLevels,
    golog.AllLevelsActive,
    golog.NewJSONWriterConfig(os.Stdout, format),
)
```

### Parsing Log Timestamps

Use `golog.Timestamp` when you need to read log timestamps back out of JSON, a database column,
or any other place where the format varies. It wraps `time.Time` with JSON, `sql.Scanner`,
`driver.Valuer`, and null semantics, and parses a wide range of common layouts out of the box:

```go
// JSON unmarshal: accepts RFC3339(+nano), golog's own
// "2006-01-02 15:04:05.000", Apache/NGINX CLF, RFC1123(Z),
// syslog Stamp, compact ISO 8601, Go log default, or a JSON
// integer interpreted as Unix epoch seconds. JSON null is zero.
type logLine struct {
    Time    golog.Timestamp `json:"time"`
    Level   string          `json:"level"`
    Message string          `json:"message"`
}

// Directly parse a string with any supported layout:
ts, err := golog.ParseTimestamp("2024-01-15 10:30:45.000")

// Plug into database/sql — SQL NULL maps to the zero value:
var ts golog.Timestamp
err = db.QueryRow("SELECT logged_at FROM events WHERE id = $1", id).Scan(&ts)
if ts.IsNull() {
    // column was NULL
}
```

The accepted layouts live in the exported `golog.TimestampFormats` slice, so you can extend
or replace it before use if your log source needs extra layouts. `MarshalJSON` always emits
`DefaultTimeFormat` (`time.RFC3339Nano`), or `null` when the value is zero.

For storing the current request's event time in `context.Context`, use
`golog.ContextWithTimestamp(ctx, t)` together with `golog.TimestampFromContextOrNow(ctx)`
(returns `time.Now()` when no timestamp is stored or the stored value is zero).

> **Changed in v1.0.6:** the old `golog.Timestamp(ctx context.Context) time.Time`
> helper has been renamed to `golog.TimestampFromContextOrNow(ctx)` to free the `Timestamp`
> identifier for the new type. `ContextWithTimestamp` is now generic over
> `time.Time | golog.Timestamp`.

## Comparison with zerolog and zap

| Behavior                                                | zerolog              | zap             | golog |
|---------------------------------------------------------|----------------------|-----------------|-------|
| Zero allocations, JSON with fields                      | Yes                  | No (1 alloc)    | Yes   |
| Zero allocations, uncolored text output                 | No (46 allocs)       | No (4 allocs)   | Yes   |
| Drops message keys that repeat a logger or sub-logger key | No                   | No              | Yes   |
| Merges attributes stored in `context.Context`           | No (stores a logger) | No              | Yes   |

All three can write to several outputs: zerolog with `MultiLevelWriter`, zap with `zapcore.NewTee`, golog by passing several writer configs to `NewConfig`.

Choose something else when:

- **zerolog**: JSON throughput is the priority. It is about 2.4× faster than golog at JSON (145 vs 353 ns with 4 fields), but its `ConsoleWriter` text output is about 6× slower (2363 vs 394 ns).
- **zap**: you prefer typed `zap.Field` values passed as variadic arguments over a fluent builder.
- **slog**: you want no dependencies beyond the standard library. You can still add golog later as a `slog` backend through `goslog`.

## API Reference

### Core Types

- **Logger**: The logger; its level methods and `With` are safe to call on a nil `*Logger`
- **Message**: Fluent message builder returned by the level methods
- **Config**: Interface for logger configuration, created with `NewConfig`
- **WriterConfig**: Interface for an output writer with its format and level filter
- **Level**: Log level type
- **LevelFilter**: Bit mask of levels to filter out
- **Format**: Layout config for timestamp keys, timestamp layouts, level keys, message keys, structured `time.Time` attributes, and an optional `*time.Location` to render every time value in a fixed timezone
- **Timestamp**: `time.Time` wrapper with JSON, `database/sql.Scanner`/`driver.Valuer`, and null semantics, tuned for parsing log timestamps in many common formats

### Writer Types

- **JSONWriter**: Structured JSON output
- **TextWriter**: Human-readable text output
- **CallbackWriter**: Custom callback-based writer
- **NopWriter**: No-operation writer for testing

To write to several outputs, pass several writer configs to `NewConfig`.

For complete API documentation, see [pkg.go.dev](https://pkg.go.dev/github.com/domonda/golog). Changes per release are listed in [CHANGELOG.md](CHANGELOG.md).

## Contributing

The repo is a Go workspace with several modules. `./test-workspace.sh` builds, vets and tests every module except `tools/`, and runs gosec on the same modules except `examples/`. [AGENTS.md](AGENTS.md) describes the module layout.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

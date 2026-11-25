/*
Package golog provides fast and flexible structured logging for Go applications.

# Overview

golog is a high-performance structured logging library inspired by zerolog,
designed for zero-allocation logging with memory pooling, type-safe field
methods, and multiple output formats.

# Basic Usage

Create a logger and start logging:

	config := golog.NewConfig(
		&golog.DefaultLevels,
		golog.AllLevelsActive,
		golog.NewTextWriterConfig(os.Stdout, nil, nil),
	)

	log := golog.NewLogger(config)

	log.Info("Application started").Log()
	log.Error("Connection failed").Err(err).Log()

# Structured Logging

Add typed fields to log messages:

	log.Info("User login").
		Str("username", "john_doe").
		Str("ip", "192.168.1.1").
		Int("user_id", 12345).
		Bool("authenticated", true).
		Duration("login_time", time.Since(start)).
		Log()

Supported field types include:
  - Str, Strs: Strings and string slices
  - Int, Int64, Uint, Uint64: Integer types
  - Float32, Float: Floating point numbers
  - Bool: Booleans
  - Time, Duration: Time and duration values
  - UUID: UUID values
  - JSON: Raw JSON data
  - Err: Error values
  - Any: Generic values (uses reflection)

Bytes have one meaning everywhere: a string if they are valid UTF-8,
else encoded with [BinaryEncoder] (base64.RawURLEncoding by default),
as [Message.StrBytes] logs them. This applies to bytes
logged with [Message.Any] or as struct fields too. A byte slice type with
a format of its own, like net.IP, is logged as one value with that format
instead of as bytes: as error or [Loggable], else with its MarshalJSON,
MarshalText or String method. Three methods exist for
other purposes: [Message.Bytes] always logs binary data encoded with [BinaryEncoder],
[Message.JSON] logs JSON, and [Message.Uint8s] logs a []uint8 (the same Go
type as []byte) as numbers.
Only a struct or map passed as a whole to [Message.Any] is encoded by
encoding/json, which writes its bytes as standard base64 with padding.

[Message.Any] logs other values like the matching typed method:
a [time.Time] like [Message.Time], with the TimeFormat and Location of the
writer's [Format] and null for the zero time, and a [encoding/json.RawMessage]
like [Message.JSON]. The elements of a []any keep their own types.
A value that implements error is logged as its error string,
also as an element of a slice. Like encoding/json, a value with a
MarshalJSON or MarshalText method is logged with it, also with pointer
receiver for a struct passed as pointer, so a type can redact a secret.
A [16]byte is logged as UUID, and so is a named [16]byte type that holds
a UUID, unless it has such a method that formats it differently.

Logging never panics or hangs because of a logged value: a panic in a
method that golog calls on a value, like Error, String, Log or MarshalJSON,
is logged as error string, and a value that references itself, like a
[]any that contains itself or Loggables that log each other,
is logged with a cycle error instead of recursing forever.
A value that references the same values so many times that writing
it would take exponential time is logged as error, and so is a value
nested more than 256 deep through pointers, slices, arrays, maps and Loggables,
or more than 10000 deep when it is formatted by the fmt package or
marshalled by encoding/json. The limits apply to all values of a log
message together.

A type implements [Loggable] with a Log method that logs its fields,
for example to leave out personal data:

	type User struct {
		ID    int
		Name  string
		Email string
	}

	func (u User) Log(m *golog.Message) {
		m.Int("id", u.ID).Str("name", u.Name)
	}

	log.Info("Signed in").Loggable(user).Log()   // "id":7,"name":"Alice"
	log.Info("Signed in").Any("user", user).Log() // "user":{"id":7,"name":"Alice"}

[Message.Loggable] logs the fields as fields of the message,
[Message.Any] logs them as JSON object under the key, also for the elements
of a slice and behind several pointers, and as null a nil pointer,
a nil [LoggableFunc] or a Loggable whose IsNull method returns true,
for which [Message.Loggable] logs nothing.
[LoggableFunc] turns a function into a Loggable,
and [Attribs] is a Loggable as well.
A Log method declared on the pointer type *User is only called for a *User:
[Message.Any] encodes a User value with encoding/json instead,
which logs all exported fields, Email included.

# Log Levels

Six standard log levels are supported:

  - TRACE (-20): Most verbose, for execution flow tracing
  - DEBUG (-10): Debug information for development
  - INFO (0): General information messages
  - WARN (10): Warnings for potentially harmful situations
  - ERROR (20): Error conditions
  - FATAL (30): Critical errors (set [GlobalPanicLevel] to panic after logging)

Level-specific methods:

	log.Trace("Entering function").Log()
	log.Debug("Variable state").Int("counter", c).Log()
	log.Info("Operation completed").Log()
	log.Warn("Deprecated API used").Log()
	log.Error("Failed to connect").Err(err).Log()
	log.Fatal("Critical failure").Log()

# Sub-loggers

Create child loggers with inherited attributes:

	subLog := log.With().
		Str("service", "user-management").
		UUID("request_id", requestID).
		SubLogger()

	// All logs from subLog include the above attributes
	subLog.Info("User created").Str("username", "john").Log()

# Context Integration

Store and retrieve log attributes in context:

	// Add attributes to context
	ctx = golog.ContextWithAttribs(ctx,
		golog.NewString("correlation_id", corrID),
		golog.NewString("user_id", userID),
	)

	// Create logger from context
	ctxLogger := log.WithCtx(ctx)
	ctxLogger.Info("Operation started").Log() // Includes context attributes

The context gets clones of the passed attribs, so the caller keeps ownership
of them. Use [ContextWithOwnedAttribs] or [RequestWithOwnedAttribs] to hand
attribs that were built for the call over to the context instead of cloning
them:

	ctx = golog.ContextWithOwnedAttribs(ctx, golog.NewUUID("requestID", requestID))

A sub-logger created with [Message.SubLoggerContext] inherits the attribs of
the passed context and keeps logging them when used with another context.
Those inherited attribs are written after the attribs of the context that a
message is logged with, so an attrib with the same key from that context
takes precedence over the inherited one, see [Logger.CtxAttribs].
Once an attrib of the logger, of an inherited context or of the context of
the message was written, [Message.Ctx] and the field methods of the message
drop attribs with the same key. The fields of a message are not checked
against each other: a message that sets the same key twice writes it twice.

# Output Formats

## JSON Output

Structured JSON logging for machine parsing:

	config := golog.NewConfig(
		&golog.DefaultLevels,
		golog.AllLevelsActive,
		golog.NewJSONWriterConfig(os.Stdout, nil),
	)

Output (matches [NewDefaultFormat]: [Format.TimestampKey] and [Format.TimestampFormat]; instant shown as UTC):

	{"time":"2024-01-15 10:30:45.000","level":"INFO","message":"User login","username":"john_doe"}

## Text Output

Human-readable text output with optional colors:

	config := golog.NewConfig(
		&golog.DefaultLevels,
		golog.AllLevelsActive,
		golog.NewTextWriterConfig(os.Stdout, nil, log.NewStyledColorizer()), // github.com/domonda/golog/log
	)

## Message timestamp formats

[NewDefaultFormat] sets three related fields: [Format.TimestampKey] is `"time"`,
[Format.TimestampFormat] is `"2006-01-02 15:04:05.000"`, and [Format.TimeFormat] is
[DefaultTimeFormat] ([time.RFC3339Nano]).

The **log line** timestamp (the event time from context, written at the start of each
record) is formatted with [Format.TimestampFormat] in both JSON and text writers
([JSONWriter], [TextWriter]). Layouts follow Go’s reference time; see [time.Time.Format].

[Format.TimeFormat] applies only to structured [time.Time] attributes on messages (for
example fields added with [Message.Time]), not to that log-line timestamp.

Set [Format.Location] to a [*time.Location] (e.g. [time.UTC] or one returned by
[time.LoadLocation]) to render every formatted time value, both the log line timestamp
and structured [time.Time] attributes, in that timezone via [time.Time.In]. When
nil, times are formatted in their original location.

To parse timestamp strings from logs or databases, use [ParseTimestamp] or [Timestamp]
(JSON and [sql.Scanner]); they try each layout in [TimestampFormats] in order.

## Multiple Writers

Log to multiple outputs simultaneously:

	config := golog.NewConfig(
		&golog.DefaultLevels,
		golog.AllLevelsActive,
		golog.NewTextWriterConfig(os.Stdout, nil, colorizer),
		golog.NewJSONWriterConfig(logFile, nil),
	)

# Level Filtering

Control which log levels are written:

	// Only log WARN and above to file
	filter := golog.DefaultLevels.Warn.FilterOutBelow()
	fileWriter := golog.NewJSONWriterConfig(logFile, nil, filter)

	// Combine filters: a level filtered out by any of them is not logged
	filter := golog.JoinLevelFilters(
		golog.DefaultLevels.Trace.FilterOut(),
		golog.DefaultLevels.Debug.FilterOut(),
	)

# HTTP Middleware

Built-in HTTP request logging with a request ID taken from the
X-Request-ID header (or X-Correlation-ID when it is missing), or generated:

	handler := golog.HTTPMiddlewareHandler(yourHandler, log, golog.DefaultLevels.Info, "HTTP request")

Or create custom middleware:

	func middleware(log *golog.Logger) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				log.Info("HTTP request").Request(r).Log()
				next.ServeHTTP(w, r)
			})
		}
	}

# Performance

golog is designed for high performance:

  - Zero-allocation logging in most cases
  - Memory pooling for message and writer instances
  - Efficient JSON encoding
  - Minimal overhead when log levels are inactive

In the comparative benchmarks zerolog is faster at JSON output, golog is
faster at text output, and golog allocates nothing for either.

# Advanced Features

## Call Stack Logging

Log call stacks for debugging:

	log.Error("Panic recovered").
		CallStack("stack").
		Any("panic_value", v).
		Log()

## Custom Colorizers

Implement the [Colorizer] interface for custom output styling.
It has one method per element of a log line, for example
ColorizeMsg, ColorizeLevel, ColorizeKey and ColorizeError.

Built-in colorizers:
  - [StyledColorizer]: configurable termenv styles per element;
    log.NewStyledColorizer from github.com/domonda/golog/log
    returns one with default colors
  - [NoColorizer]: plain text

## Nil-Safe Logger

The level methods of a nil Logger return a nil Message,
and logging a nil Message does nothing:

	var log *golog.Logger
	log.Info("This is safe").Log() // No panic, no output

## Package Logger

Convenient package-level logger for libraries:

	import "github.com/domonda/golog/log"

	var log = log.NewPackageLogger()

	func MyFunction() {
		log.Info("Function called").Log()
	}

# Error Handling

The package provides an error handler for internal errors:

	golog.SetErrorHandler(func(err error) {
		// Handle logging errors
	})

# Best Practices

1. Create one logger per application/service
2. Use sub-loggers for components with common attributes
3. Always call Log() to emit the message
4. Use context integration for request-scoped attributes
5. Configure appropriate log levels for production
6. Use JSON format for production, text for development
7. Set up log rotation for file outputs
8. Use level filtering to reduce I/O overhead

# Thread Safety

A Logger is safe for concurrent use. Multiple goroutines can log with the
same Logger, derive sub-loggers from it and share contexts with attribs
without external synchronization, because loggers, configs, writer configs
and the attribs of contexts are not changed after they were created.
The guarantee depends on these rules:

  - A Message belongs to the goroutine that created it.
    Don't use it after Log, LogAndPanic or SubLogger.
  - The io.Writer passed to [NewJSONWriterConfig] or [NewTextWriterConfig]
    gets one Write call per message, concurrently from all logging goroutines.
    An *os.File and a logfile.RotatingWriter are safe for that;
    guard other writers, like a bytes.Buffer, with a mutex.
  - The callback of [NewCallbackWriterConfig] is called concurrently.
  - [Logger.RemoveAttribs] returns the attribs of the logger to the mempool,
    so call it only when no other goroutine uses the logger.
  - Package variables like [GlobalPanicLevel], [FilterHTTPHeaders],
    [TrimCallStackPathPrefix] and the variables of the log package
    are read without synchronization: set them before logging starts.
    Change the parent, level filter or writer configs of a logger at runtime
    with a [DynDerivedConfig]. [SetErrorHandler] is safe to call at any time.

# Compatibility

golog provides compatibility with Go's standard log/slog package
through the goslog subpackage for gradual migration.
*/
package golog

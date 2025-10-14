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
  - Float32, Float64: Floating point numbers
  - Bool: Booleans
  - Time, Duration: Time and duration values
  - UUID: UUID values
  - JSON, RawJSON: JSON data
  - Err: Error values
  - Any: Generic values (uses reflection)

# Log Levels

Six standard log levels are supported:

  - TRACE (-20): Most verbose, for execution flow tracing
  - DEBUG (-10): Debug information for development
  - INFO (0): General information messages
  - WARN (10): Warnings for potentially harmful situations
  - ERROR (20): Error conditions
  - FATAL (30): Critical errors (calls panic after logging)

Level-specific methods:

	log.Trace("Entering function").Log()
	log.Debug("Variable state").Int("counter", c).Log()
	log.Info("Operation completed").Log()
	log.Warn("Deprecated API used").Log()
	log.Error("Failed to connect").Err(err).Log()
	log.Fatal("Critical failure").Log() // Panics after logging

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
		golog.Str("correlation_id", corrID),
		golog.Str("user_id", userID),
	)

	// Create logger from context
	ctxLogger := log.WithCtx(ctx)
	ctxLogger.Info("Operation started").Log() // Includes context attributes

# Output Formats

## JSON Output

Structured JSON logging for machine parsing:

	config := golog.NewConfig(
		&golog.DefaultLevels,
		golog.AllLevelsActive,
		golog.NewJSONWriterConfig(os.Stdout, nil),
	)

Output:

	{"timestamp":"2024-01-15T10:30:45Z","level":"INFO","message":"User login","username":"john_doe"}

## Text Output

Human-readable text output with optional colors:

	config := golog.NewConfig(
		&golog.DefaultLevels,
		golog.AllLevelsActive,
		golog.NewTextWriterConfig(os.Stdout, nil, golog.NewStyledColorizer()),
	)

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
	filter := golog.LevelFilterFrom(golog.DefaultLevels.Warn)
	fileWriter := golog.NewJSONWriterConfig(logFile, nil, filter)

	// Multiple filters with custom logic
	filter := golog.JoinLevelFilters(
		golog.LevelFilterFrom(golog.DefaultLevels.Debug),
		golog.NewLevelFilterFunction(func(ctx context.Context, level golog.Level) bool {
			return customLogic(ctx, level)
		}),
	)

# HTTP Middleware

Built-in HTTP request/response logging:

	handler := golog.HTTPMiddleware(log, yourHandler)

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
  - Lazy evaluation of expensive operations
  - Minimal overhead when log levels are inactive

Benchmarks show golog performs comparably to zerolog with additional
flexibility and features.

# Advanced Features

## Call Stack Logging

Log call stacks for debugging:

	log.Error("Panic recovered").
		CallStack("stack").
		Any("panic_value", v).
		Log()

## Custom Colorizers

Implement the Colorizer interface for custom output styling:

	type Colorizer interface {
		Colorize(level Level, text string) string
	}

Built-in colorizers:
  - StyledColorizer: Colorful, styled output
  - Custom implementations for specific needs

## Nil-Safe Logger

Logger is nil-safe and will not panic:

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

Logger is safe for concurrent use. Multiple goroutines can log
simultaneously without external synchronization.

# Compatibility

golog provides compatibility with Go's standard log/slog package
through the goslog subpackage for gradual migration.
*/
package golog

package golog

import (
	"errors"
	"os"
)

// Example shows the minimal logging setup: build a Config, create a Logger,
// and emit messages with the level methods. Passing a nil Format uses
// NewDefaultFormat, which prefixes each line with a timestamp; this example
// sets a Format without a TimestampKey so the output is reproducible.
func Example() {
	config := NewConfig(
		&DefaultLevels,
		AllLevelsActive,
		NewJSONWriterConfig(os.Stdout, &Format{LevelKey: "level", MessageKey: "message"}),
	)
	log := NewLogger(config)

	log.Info("Application started").Log()
	log.Error("Connection failed").Err(errors.New("timeout")).Log()

	// Output:
	// {"level":"INFO","message":"Application started"}
	// {"level":"ERROR","message":"Connection failed","error":"timeout"}
}

// ExampleLogger demonstrates structured logging with typed field methods and a
// sub-logger whose attributes are inherited by every message it emits. The
// Format omits the TimestampKey so the output is reproducible.
func ExampleLogger() {
	format := &Format{LevelKey: "level", MessageKey: "message"}
	log := NewLogger(NewConfig(&DefaultLevels, AllLevelsActive, NewJSONWriterConfig(os.Stdout, format)))

	// Typed field methods avoid reflection and stay zero-allocation.
	log.Info("user login").
		Str("user", "john_doe").
		Int("attempt", 2).
		Bool("admin", false).
		Log()

	// A sub-logger inherits attributes for every message it emits.
	svc := log.With().Str("service", "auth").SubLogger()
	svc.Warn("token expiring soon").Log()

	// Output:
	// {"level":"INFO","message":"user login","user":"john_doe","attempt":2,"admin":false}
	// {"level":"WARN","message":"token expiring soon","service":"auth"}
}

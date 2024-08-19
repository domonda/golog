package golog

import (
	"context"
	"os"
	"time"
)

func ExampleJSONWriter() {
	format := &Format{
		TimestampFormat: "2006-01-02 15:04:05",
		TimestampKey:    "time",
		LevelKey:        "level",
		MessageKey:      "message",
	}
	formatter := NewJSONWriter(os.Stdout, format)
	config := NewConfig(&DefaultLevels, AllLevelsActive, formatter)
	log := NewLogger(config)

	// Use fixed time for reproducable example output
	at, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")

	log.NewMessageAt(context.Background(), at, config.InfoLevel(), "My log message").
		Int("int", 66).
		Str("str", "Hello\tWorld!\n").
		Log()

	log.NewMessageAt(context.Background(), at, config.ErrorLevel(), "This is an error").Log()

	// Output:
	// {"time":"2006-01-02 15:04:05","level":"INFO","message":"My log message","int":66,"str":"Hello\tWorld!\n"},
	// {"time":"2006-01-02 15:04:05","level":"ERROR","message":"This is an error"},
}

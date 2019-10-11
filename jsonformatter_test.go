package golog

import (
	"os"
	"time"
)

func ExampleJSONFormatter() {
	format := &Format{
		TimestampFormat: "2006-01-02 15:04:05",
		TimestampKey:    "time",
		LevelKey:        "level",
		MessageKey:      "message",
	}
	formatter := NewJSONFormatter(os.Stdout, format)
	config := NewConfig(DefaultLevels, AllLevels, formatter)
	log := NewLogger(config)

	// Use fixed time for reproducable example output
	at, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")

	log.NewMessageAt(at, config.Info(), "My log message").
		Int("int", 66).
		Str("str", "Hello\tWorld!\n").
		Log()

	log.NewMessageAt(at, config.Error(), "This is an error").Log()

	// Output:
	// {"time":"2006-01-02 15:04:05","level":"INFO","message":"My log message","int":66,"str":"Hello\tWorld!\n"},
	// {"time":"2006-01-02 15:04:05","level":"ERROR","message":"This is an error"},
}

package golog

import (
	"os"
	"time"
)

func ExampleJSONFormatter() {
	t, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")

	format := &Format{
		TimestampKey:    "time",
		TimestampFormat: "2006-01-02 15:04:05.999",
		LevelKey:        "level",
		Levels:          DefaultLevels,
		MessageKey:      "msg",
	}

	formatter := NewJSONFormatter(os.Stdout, format)
	log := NewLogger(LevelFilterNone, formatter)

	log.NewMessageAt(t, LevelInfo, "My log message").
		Int("int", 66).
		Str("str", "Hello\tWorld!\n").
		Log()
	log.NewMessageAt(t, LevelError, "This is an error").Log()

	// Output:
	// {"time":"2006-01-02 15:04:05","level":"INFO","msg":"My log message","int":66,"str":"Hello\tWorld!\n"}
	// {"time":"2006-01-02 15:04:05","level":"ERROR","msg":"This is an error"}
}

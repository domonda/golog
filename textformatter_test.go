package golog

import (
	"os"
	"time"
)

func ExampleTextFormatter() {
	t, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")

	format := &Format{
		TimestampFormat: "2006-01-02 15:04:05",
		Levels:          DefaultLevels,
		// MessageKey:      "msg",
	}

	formatter := NewTextFormatter(os.Stdout, format, NoColorizer)
	log := NewLogger(LevelFilterNone, formatter)

	log.NewMessageAt(t, LevelInfo, "My log message").
		Int("int", 66).
		Str("str", "Hello\tWorld!\n").
		Log()
	log.NewMessageAt(t, LevelError, "This is an error").Log()

	// Output:
	// 2006-01-02 15:04:05 |INFO | My log message int=66 str="Hello\tWorld!\n"
	// 2006-01-02 15:04:05 |ERROR| This is an error
}

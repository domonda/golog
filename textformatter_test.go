package golog

import (
	"os"
	"time"
)

func ExampleFormatter() {
	newFormatter := NewTextFormatterFuncWithColorizer(InactiveColorizer)
	format := &Format{
		TimestampFormat: "2006-01-02 15:04:05",
		Levels:          DefaultLevels,
		// MessageKey:      "msg",
	}
	t, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")
	log := NewLogger(LevelFilterNone, os.Stdout, newFormatter, format)
	log.NewMessageAt(t, LevelInfo, "My log message").
		Int("int", 66).
		Str("str", "Hello\tWorld!\n").
		Log()
	// Output: 2006-01-02 15:04:05 INFO: "My log message" int=66 str="Hello\tWorld!\n"
}

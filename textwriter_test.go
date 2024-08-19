package golog

import (
	"context"
	"errors"
	"os"
	"time"
)

func ExampleTextWriter() {
	format := &Format{
		TimestampFormat: "2006-01-02 15:04:05",
		TimestampKey:    "time",
		LevelKey:        "level",
		MessageKey:      "message",
	}
	formatter := NewTextWriter(os.Stdout, format, NoColorizer)
	config := NewConfig(&DefaultLevels, AllLevelsActive, formatter)
	log := NewLogger(config)

	// Use fixed time for reproducable example output
	at, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")

	log.NewMessageAt(context.Background(), at, config.InfoLevel(), "My log message").
		Int("int", 66).
		Str("str", "Hello\tWorld!\n").
		Log()
	log.NewMessageAt(context.Background(), at, config.ErrorLevel(), "Something went wrong!").
		Err(errors.New("Multi\nLine\n\"Error\"")).
		Int("numberOfTheBeast", 666).
		Log()

	// Output:
	// 2006-01-02 15:04:05 |INFO | My log message int=66 str="Hello\tWorld!\n"
	// 2006-01-02 15:04:05 |ERROR| Something went wrong! error=`
	// Multi
	// Line
	// "Error"
	// ` numberOfTheBeast=666
}

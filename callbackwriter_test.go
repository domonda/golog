package golog

import (
	"errors"
	"fmt"
	"time"
)

func ExampleCallbackWriter() {
	config := NewConfig(
		&DefaultLevels,
		AllLevelsActive,
		NewCallbackWriterConfig(func(timestamp time.Time, level Level, prefix, text string, attribs Attribs) {
			fmt.Printf("%s|%s|%s: %s", timestamp.Format("2006-01-02 15:04:05"), DefaultLevels.Name(level), prefix, text)
			for _, attrib := range attribs {
				fmt.Printf(" %s", attrib)
			}
			fmt.Println()
		}),
	)
	log := NewLogger(config).WithPrefix("test")

	// Use fixed timestamp for reproducable example output
	timestamp, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")

	log.InfoAt(timestamp, "My log message").
		Int("int", 66).
		Str("str", "Hello\tWorld!\n").
		Log()

	log.ErrorAt(timestamp, "This is an error").
		Err(errors.New("test error")).
		Log()

	log = log.With().
		Str("subLoggerAttrib", "original").
		SubLogger()

	log.DebugAt(timestamp, "Don't overwrite subLoggerAttrib").
		Str("subLoggerAttrib", "overwritten").
		Log()

	// Output:
	// 2006-01-02 15:04:05|INFO|test: My log message Int{"int": 66} String{"str": "Hello\tWorld!\n"}
	// 2006-01-02 15:04:05|ERROR|test: This is an error Error{"error": "test error"}
	// 2006-01-02 15:04:05|DEBUG|test: Don't overwrite subLoggerAttrib String{"subLoggerAttrib": "original"}
}

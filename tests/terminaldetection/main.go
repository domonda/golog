package main

import (
	"os"

	"github.com/domonda/golog"
)

func main() {
	// Create a configuration that decides between Text and JSON
	// based on whether we're running in a terminal
	levels := &golog.DefaultLevels
	format := golog.NewDefaultFormat()

	config := golog.NewConfig(
		levels,
		levels.Debug.FilterOutBelow(),
		golog.DecideWriterConfigForTerminal(
			golog.NewTextWriterConfig(os.Stdout, format, golog.NoColorizer),
			golog.NewJSONWriterConfig(os.Stdout, format),
		),
	)

	logger := golog.NewLogger(config)

	// Log several messages with different levels
	logger.Info("This is an info message").Log()
	logger.Warn("This is a warning message").Log()
	logger.Error("This is an error message").Log()
	logger.Debug("This is a debug message").Log()
}

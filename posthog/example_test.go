package posthog

import (
	"context"
	"os"

	"github.com/domonda/golog"
)

func ExampleNewWriterConfigFromEnv() {
	// Set up PostHog API key (in real usage, this would be set in environment)
	os.Setenv("POSTHOG_API_KEY", "your-api-key-here")
	defer os.Unsetenv("POSTHOG_API_KEY")

	// Create a PostHog writer config
	config, err := NewWriterConfigFromEnv(
		golog.NewDefaultFormat(), // Use default format
		golog.AllLevelsActive,    // Use level filter that allows all levels
		"system",                 // PostHog's unique identifier for tracking events
		true,                     // Include values in message
		map[string]any{ // Extra properties to add to every log
			"service": "my-service",
			"version": "1.0.0",
		},
	)
	if err != nil {
		panic(err)
	}

	// Create a logger with PostHog writer
	logger := golog.NewLogger(golog.NewConfig(
		&golog.DefaultLevels,  // Use default levels
		golog.AllLevelsActive, // Use level filter that allows all levels
		config,
	))

	// Log some messages
	ctx := context.Background()
	logger.NewMessage(ctx, golog.DefaultLevels.Info, "User logged in").
		Str("user_id", "12345").
		Str("email", "user@example.com").
		Log()

	logger.NewMessage(ctx, golog.DefaultLevels.Error, "Database connection failed").
		Str("database", "postgres").
		Int("retry_count", 3).
		Log()
}

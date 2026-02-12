// Command testserver sends test log messages to a local OTel Collector
// via the golog/otel bridge.
//
// Usage:
//
//	docker-compose up -d
//	go run .
//	docker-compose logs otel-collector
//	docker-compose down
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	"github.com/domonda/golog"
	logotel "github.com/domonda/golog/otel"
)

func main() {
	ctx := context.Background()

	// Create OTLP HTTP exporter pointing at local collector
	exporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint("localhost:4318"),
		otlploghttp.WithInsecure(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating exporter: %v\n", err)
		os.Exit(1)
	}

	// Create SDK log provider with a simple processor for immediate export
	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewSimpleProcessor(exporter)),
	)
	defer func() {
		if err := provider.Shutdown(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Error shutting down provider: %v\n", err)
		}
	}()

	// Create golog/otel writer config
	writerConfig := logotel.NewWriterConfig(
		provider,
		golog.NewDefaultFormat(),
		golog.AllLevelsActive,
		log.String("service", "golog-otel-test"),
	)

	// Create a golog logger with the OTel writer
	logger := golog.NewLogger(golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, writerConfig))

	// Send test messages at various levels
	logger.Trace("This is a trace message").Str("key", "trace-value").Log()
	logger.Debug("This is a debug message").Int("count", 42).Log()
	logger.Info("This is an info message").Str("user", "erik").Str("action", "test").Log()
	logger.Warn("This is a warning message").Float("duration", 1.234).Log()
	logger.Error("This is an error message").Err(fmt.Errorf("something went wrong")).Log()

	// Give the exporter a moment to flush
	time.Sleep(time.Second)

	fmt.Println("Test messages sent. Check collector output with: docker-compose logs otel-collector")
}

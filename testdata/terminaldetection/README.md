# Terminal Detection Test

This directory contains a Docker-based integration test for golog's terminal detection feature.

## Overview

The test verifies that golog automatically switches between Text and JSON output formats based on whether the process is running in a terminal (TTY) or not:

- **With terminal (TTY)**: Outputs human-readable text format
- **Without terminal**: Outputs machine-readable JSON format

## Components

### 1. Test Executable (`main.go`)

A simple Go program that:
- Creates a logger using `DecideWriterConfigForTerminal()`
- Logs messages at different levels (INFO, WARN, ERROR, DEBUG)
- Exits

### 2. Dockerfile

Builds a Docker image containing:
- Go 1.24 runtime
- The golog library
- The compiled test executable

### 3. Integration Test (`../../terminaldetection_test.go`)

A Go test that:
1. Builds the Docker image
2. Runs the container **with** a terminal (`docker run -t`)
3. Runs the container **without** a terminal (`docker run`)
4. Verifies the output format in each case

## Running the Test

### Prerequisites

- Docker must be installed and running
- Go 1.24 or later

### Run the Test

From the repository root:

```bash
go test -v -run TestTerminalDetection
```

### Expected Output

#### With Terminal
```
2026-01-28 12:34:56.789 INFO: This is an info message
2026-01-28 12:34:56.790 WARN: This is a warning message
2026-01-28 12:34:56.791 ERROR: This is an error message
2026-01-28 12:34:56.792 DEBUG: This is a debug message
```

#### Without Terminal
```json
{"time":"2026-01-28 12:34:56.789","level":"INFO","message":"This is an info message"}
{"time":"2026-01-28 12:34:56.790","level":"WARN","message":"This is a warning message"}
{"time":"2026-01-28 12:34:56.791","level":"ERROR","message":"This is an error message"}
{"time":"2026-01-28 12:34:56.792","level":"DEBUG","message":"This is a debug message"}
```

## Manual Testing

You can also manually test the terminal detection:

### Build the Docker Image

```bash
docker build -t golog-terminaldetection-test -f testdata/terminaldetection/Dockerfile .
```

### Run with Terminal (Text Format)

```bash
docker run --rm -t golog-terminaldetection-test
```

### Run without Terminal (JSON Format)

```bash
docker run --rm golog-terminaldetection-test
```

## How It Works

The terminal detection uses `golang.org/x/term.IsTerminal()` to check if stdout is connected to a terminal. This is the standard way to detect TTY in Go and works reliably across different environments.

The `DecideWriterConfigForTerminal()` function in `writerconfig.go` performs this check and returns the appropriate writer configuration:

```go
func DecideWriterConfigForTerminal(terminalWriter WriterConfig, nonTerminalWriter WriterConfig) WriterConfig {
	if IsTerminal() {
		return terminalWriter
	}
	return nonTerminalWriter
}
```

## Use Cases

This feature is useful for:

1. **Development**: Human-readable logs in the terminal
2. **Production**: Machine-parseable JSON logs for log aggregation systems
3. **CI/CD**: Automatic format switching based on the environment
4. **Containerized Applications**: Proper format when logs are piped to files or log collectors

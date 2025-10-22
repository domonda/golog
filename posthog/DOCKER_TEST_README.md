# PostHog Docker Integration Test

This directory contains a comprehensive test setup for running PostHog in a local Docker container and testing the golog PostHog integration.

## Files Created

- `docker-compose.yml` - Docker Compose configuration for PostHog, PostgreSQL, and Redis
- `docker_test.go` - Comprehensive test suite including:
  - Docker container management
  - PostHog health checks
  - Basic logging functionality tests
  - Configuration validation tests
  - DistinctId validation tests
- `run-docker-test.sh` - Convenience script to run the Docker test
- Updated `example_test.go` - Updated example to match new API

## Test Features

### Docker Integration
- **Automatic container management**: Starts and stops PostHog services automatically
- **Health checks**: Waits for PostHog to be ready before running tests
- **Cleanup**: Properly cleans up containers and volumes after tests

## Test Features

### Docker Integration
- **Automatic container management**: Starts and stops PostHog services automatically
- **Health checks**: Waits for PostHog to be ready before running tests
- **Cleanup**: Properly cleans up containers and volumes after tests

## Test Features

### Docker Integration
- **Automatic container management**: Starts and stops PostHog services automatically
- **Health checks**: Waits for PostHog to be ready before running tests
- **Cleanup**: Properly cleans up containers and volumes after tests

### Log File Management
- **Configurable log files**: PostHog logs are written to configurable local file paths
- **Temporary file handling**: Tests create random temp files that are automatically cleaned up
- **Log analysis**: Tests read and analyze PostHog logs for debugging and validation
- **Automatic cleanup**: Log files are deleted using `t.Cleanup()` when tests complete

### Test Isolation
- **Environment variable isolation**: Uses `t.Setenv()` (Go 1.23+) for per-test environment variable management
- **Command-level environment**: Docker commands use environment variables only for the specific command execution
- **No global state pollution**: Each test runs with isolated environment variables
- **Automatic cleanup**: Environment variables are automatically restored after each test

### Test Coverage
- **Basic logging**: Tests different log levels (Info, Error, Debug)
- **Structured logging**: Tests complex structured data logging
- **Context control**: Tests logging disable via context
- **Configuration validation**: Tests API key validation and error handling
- **DistinctId validation**: Tests different DistinctId patterns

### Test Scenarios

1. **Info Logging**: Basic informational messages with structured data
2. **Error Logging**: Error messages with error codes and retry counts
3. **De
bug Logging**: Debug messages with detailed information
4. **Structured Logging**: Complex user action logging with multiple fields
5. **Context Without Logging**: Tests that logging can be disabled via context
6. **Configuration Validation**: Tests missing/empty API key handling
7. **DistinctId Validation**: Tests various DistinctId patterns
8. **Log File Analysis**: Tests PostHog log file creation and content analysis

## Running the Tests

### Prerequisites
- Docker and docker-compose installed
- Go 1.23+ (for `t.Setenv()` support)

### Quick Start
```bash
# Run the Docker test
./run-docker-test.sh
```

### Manual Testing
```bash
# Start PostHog services
docker-compose -f docker-compose.yml up -d

# Wait for services to be ready (about 30 seconds)
sleep 30

# Run specific tests
go test -v -run TestPostHogWithDocker
go test -v -run TestPostHogConfigValidation
go test -v -run TestDistinctIdValidation

# Stop services
docker-compose -f docker-compose.yml down -v
```

### Individual Test Functions
```bash
# Test Docker integration
go test -v -run TestPostHogWithDocker

# Test configuration validation
go test -v -run TestPostHogConfigValidation

# Test DistinctId validation
go test -v -run TestDistinctIdValidation

# Run all tests
go test -v
```

## Environment Variable Management

The tests use Go's `t.Setenv()` method (available in Go 1.23+) for better test isolation:

### Benefits of `t.Setenv()`
- **Per-test isolation**: Environment variables are set only for the duration of each test
- **Automatic cleanup**: Variables are automatically restored to their original values after the test
- **Parallel test safety**: Tests can run in parallel without interfering with each other
- **No global state pollution**: Changes don't affect other tests or the system environment

### Usage Example
```go
func TestExample(t *testing.T) {
    // Set environment variable for this test only (Go 1.23+)
    t.Setenv("POSTHOG_API_KEY", "test-key")
    
    // Test code here...
    
    // Environment variable is automatically restored when test completes
}
```

### Command-Level Environment Variables
For Docker commands, environment variables are set only for the specific command execution, not for the entire system:

```go
func startPostHogServices(logFilePath string) error {
    cmd := exec.Command("docker-compose", "-f", "docker-compose.yml", "up", "-d")
    cmd.Dir = "."
    
    // Set environment variables only for this command
    cmd.Env = append(os.Environ(), "POSTHOG_LOG_FILE_PATH="+logFilePath)
    
    return cmd.Run()
}
```

This approach ensures:
- **No system pollution**: Environment variables don't affect the local system
- **Command isolation**: Each Docker command gets its own environment
- **Cleaner tests**: No need for manual cleanup of environment variables

## Log File Configuration

The Docker setup supports configurable log file paths:

### Environment Variables
- **`POSTHOG_LOG_FILE_PATH`**: Path to the log file (defaults to `./posthog-test.log`)
- **`LOG_LEVEL`**: PostHog log level (set to `DEBUG` for detailed logging)

### Docker Compose Configuration
```yaml
environment:
  - LOG_FILE_PATH=${POSTHOG_LOG_FILE_PATH:-/tmp/posthog.log}
volumes:
  - ${POSTHOG_LOG_FILE_PATH:-./posthog-test.log}:/tmp/posthog.log:rw
```

### Test Usage
```go
// Create a temporary log file
logFilePath := createTempLogFile(t)

// Start PostHog with the log file
startPostHogServices(logFilePath)

// Read and analyze logs
logContent := readLogFile(t, logFilePath)
```

## Test Configuration

The tests use the following configuration:
- **PostHog Endpoint**: `http://localhost:8000`
- **API Key**: `test-api-key` (for testing only)
- **DistinctId**: `test-system`
- **Default Properties**: service, version, environment
- **Log File**: Random temporary file (automatically cleaned up)

## Docker Services

The Docker Compose setup includes:
- **PostHog**: Main application on port 8000
- **PostgreSQL**: Database for PostHog data
- **Redis**: Cache and session storage

## CI/CD Considerations

The tests automatically skip if:
- Running in CI environment (`CI` environment variable is set)
- Docker is not available
- Docker Compose is not available

This ensures tests can run in environments without Docker while still providing comprehensive testing locally.

## Troubleshooting

### PostHog Not Starting
- Check Docker logs: `docker-compose logs posthog`
- Ensure ports 8000, 5432, and 6379 are available
- Wait longer for initialization (PostHog can take 60+ seconds to start)

### Test Failures
- Verify Docker services are running: `docker-compose ps`
- Check PostHog health: `curl http://localhost:8000/health`
- Review test logs for specific error messages

### Port Conflicts
If you have conflicts with the default ports, modify `docker-compose.yml`:
```yaml
ports:
  - "8001:8000"  # Use port 8001 instead of 8000
```

And update the test endpoint:
```bash
export POSTHOG_ENDPOINT="http://localhost:8001"
```

#!/bin/bash

# Script to run PostHog Docker test with configurable log file
# This script starts PostHog in Docker and runs the integration test

set -e

echo "Starting PostHog Docker test with log file support..."

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo "Error: Docker is not installed or not in PATH"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo "Error: docker-compose is not installed or not in PATH"
    exit 1
fi

# Change to the posthog directory
cd "$(dirname "$0")"

# Create a temporary log file for this run
TEMP_LOG_FILE=$(mktemp /tmp/posthog-test-XXXXXX.log)
echo "Using log file: $TEMP_LOG_FILE"

echo "Starting PostHog services..."
POSTHOG_LOG_FILE_PATH="$TEMP_LOG_FILE" docker-compose -f docker-compose.yml up -d

echo "Waiting for PostHog to be ready..."
sleep 30

echo "Running tests..."
go test -v -run TestPostHogWithDocker
go test -v -run TestPostHogLogFile

echo "Stopping PostHog services..."
docker-compose -f docker-compose.yml down -v

# Show log file content if it exists
if [ -f "$TEMP_LOG_FILE" ]; then
    echo "PostHog log file content:"
    echo "========================"
    head -50 "$TEMP_LOG_FILE"  # Show first 50 lines
    echo "========================"
    echo "Log file size: $(wc -l < "$TEMP_LOG_FILE") lines"
    echo "Log file location: $TEMP_LOG_FILE"
else
    echo "Warning: Log file was not created"
fi

# Clean up the temporary log file
rm -f "$TEMP_LOG_FILE"

echo "Test completed successfully!"

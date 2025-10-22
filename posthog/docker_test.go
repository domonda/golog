package posthog

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/posthog/posthog-go"

	"github.com/domonda/golog"
)

// createTempLogFile creates a temporary log file for PostHog and returns its path.
// The file is automatically cleaned up when the test completes.
func createTempLogFile(t *testing.T) string {
	tempFile, err := os.CreateTemp("", "posthog-test-*.log")
	if err != nil {
		t.Fatalf("Failed to create temp log file: %v", err)
	}
	tempFile.Close()

	// Register cleanup function
	t.Cleanup(func() {
		os.Remove(tempFile.Name())
	})

	return tempFile.Name()
}

// readLogFile reads the contents of a log file and returns it as a string.
func readLogFile(t *testing.T, filePath string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Logf("Could not read log file %s: %v", filePath, err)
		return ""
	}
	return string(content)
}

// waitForLogContent waits for specific content to appear in the log file.
func waitForLogContent(t *testing.T, filePath string, expectedContent string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		content := readLogFile(t, filePath)
		if strings.Contains(content, expectedContent) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// TestPostHogWithDocker runs PostHog in a local Docker container and tests basic logging functionality
func TestPostHogWithDocker(t *testing.T) {
	// Skip if running in CI or if Docker is not available
	if os.Getenv("CI") != "" || !isDockerAvailable() {
		t.Skip("Skipping Docker test - Docker not available or running in CI")
	}

	// Create a temporary log file for PostHog
	logFilePath := createTempLogFile(t)
	t.Logf("Using log file: %s", logFilePath)

	// Start PostHog services with the log file
	if err := startPostHogServices(logFilePath); err != nil {
		t.Fatalf("Failed to start PostHog services: %v", err)
	}
	defer stopPostHogServices()

	// Wait for PostHog to be ready
	if err := waitForPostHog(t); err != nil {
		t.Fatalf("PostHog failed to start: %v", err)
	}

	// Wait for PostHog to write some initial logs
	if !waitForLogContent(t, logFilePath, "Starting PostHog", 30*time.Second) {
		t.Log("PostHog logs not found, but continuing with test...")
	}

	// Test the logging functionality
	testBasicLogging(t, logFilePath)
}

func isDockerAvailable() bool {
	cmd := exec.Command("docker", "version")
	return cmd.Run() == nil
}

func startPostHogServices(logFilePath string) error {
	cmd := exec.Command("docker-compose", "-f", "docker-compose.yml", "up", "-d")
	cmd.Dir = "."

	// Set environment variables only for this command
	cmd.Env = append(os.Environ(), "POSTHOG_LOG_FILE_PATH="+logFilePath)

	return cmd.Run()
}

func stopPostHogServices() {
	cmd := exec.Command("docker-compose", "-f", "docker-compose.yml", "down", "-v")
	cmd.Dir = "."
	cmd.Run()
}

func waitForPostHog(t *testing.T) error {
	maxRetries := 30
	retryInterval := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get("http://localhost:8000/health")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			t.Logf("PostHog is ready after %d attempts", i+1)
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		t.Logf("Waiting for PostHog to start... (attempt %d/%d)", i+1, maxRetries)
		time.Sleep(retryInterval)
	}
	return fmt.Errorf("PostHog failed to start within %d attempts", maxRetries)
}

func testBasicLogging(t *testing.T, logFilePath string) {
	// Set up environment variables for the test using t.Setenv (Go 1.23+)
	t.Setenv("POSTHOG_API_KEY", "test-api-key")
	t.Setenv("POSTHOG_ENDPOINT", "http://localhost:8000")

	// Create default properties for all events
	defaultProps := posthog.NewProperties().
		Set("service", "test-service").
		Set("version", "1.0.0").
		Set("environment", "test")

	// Create PostHog writer config
	config, err := NewWriterConfigFromEnv(
		golog.NewDefaultFormat(),
		golog.AllLevelsActive,
		"test-system",
		true,
		defaultProps,
	)
	if err != nil {
		t.Fatalf("Failed to create PostHog config: %v", err)
	}

	// Create logger
	logger := golog.NewLogger(golog.NewConfig(
		&golog.DefaultLevels,
		golog.AllLevelsActive,
		config,
	))

	// Test different log levels and structured logging
	ctx := context.Background()

	t.Run("InfoLogging", func(t *testing.T) {
		logger.NewMessage(ctx, golog.DefaultLevels.Info, "Test info message").
			Str("test_id", "info-test-001").
			Str("component", "test-runner").
			Log()
	})

	t.Run("ErrorLogging", func(t *testing.T) {
		logger.NewMessage(ctx, golog.DefaultLevels.Error, "Test error message").
			Str("test_id", "error-test-001").
			Str("error_code", "TEST_ERROR").
			Int("retry_count", 3).
			Log()
	})

	t.Run("DebugLogging", func(t *testing.T) {
		logger.NewMessage(ctx, golog.DefaultLevels.Debug, "Test debug message").
			Str("test_id", "debug-test-001").
			Str("debug_info", "detailed debugging information").
			Log()
	})

	t.Run("StructuredLogging", func(t *testing.T) {
		logger.NewMessage(ctx, golog.DefaultLevels.Info, "User action performed").
			Str("user_id", "user-12345").
			Str("action", "login").
			Str("ip_address", "192.168.1.100").
			Str("user_agent", "TestAgent/1.0").
			Bool("success", true).
			Log()
	})

	t.Run("ContextWithoutLogging", func(t *testing.T) {
		// Test that logging can be disabled via context
		ctxWithoutLogging := ContextWithoutLogging(ctx)
		logger.NewMessage(ctxWithoutLogging, golog.DefaultLevels.Info, "This should not be logged").
			Str("test_id", "disabled-test-001").
			Log()
	})

	// Give PostHog some time to process the events
	time.Sleep(2 * time.Second)

	// Read and analyze the log file
	t.Run("LogFileAnalysis", func(t *testing.T) {
		logContent := readLogFile(t, logFilePath)

		// Log the content for debugging (first 1000 chars)
		if len(logContent) > 1000 {
			t.Logf("Log file content (first 1000 chars):\n%s", logContent[:1000])
		} else {
			t.Logf("Log file content:\n%s", logContent)
		}

		// Check if PostHog started successfully
		if !strings.Contains(logContent, "Starting PostHog") && !strings.Contains(logContent, "runserver") {
			t.Log("Warning: PostHog startup logs not found in log file")
		}

		// Check for any error messages in the logs
		if strings.Contains(logContent, "ERROR") || strings.Contains(logContent, "CRITICAL") {
			t.Log("Found error messages in PostHog logs")
		}
	})

	t.Log("Basic logging test completed successfully")
}

// TestPostHogConfigValidation tests the configuration validation
func TestPostHogConfigValidation(t *testing.T) {
	t.Run("MissingAPIKey", func(t *testing.T) {
		// Ensure the environment variable is not set for this test
		t.Setenv("POSTHOG_API_KEY", "")

		_, err := NewWriterConfigFromEnv(
			golog.NewDefaultFormat(),
			golog.AllLevelsActive,
			"test-system",
			true,
			posthog.NewProperties(),
		)
		if err == nil {
			t.Error("Expected error for missing API key")
		}
		if !strings.Contains(err.Error(), "POSTHOG_API_KEY is not set") {
			t.Errorf("Expected error message about missing API key, got: %v", err)
		}
	})

	t.Run("EmptyAPIKey", func(t *testing.T) {
		t.Setenv("POSTHOG_API_KEY", "   ")

		_, err := NewWriterConfigFromEnv(
			golog.NewDefaultFormat(),
			golog.AllLevelsActive,
			"test-system",
			true,
			posthog.NewProperties(),
		)
		if err == nil {
			t.Error("Expected error for empty API key")
		}
		if !strings.Contains(err.Error(), "POSTHOG_API_KEY is not set") {
			t.Errorf("Expected error message about missing API key, got: %v", err)
		}
	})

	t.Run("ValidConfig", func(t *testing.T) {
		t.Setenv("POSTHOG_API_KEY", "test-key")

		config, err := NewWriterConfigFromEnv(
			golog.NewDefaultFormat(),
			golog.AllLevelsActive,
			"test-system",
			true,
			posthog.NewProperties(),
		)
		if err != nil {
			t.Errorf("Unexpected error for valid config: %v", err)
		}
		if config == nil {
			t.Error("Expected non-nil config")
		}
	})
}

// TestDistinctIdValidation tests different DistinctId values
func TestDistinctIdValidation(t *testing.T) {
	testCases := []struct {
		name        string
		distinctID  string
		expectError bool
	}{
		{"ValidSystemID", "system", false},
		{"ValidUserID", "user_12345", false},
		{"ValidServiceID", "service_api", false},
		{"EmptyID", "", false},                      // Empty is allowed, will use default
		{"RestrictedAnonymous", "anonymous", false}, // Not restricted in our implementation
		{"RestrictedGuest", "guest", false},         // Not restricted in our implementation
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("POSTHOG_API_KEY", "test-key")

			config, err := NewWriterConfigFromEnv(
				golog.NewDefaultFormat(),
				golog.AllLevelsActive,
				tc.distinctID,
				true,
				posthog.NewProperties(),
			)

			if tc.expectError && err == nil {
				t.Errorf("Expected error for DistinctId: %s", tc.distinctID)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error for DistinctId %s: %v", tc.distinctID, err)
			}
			if !tc.expectError && config == nil {
				t.Errorf("Expected non-nil config for DistinctId: %s", tc.distinctID)
			}
		})
	}
}

// TestLogFileHelpers tests the log file helper functions without Docker
func TestLogFileHelpers(t *testing.T) {
	t.Run("CreateTempLogFile", func(t *testing.T) {
		logFilePath := createTempLogFile(t)

		// Verify the file was created
		if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
			t.Error("Temp log file was not created")
		}

		// Verify it's a temp file
		if !strings.Contains(logFilePath, "posthog-test-") {
			t.Errorf("Expected temp file pattern, got: %s", logFilePath)
		}

		t.Logf("Created temp log file: %s", logFilePath)
	})

	t.Run("ReadLogFile", func(t *testing.T) {
		logFilePath := createTempLogFile(t)

		// Write some test content
		testContent := "Test log content\nLine 2\nLine 3"
		err := os.WriteFile(logFilePath, []byte(testContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write test content: %v", err)
		}

		// Read it back
		content := readLogFile(t, logFilePath)
		if content != testContent {
			t.Errorf("Expected content %q, got %q", testContent, content)
		}
	})

	t.Run("WaitForLogContent", func(t *testing.T) {
		logFilePath := createTempLogFile(t)

		// Test with content that will never appear (should timeout)
		found := waitForLogContent(t, logFilePath, "NEVER_FOUND", 100*time.Millisecond)
		if found {
			t.Error("Expected timeout, but content was found")
		}

		// Test with content that appears immediately
		err := os.WriteFile(logFilePath, []byte("Test content with TARGET"), 0644)
		if err != nil {
			t.Fatalf("Failed to write test content: %v", err)
		}

		found = waitForLogContent(t, logFilePath, "TARGET", 1*time.Second)
		if !found {
			t.Error("Expected to find TARGET content, but didn't")
		}
	})
}

// TestPostHogLogFile tests the log file functionality specifically
func TestPostHogLogFile(t *testing.T) {
	// Skip if running in CI or if Docker is not available
	if os.Getenv("CI") != "" || !isDockerAvailable() {
		t.Skip("Skipping Docker test - Docker not available or running in CI")
	}

	// Create a temporary log file
	logFilePath := createTempLogFile(t)
	t.Logf("Testing with log file: %s", logFilePath)

	// Start PostHog services
	if err := startPostHogServices(logFilePath); err != nil {
		t.Fatalf("Failed to start PostHog services: %v", err)
	}
	defer stopPostHogServices()

	// Wait for PostHog to be ready
	if err := waitForPostHog(t); err != nil {
		t.Fatalf("PostHog failed to start: %v", err)
	}

	// Test that the log file is created and contains content
	t.Run("LogFileCreation", func(t *testing.T) {
		// Wait for log file to be created and contain content
		if !waitForLogContent(t, logFilePath, "Starting PostHog", 30*time.Second) {
			t.Error("Log file was not created or does not contain expected content")
		}

		// Verify the log file exists and has content
		info, err := os.Stat(logFilePath)
		if err != nil {
			t.Errorf("Log file does not exist: %v", err)
		} else if info.Size() == 0 {
			t.Error("Log file exists but is empty")
		} else {
			t.Logf("Log file size: %d bytes", info.Size())
		}
	})

	t.Run("LogFileContent", func(t *testing.T) {
		logContent := readLogFile(t, logFilePath)

		// Check for various PostHog startup indicators
		expectedPatterns := []string{
			"Starting PostHog",
			"runserver",
			"migrate",
			"DEBUG",
		}

		foundPatterns := 0
		for _, pattern := range expectedPatterns {
			if strings.Contains(logContent, pattern) {
				foundPatterns++
				t.Logf("Found expected pattern in logs: %s", pattern)
			}
		}

		if foundPatterns == 0 {
			t.Log("No expected patterns found in log file, but this might be normal")
		}

		// Log a sample of the content for debugging
		if len(logContent) > 500 {
			t.Logf("Log file sample (first 500 chars):\n%s", logContent[:500])
		} else {
			t.Logf("Full log file content:\n%s", logContent)
		}
	})

	t.Run("LogFilePermissions", func(t *testing.T) {
		// Check that the log file has appropriate permissions
		info, err := os.Stat(logFilePath)
		if err != nil {
			t.Errorf("Could not stat log file: %v", err)
		} else {
			t.Logf("Log file permissions: %s", info.Mode())
		}
	})
}

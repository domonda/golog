package golog_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTerminalDetection tests that the logger automatically switches between
// Text and JSON format based on whether it's running in a terminal or not.
// This test uses a Docker container to simulate both scenarios.
func TestTerminalDetection(t *testing.T) {
	// Check if Docker is available
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping terminal detection test")
	}

	const (
		imageName      = "golog-terminaldetection-test"
		dockerfilePath = "./testdata/terminaldetection"
	)

	// Build the Docker image
	t.Log("Building Docker image...")
	buildCmd := exec.Command("docker", "build", "-t", imageName, "-f", dockerfilePath+"/Dockerfile", ".")
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build Docker image: %v\nOutput: %s", err, buildOutput)
	}

	// Clean up the image after the test
	defer func() {
		t.Log("Cleaning up Docker image...")
		cleanupCmd := exec.Command("docker", "rmi", "-f", imageName)
		_ = cleanupCmd.Run() // Ignore errors during cleanup
	}()

	t.Run("with terminal produces text format", func(t *testing.T) {
		// Run with -t flag to allocate a pseudo-TTY
		cmd := exec.Command("docker", "run", "--rm", "-t", imageName)
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stdout

		err := cmd.Run()
		require.NoError(t, err, "Failed to run Docker container with terminal")

		output := stdout.String()
		t.Logf("Output with terminal:\n%s", output)

		// Verify it's text format (not JSON)
		// Text format should contain readable timestamps and log levels
		// and should NOT be valid JSON
		lines := strings.Split(strings.TrimSpace(output), "\n")
		require.NotEmpty(t, lines, "Expected output")

		for _, line := range lines {
			// Each line should NOT be valid JSON (text format)
			var jsonData map[string]interface{}
			err := json.Unmarshal([]byte(line), &jsonData)
			assert.Error(t, err, "Expected text format, but line is valid JSON: %s", line)

			// Text format should contain recognizable patterns
			// Look for timestamp-like patterns and log level indicators
			hasTimestamp := strings.Contains(line, ":") &&
				(strings.Contains(line, "INFO") ||
				 strings.Contains(line, "WARN") ||
				 strings.Contains(line, "ERROR") ||
				 strings.Contains(line, "DEBUG"))
			assert.True(t, hasTimestamp, "Expected text format with timestamp and level, got: %s", line)
		}
	})

	t.Run("without terminal produces JSON format", func(t *testing.T) {
		// Run without -t flag (no pseudo-TTY)
		cmd := exec.Command("docker", "run", "--rm", imageName)
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stdout

		err := cmd.Run()
		require.NoError(t, err, "Failed to run Docker container without terminal")

		output := stdout.String()
		t.Logf("Output without terminal:\n%s", output)

		// Verify it's JSON format
		lines := strings.Split(strings.TrimSpace(output), "\n")
		require.NotEmpty(t, lines, "Expected output")

		// Expected log levels in order
		expectedLevels := []string{"INFO", "WARN", "ERROR", "DEBUG"}
		levelIndex := 0

		for i, line := range lines {
			if line == "" {
				continue
			}

			// Each line should be valid JSON
			var jsonData map[string]interface{}
			err := json.Unmarshal([]byte(line), &jsonData)
			require.NoError(t, err, "Expected JSON format on line %d, got: %s", i+1, line)

			// Verify JSON structure contains expected fields
			assert.Contains(t, jsonData, "time", "JSON should contain 'time' field")
			assert.Contains(t, jsonData, "level", "JSON should contain 'level' field")
			assert.Contains(t, jsonData, "message", "JSON should contain 'message' field")

			// Verify log level matches expected order
			if levelIndex < len(expectedLevels) {
				level, ok := jsonData["level"].(string)
				require.True(t, ok, "Level should be a string")
				assert.Equal(t, expectedLevels[levelIndex], level,
					"Expected level %s on line %d", expectedLevels[levelIndex], i+1)
				levelIndex++
			}
		}

		assert.Equal(t, len(expectedLevels), levelIndex,
			"Expected %d log entries, got %d", len(expectedLevels), levelIndex)
	})
}

// isDockerAvailable checks if Docker is available on the system
func isDockerAvailable() bool {
	cmd := exec.Command("docker", "version")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run() == nil
}
